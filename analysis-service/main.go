package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/app"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/config"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/database"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/repository"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/service"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/service/analyzer"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/service/integration"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/worker"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/worker/queue"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/pkg/logger"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "migrate":
			// Legacy style:
			//   analysis-service migrate up
			//   analysis-service migrate down
			//   analysis-service migrate force <version>
			sub := "up"
			if len(os.Args) > 2 {
				sub = os.Args[2]
			}
			switch sub {
			case "up", "down":
				runMigrations(sub)
			case "force":
				if len(os.Args) < 4 {
					fmt.Fprintln(os.Stderr, "usage: analysis-service migrate force <version>")
					os.Exit(2)
				}
				v, err := strconv.Atoi(os.Args[3])
				if err != nil {
					fmt.Fprintln(os.Stderr, "invalid version:", os.Args[3])
					os.Exit(2)
				}
				runMigrationsForce(v)
			default:
				fmt.Fprintln(os.Stderr, "unknown migrate subcommand:", sub)
				os.Exit(2)
			}
			return
		case "worker":
			runWorker()
			return
		}
	}
	// Парсинг аргументов командной строки
	migrateCmd := flag.NewFlagSet("migrate", flag.ExitOnError)
	migrateDirection := migrateCmd.String("direction", "up", "direction of migration (up/down)")
	migrateForce := migrateCmd.Int("force", -1, "force version (dangerous). Example: -force 1")

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "migrate":
			migrateCmd.Parse(os.Args[2:])
			if *migrateForce >= 0 {
				runMigrationsForce(*migrateForce)
				return
			}
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

func runMigrationsForce(version int) {
	log := logger.New()
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	migrator := database.NewMigrator(cfg.Database)
	if err := migrator.Force(version); err != nil {
		log.Fatal().Err(err).Msg("Failed to force migration version")
	}

	log.Info().Int("version", version).Msg("Migration version forced successfully")
}

func runWorker() {
	log := logger.New()
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Init database
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

	// Init RabbitMQ
	rabbitMQRepo, err := repository.NewRabbitMQRepository(cfg.RabbitMQ.URL, log)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to RabbitMQ")
	}
	defer rabbitMQRepo.Close()

	if err := rabbitMQRepo.SetupQueue(
		cfg.RabbitMQ.Exchange,
		cfg.RabbitMQ.QueueName,
		cfg.RabbitMQ.RoutingKey,
	); err != nil {
		log.Fatal().Err(err).Msg("Failed to setup RabbitMQ queue")
	}

	// Create publishers/consumers
	rabbitMQPublisher := queue.NewRabbitMQPublisher(rabbitMQRepo.Channel(), log)
	rabbitMQConsumer := queue.NewRabbitMQConsumer(
		rabbitMQRepo.Channel(),
		cfg.RabbitMQ.QueueName,
		cfg.RabbitMQ.ConsumerTag,
		log,
	)

	// Repositories
	reportRepo := repository.NewReportRepository(db, log)
	plagiarismRepo := repository.NewPlagiarismRepository(db, log)

	// Integration clients
	workClient := integration.NewWorkClient(
		cfg.Services.Work.URL,
		cfg.Services.Work.Timeout,
		cfg.Services.Work.RetryCount,
		cfg.Services.Work.RetryDelay,
		log,
	)

	fileClient := integration.NewFileClient(
		cfg.Services.File.URL,
		cfg.Services.File.Timeout,
		cfg.Services.File.RetryCount,
		cfg.Services.File.RetryDelay,
		log,
	)

	// Analyzers
	hashComparator := analyzer.NewHashComparator(cfg.Analysis.HashAlgorithm)
	plagiarismChecker := analyzer.NewPlagiarismChecker(
		workClient,
		fileClient,
		hashComparator,
		log,
		analyzer.PlagiarismCheckerConfig{
			HashAlgorithm:       cfg.Analysis.HashAlgorithm,
			SimilarityThreshold: cfg.Analysis.SimilarityThreshold,
			EnableDeepAnalysis:  cfg.Analysis.EnableContentAnalysis,
			Timeout:             cfg.Analysis.Timeout,
			MaxRetries:          cfg.Services.Work.RetryCount,
		},
	)

	messageHandler := queue.NewMessageHandler(log)

	// Services
	analysisService := service.NewAnalysisService(
		reportRepo,
		plagiarismRepo,
		workClient,
		fileClient,
		plagiarismChecker,
		messageHandler,
		rabbitMQPublisher,
		log,
		service.AnalysisConfig{
			HashAlgorithm:       cfg.Analysis.HashAlgorithm,
			SimilarityThreshold: cfg.Analysis.SimilarityThreshold,
			EnableDeepAnalysis:  cfg.Analysis.EnableContentAnalysis,
			Timeout:             cfg.Analysis.Timeout,
			MaxRetries:          cfg.Services.Work.RetryCount,
			BatchSize:           cfg.Analysis.BatchSize,
		},
	)

	// Worker
	workerPool := worker.NewWorkerPool(cfg.Analysis.MaxWorkers, log)
	analysisWorker := worker.NewAnalysisWorker(
		workerPool,
		rabbitMQConsumer,
		reportRepo,
		analysisService,
		log,
	)

	ctxRun, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	log.Info().
		Str("rabbitmq_url", cfg.RabbitMQ.URL).
		Str("queue", cfg.RabbitMQ.QueueName).
		Msg("Starting standalone analysis worker")

	if err := analysisWorker.Start(ctxRun); err != nil {
		log.Fatal().Err(err).Msg("Failed to start analysis worker")
	}

	<-ctxRun.Done()
	log.Info().Msg("Shutting down standalone worker...")

	if err := analysisWorker.Stop(); err != nil {
		log.Error().Err(err).Msg("Failed to stop analysis worker gracefully")
	}
}
