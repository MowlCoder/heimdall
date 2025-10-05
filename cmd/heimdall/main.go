package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/MowlCoder/heimdall/internal/checker"
	"github.com/MowlCoder/heimdall/internal/config"
	"github.com/MowlCoder/heimdall/internal/notifier"
)

func main() {
	flagCfg := config.NewFlagConfig()
	if err := flagCfg.Parse(); err != nil {
		log.Fatal(err)
	}

	cfg, err := config.ParseConfigFromFile(flagCfg.PathToConfigFile)
	if err != nil {
		log.Fatal(err)
	}

	notifierManager := notifier.NewNotifierManager()

	if cfg.IsTelegramEnabled() {
		notifierManager.AddService(notifier.NewTelegramNotifier(
			cfg.Notifiers.Telegram.ChatID,
			cfg.Notifiers.Telegram.BotToken,
		))
	}

	if cfg.IsDiscordEnabled() {
		notifierManager.AddService(notifier.NewDiscordNotifier(
			cfg.Notifiers.Discord.Webhook,
		))
	}

	if cfg.IsSlackEnabled() {
		notifierManager.AddService(notifier.NewSlackNotifier(
			cfg.Notifiers.Slack.ChatID,
			cfg.Notifiers.Slack.BotToken,
		))
	}

	if cfg.IsWebhookEnabled() {
		notifierManager.AddService(notifier.NewWebhookNotifier(
			cfg.Notifiers.Webhook.Webhook,
		))
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	serviceChecker := checker.NewServiceChecker(notifierManager, cfg.Services, cfg.MetricsBackend)
	serviceChecker.Start(ctx)

	log.Println("[INFO]: heimdall service status checker was started")

	<-ctx.Done()
	stop()

	log.Println("[INFO]: starting shutdown of status checker...")

	serviceChecker.WaitShutdown()
}
