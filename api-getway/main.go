package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"github.com/RubachokBoss/plagiarism-checker/api-gateway/internal/app"
	"github.com/RubachokBoss/plagiarism-checker/api-gateway/internal/config"
	"github.com/RubachokBoss/plagiarism-checker/api-gateway/pkg/logger"
)

func main() {
	log := logger.New()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	application, err := app.New(cfg, log)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create application")
	}

	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	go func() {
		if err := application.Run(); err != nil {
			log.Fatal().Err(err).Msg("Failed to run application")
		}
	}()

	log.Info().Msgf("API Gateway started on %s", cfg.Server.Address)

	<-ctx.Done()
	log.Info().Msg("Shutting down API Gateway...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := application.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Failed to shutdown gracefully")
	}

	log.Info().Msg("API Gateway stopped")
}
