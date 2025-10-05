package metrics

import (
	"net/http"
	"strconv"
	"time"
)

type InstrumentedRoundTripper struct {
	transport http.RoundTripper
	collector MetricsCollector
}

func NewInstrumentedRoundTripper(collector MetricsCollector, transport http.RoundTripper) *InstrumentedRoundTripper {

	if transport == nil {
		transport = http.DefaultTransport
	}

	return &InstrumentedRoundTripper{

		transport: transport,
		collector: collector,
	}

}

func (irt *InstrumentedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {

	start := time.Now()

	resp, err := irt.transport.RoundTrip(req)

	duration := time.Since(start)

	method := req.Method
	code := ""

	if err != nil {
		code = "client_error"
	} else {

		code = strconv.Itoa(resp.StatusCode)
	}

	irt.collector.ObserveRequestDuration(duration, MetricLabels{

		"method": method,
	})

	irt.collector.IncRequestsTotal(MetricLabels{
		"code":   code,
		"method": method,
	})

	return resp, err

}
