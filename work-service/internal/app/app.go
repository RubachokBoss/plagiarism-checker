package app

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/RubachokBoss/plagiarism-checker/work-service/internal/config"
	"github.com/RubachokBoss/plagiarism-checker/work-service/internal/delivery/httpd"
	"github.com/RubachokBoss/plagiarism-checker/work-service/internal/repository"
	"github.com/RubachokBoss/plagiarism-checker/work-service/internal/service"
	"github.com/RubachokBoss/plagiarism-checker/work-service/internal/service/integration"
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
	rabbitmqClient integration.RabbitMQClient
}

func New(cfg *config.Config, log zerolog.Logger, db *sql.DB) (*App, error) {
	// Создаем интеграционные клиенты
	fileClient := integration.NewFileClient(
		cfg.Services.File.URL,
		cfg.Services.File.UploadEndpoint,
		cfg.Services.File.Timeout,
		cfg.Services.File.RetryCount,
		cfg.Services.File.RetryDelay,
		log,
	)

	analysisClient := integration.NewAnalysisClient(
		cfg.Services.Analysis.URL,
		cfg.Services.Analysis.ReportsEndpoint,
		cfg.Services.Analysis.Timeout,
		cfg.Services.Analysis.RetryCount,
		cfg.Services.Analysis.RetryDelay,
		log,
	)

	rabbitmqClient, err := integration.NewRabbitMQClient(
		cfg.RabbitMQ.URL,
		cfg.RabbitMQ.Exchange,
		cfg.RabbitMQ.RoutingKey,
		cfg.RabbitMQ.QueueName,
		log,
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create RabbitMQ client")
		// Продолжаем без RabbitMQ, это допустимо для разработки
	}

	// Создаем репозитории
	workRepo := repository.NewWorkRepository(db, log)
	assignmentRepo := repository.NewAssignmentRepository(db, log)
	studentRepo := repository.NewStudentRepository(db, log)

	// Создаем сервисы
	assignmentService := service.NewAssignmentService(assignmentRepo, log)
	studentService := service.NewStudentService(studentRepo, log)
	workService := service.NewWorkService(
		workRepo,
		studentRepo,
		assignmentRepo,
		fileClient,
		rabbitmqClient,
		log,
	)
	reportService := service.NewReportService(
		workRepo,
		studentRepo,
		assignmentRepo,
		analysisClient,
		log,
	)

	// Создаем обработчики
	handler := httpd.NewHandler(
		workService,
		assignmentService,
		studentService,
		reportService,
		log,
	)

	// Создаем роутер
	router := chi.NewRouter()

	// Настраиваем middleware
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(60 * time.Second))

	// Настраиваем CORS
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORS.AllowedOrigins,
		AllowedMethods:   cfg.CORS.AllowedMethods,
		AllowedHeaders:   cfg.CORS.AllowedHeaders,
		ExposedHeaders:   cfg.CORS.ExposedHeaders,
		AllowCredentials: cfg.CORS.AllowCredentials,
		MaxAge:           cfg.CORS.MaxAge,
	}))

	// Регистрируем маршруты
	handler.RegisterRoutes(router)

	// Создаем HTTP сервер
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
		rabbitmqClient: rabbitmqClient,
	}, nil
}

func (a *App) Run() error {
	a.logger.Info().Msgf("Starting work service on %s", a.config.Server.Address)
	return a.server.ListenAndServe()
}

func (a *App) Shutdown(ctx context.Context) error {
	a.logger.Info().Msg("Shutting down work service...")

	// Закрываем RabbitMQ соединение
	if a.rabbitmqClient != nil {
		if err := a.rabbitmqClient.Close(); err != nil {
			a.logger.Error().Err(err).Msg("Failed to close RabbitMQ connection")
		}
	}

	// Закрываем соединение с БД
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			a.logger.Error().Err(err).Msg("Failed to close database connection")
		}
	}

	// Останавливаем сервер
	return a.server.Shutdown(ctx)
}
