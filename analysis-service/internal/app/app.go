package app

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/config"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/delivery/httpd"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/repository"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/service"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/service/analyzer"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/service/integration"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/worker"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/worker/queue"
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
	rabbitMQRepo, err := repository.NewRabbitMQRepository(cfg.RabbitMQ.URL, log)
	if err != nil {
		return nil, err
	}

	if err := rabbitMQRepo.SetupQueue(
		cfg.RabbitMQ.Exchange,
		cfg.RabbitMQ.QueueName,
		cfg.RabbitMQ.RoutingKey,
	); err != nil {
		return nil, err
	}

	rabbitMQPublisher := queue.NewRabbitMQPublisher(rabbitMQRepo.Channel(), log)
	rabbitMQConsumer := queue.NewRabbitMQConsumer(
		rabbitMQRepo.Channel(),
		cfg.RabbitMQ.QueueName,
		cfg.RabbitMQ.ConsumerTag,
		log,
	)

	reportRepo := repository.NewReportRepository(db, log)
	plagiarismRepo := repository.NewPlagiarismRepository(db, log)

	fileClient := integration.NewFileClient(
		cfg.Services.File.URL,
		cfg.Services.File.Timeout,
		cfg.Services.File.RetryCount,
		cfg.Services.File.RetryDelay,
		log,
	)

	workClient := integration.NewWorkClient(
		cfg.Services.Work.URL,
		cfg.Services.Work.Timeout,
		cfg.Services.Work.RetryCount,
		cfg.Services.Work.RetryDelay,
		fileClient,
		log,
	)

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

	workerPool := worker.NewWorkerPool(cfg.Analysis.MaxWorkers, log)

	analysisWorker := worker.NewAnalysisWorker(
		workerPool,
		rabbitMQConsumer,
		reportRepo,
		analysisService,
		log,
	)

	handler := httpd.NewHandler(
		analysisService,
		reportService,
		log,
	)

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(60 * time.Second))

	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORS.AllowedOrigins,
		AllowedMethods:   cfg.CORS.AllowedMethods,
		AllowedHeaders:   cfg.CORS.AllowedHeaders,
		ExposedHeaders:   cfg.CORS.ExposedHeaders,
		AllowCredentials: cfg.CORS.AllowCredentials,
		MaxAge:           cfg.CORS.MaxAge,
	}))

	handler.RegisterRoutes(router)

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
	ctx := context.Background()
	if err := a.analysisWorker.Start(ctx); err != nil {
		a.logger.Error().Err(err).Msg("Failed to start analysis worker")
		return err
	}

	a.logger.Info().Msgf("Starting analysis service on %s", a.config.Server.Address)
	return a.server.ListenAndServe()
}

func (a *App) Shutdown(ctx context.Context) error {
	a.logger.Info().Msg("Shutting down analysis service...")

	if err := a.analysisWorker.Stop(); err != nil {
		a.logger.Error().Err(err).Msg("Failed to stop analysis worker")
	}

	if a.rabbitMQRepo != nil {
		if err := a.rabbitMQRepo.Close(); err != nil {
			a.logger.Error().Err(err).Msg("Failed to close RabbitMQ connection")
		}
	}

	if a.db != nil {
		if err := a.db.Close(); err != nil {
			a.logger.Error().Err(err).Msg("Failed to close database connection")
		}
	}

	if err := a.server.Shutdown(ctx); err != nil {
		a.logger.Error().Err(err).Msg("Failed to shutdown HTTP server")
		return err
	}

	a.logger.Info().Msg("Analysis service stopped")
	return nil
}
