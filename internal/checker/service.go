package checker

import (
	"context"
	"crypto/tls"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/MowlCoder/heimdall/internal/domain"
	"github.com/MowlCoder/heimdall/internal/metrics"
)

type notifier interface {
	Notify(serviceErr *domain.ServiceError) error
}

type ServiceChecker struct {
	notifier notifier

	services       []domain.Service
	wg             sync.WaitGroup
	metricsBackend string
}

func NewServiceChecker(notifier notifier, services []domain.Service, metricsBackend string) *ServiceChecker {
	return &ServiceChecker{
		notifier:       notifier,
		services:       services,
		wg:             sync.WaitGroup{},
		metricsBackend: metricsBackend,
	}
}

func (sc *ServiceChecker) Start(ctx context.Context) {

	var collector metrics.MetricsCollector
	var client *http.Client

	baseTransport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	//To use any other scraper like Victoria add the implementation here.
	if sc.metricsBackend == "prometheus" {

		promCollector := metrics.NewPrometheusCollector()
		promCollector.StartServer("9092")
		collector = promCollector
		instrumentClient := http.Client{
			Transport: metrics.NewInstrumentedRoundTripper(collector, baseTransport),
			Timeout:   30 * time.Second,
		}

		client = &instrumentClient

	} else {

		client = &http.Client{
			Transport: baseTransport,
		}
	}
	for _, service := range sc.services {
		go func(wg *sync.WaitGroup) {
			if err := sc.startCheckService(ctx, service, client); err != nil {
				log.Printf("[ERROR]: failed to start check service %s: %v\n", service.Name, err)
			}
			wg.Done()
		}(&sc.wg)
		sc.wg.Add(1)
	}
}

func (sc *ServiceChecker) WaitShutdown() {
	sc.wg.Wait()
}

func (sc *ServiceChecker) startCheckService(ctx context.Context, service domain.Service, client *http.Client) error {
	checkServiceInterval, err := service.ParseInterval()
	if err != nil {
		return err
	}

	for {
		func() {
			request, err := http.NewRequest(http.MethodHead, service.URL, nil)
			if err != nil {
				log.Printf("[ERROR]: error happened when checking %s service: %v\n", service.Name, err)
				return
			}

			for hName, hValue := range service.Headers {
				request.Header.Set(hName, hValue)
			}

			request = request.WithContext(ctx)

			if service.Timeout > 0 {
				timeoutCtx, timeoutCtxDone := context.WithTimeout(request.Context(), time.Duration(service.Timeout)*time.Millisecond)
				request = request.WithContext(timeoutCtx)
				defer timeoutCtxDone()
			}

			response, err := http.DefaultClient.Do(request)
			if err != nil {
				sc.sendFailNotification(&domain.ServiceError{
					Name:       service.Name,
					StatusCode: 0,
					Body:       nil,
					Error:      err,
				})
				return
			}
			defer response.Body.Close()

			if (service.Strict && response.StatusCode != 200) || (response.StatusCode >= 500) {
				body, _ := io.ReadAll(response.Body)

				sc.sendFailNotification(&domain.ServiceError{
					Name:       service.Name,
					StatusCode: response.StatusCode,
					Error:      nil,
					Body:       body,
				})
				return
			}

			log.Printf("[SUCCESS] %s is healthy\n", service.Name)
		}()

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(checkServiceInterval):
		}
	}
}

func (sc *ServiceChecker) sendFailNotification(serviceErr *domain.ServiceError) {
	log.Printf(
		"[ERROR]: %s is not healthy, statusCode=%d,body=%s,error=%v\n",
		serviceErr.Name,
		serviceErr.StatusCode,
		serviceErr.Body,
		serviceErr.Error,
	)
	if err := sc.notifier.Notify(serviceErr); err != nil {
		log.Printf("failed to notify: %v\n", err)
	}
}
