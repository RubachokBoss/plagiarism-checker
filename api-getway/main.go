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
	// Инициализация логгера
	log := logger.New()

	// Загрузка конфигурации
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Создание приложения
	application, err := app.New(cfg, log)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create application")
	}

	// Контекст для graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	// Запуск сервера в горутине
	go func() {
		if err := application.Run(); err != nil {
			log.Fatal().Err(err).Msg("Failed to run application")
		}
	}()

	log.Info().Msgf("API Gateway started on %s", cfg.Server.Address)

	// Ожидание сигнала завершения
	<-ctx.Done()
	log.Info().Msg("Shutting down API Gateway...")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := application.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Failed to shutdown gracefully")
	}

	log.Info().Msg("API Gateway stopped")
}
