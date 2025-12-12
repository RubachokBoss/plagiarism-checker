package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/app"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/config"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/database"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/pkg/logger"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "migrate":
			var direction string
			if len(os.Args) > 2 {
				direction = os.Args[2]
			} else {
				direction = "up"
			}
			runMigrations(direction)
			return
		case "worker":
			runWorker()
			return
		}
	}
	// Парсинг аргументов командной строки
	migrateCmd := flag.NewFlagSet("migrate", flag.ExitOnError)
	migrateDirection := migrateCmd.String("direction", "up", "direction of migration (up/down)")

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "migrate":
			migrateCmd.Parse(os.Args[2:])
			runMigrations(*migrateDirection)
			return
		case "worker":
			runWorker()
			return
		}
	}

	// Инициализация логгера
	log := logger.New()

	// Загрузка конфигурации
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Инициализация базы данных
	db, err := database.NewPostgres(cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	// Проверка соединения с БД
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to ping database")
	}

	log.Info().Msg("Database connection established")

	// Создание приложения
	application, err := app.New(cfg, log, db)
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

	log.Info().Msgf("Analysis Service started on %s", cfg.Server.Address)

	// Ожидание сигнала завершения
	<-ctx.Done()
	log.Info().Msg("Shutting down Analysis Service...")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := application.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Failed to shutdown gracefully")
	}

	log.Info().Msg("Analysis Service stopped")
}

func runMigrations(direction string) {
	log := logger.New()
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	migrator := database.NewMigrator(cfg.Database)

	switch direction {
	case "up":
		if err := migrator.Up(); err != nil {
			log.Fatal().Err(err).Msg("Failed to apply migrations")
		}
		log.Info().Msg("Migrations applied successfully")
	case "down":
		if err := migrator.Down(); err != nil {
			log.Fatal().Err(err).Msg("Failed to rollback migrations")
		}
		log.Info().Msg("Migrations rolled back successfully")
	default:
		log.Fatal().Msg("Invalid migration direction. Use 'up' or 'down'")
	}
}

func runWorker() {
	log := logger.New()
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	log.Info().Msg("Starting standalone worker...")
	// Здесь можно запустить только воркер без HTTP сервера
}
