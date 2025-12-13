package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/RubachokBoss/plagiarism-checker/work-service/internal/app"
	"github.com/RubachokBoss/plagiarism-checker/work-service/internal/config"
	"github.com/RubachokBoss/plagiarism-checker/work-service/internal/database"
	"github.com/RubachokBoss/plagiarism-checker/work-service/pkg/logger"
)

func main() {
	migrateCmd := flag.NewFlagSet("migrate", flag.ExitOnError)
	migrateDirection := migrateCmd.String("direction", "up", "direction of migration (up/down)")

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "migrate":
			migrateCmd.Parse(os.Args[2:])
			runMigrations(*migrateDirection)
			return
		}
	}

	log := logger.New()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	db, err := database.NewPostgres(cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to ping database")
	}

	log.Info().Msg("Database connection established")

	application, err := app.New(cfg, log, db)
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

	log.Info().Msgf("Work Service started on %s", cfg.Server.Address)

	<-ctx.Done()
	log.Info().Msg("Shutting down Work Service...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := application.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Failed to shutdown gracefully")
	}

	log.Info().Msg("Work Service stopped")
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
