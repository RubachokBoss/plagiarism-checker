package app

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/config"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/database"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/delivery/httpd"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/repository"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/service"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/service/analyzer"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/service/integration"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/worker"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/worker/queue"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/pkg/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog"
)

type App struct {
	server         *http.Server
	logger         zerolog.Logger
	config         *config.Config
	db             *sql.DB
	analysisWorker worker.AnalysisWorker
	rabbitMQRepo   repository.RabbitMQRepository
}

func New(cfg *config.Config, log zerolog.Logger, db *sql.DB) (*App, error) {
	// Create RabbitMQ repository
	rabbitMQRepo, err := repository.NewRabbitMQRepository(cfg.RabbitMQ.URL, log)
	if err != nil {
		return nil, err
	}

	// Setup RabbitMQ queue
	if err := rabbitMQRepo.SetupQueue(
		cfg.RabbitMQ.Exchange,
		cfg.RabbitMQ.QueueName,
		cfg.RabbitMQ.RoutingKey,
	); err != nil {
		return nil, err
	}

	// Create RabbitMQ publisher and consumer
	rabbitMQPublisher := queue.NewRabbitMQPublisher(rabbitMQRepo.Channel, log)
	rabbitMQConsumer := queue.NewRabbitMQConsumer(
		rabbitMQRepo.Channel,
		cfg.RabbitMQ.QueueName,
		cfg.RabbitMQ.ConsumerTag,
		log,
	)

	// Create repositories
	reportRepo := repository.NewReportRepository(db, log)
	plagiarismRepo := repository.NewPlagiarismRepository(db, log)

	// Create integration clients
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

	// Create analyzers
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

	// Create message handler
	messageHandler := queue.NewMessageHandler(log)

	// Create services
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

	reportService := service.NewReportService(
		reportRepo,
		plagiarismRepo,
		log,
	)

	// Create worker pool
	workerPool := worker.NewWorkerPool(cfg.Analysis.MaxWorkers, log)

	// Create analysis worker
	analysisWorker := worker.NewAnalysisWorker(
		workerPool,
		rabbitMQConsumer,
		reportRepo,
		analysisService,
		log,
	)

	// Create HTTP handlers
	handler := httpd.NewHandler(
		analysisService,
		reportService,
		log,
	)

	// Create router
	router := chi.NewRouter()

	// Setup middleware
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(60 * time.Second))

	// Setup CORS
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORS.AllowedOrigins,
		AllowedMethods:   cfg.CORS.AllowedMethods,
		AllowedHeaders:   cfg.CORS.AllowedHeaders,
		ExposedHeaders:   cfg.CORS.ExposedHeaders,
		AllowCredentials: cfg.CORS.AllowCredentials,
		MaxAge:           cfg.CORS.MaxAge,
	}))

	// Register routes
	handler.RegisterRoutes(router)

	// Create HTTP server
	server := &http.Server{
		Addr:         cfg.Server.Address,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	return &App{
		server:         server,
		logger:         log,
		config:         cfg,
		db:             db,
		analysisWorker: analysisWorker,
		rabbitMQRepo:   rabbitMQRepo,
	}, nil
}

func (a *App) Run() error {
	// Start analysis worker
	ctx := context.Background()
	if err := a.analysisWorker.Start(ctx); err != nil {
		a.logger.Error().Err(err).Msg("Failed to start analysis worker")
		return err
	}

	// Start HTTP server
	a.logger.Info().Msgf("Starting analysis service on %s", a.config.Server.Address)
	return a.server.ListenAndServe()
}

func (a *App) Shutdown(ctx context.Context) error {
	a.logger.Info().Msg("Shutting down analysis service...")

	// Stop analysis worker
	if err := a.analysisWorker.Stop(); err != nil {
		a.logger.Error().Err(err).Msg("Failed to stop analysis worker")
	}

	// Close RabbitMQ connection
	if a.rabbitMQRepo != nil {
		if err := a.rabbitMQRepo.Close(); err != nil {
			a.logger.Error().Err(err).Msg("Failed to close RabbitMQ connection")
		}
	}

	// Close database connection
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			a.logger.Error().Err(err).Msg("Failed to close database connection")
		}
	}

	// Shutdown HTTP server
	if err := a.server.Shutdown(ctx); err != nil {
		a.logger.Error().Err(err).Msg("Failed to shutdown HTTP server")
		return err
	}

	a.logger.Info().Msg("Analysis service stopped")
	return nil
}
