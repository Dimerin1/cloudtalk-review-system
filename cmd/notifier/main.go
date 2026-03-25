package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/dimerin1/cloudtalk-review-system/internal/config"
	"github.com/dimerin1/cloudtalk-review-system/internal/events"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	cfg := config.Load()

	consumer := events.NewConsumer(cfg.KafkaBrokers, "notifier-service", logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		logger.Info("notifier shutting down")
		cancel()
	}()

	logger.Info("notifier service starting")
	if err := consumer.Start(ctx); err != nil {
		logger.Error("consumer error", "error", err)
		os.Exit(1)
	}
	logger.Info("notifier service stopped")
}
