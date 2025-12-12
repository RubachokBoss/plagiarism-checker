package app

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/plagiarism-checker/file-service/internal/config"
	"github.com/plagiarism-checker/file-service/internal/database"
	"github.com/plagiarism-checker/file-service/internal/delivery/http"
	"github.com/plagiarism-checker/file-service/internal/repository"
	"github.com/plagiarism-checker/file-service/internal/service"
	"github.com/plagiarism-checker/file-service/pkg/logger"
	"github.com/rs/zerolog"
)

type App struct {
	server *http.Server
	logger zerolog.Logger
	config *config.Config
	db     *sql.DB
}

func New(cfg *config.Config, log zerolog.Logger, db *sql.DB) (*App, error) {
	// Создаем репозиторий MinIO
	minioRepo, err := repository.NewMinIORepository(
		cfg.MinIO.Endpoint,
		cfg.MinIO.AccessKey,
		cfg.MinIO.SecretKey,
		cfg.Storage.BucketName,
		cfg.Storage.Region,
		cfg.MinIO.UseSSL,
		log,
	)
	if err != nil {
		return nil, err
	}

	// Создаем оберточный репозиторий хранилища
	storageRepo := repository.NewStorageRepository(minioRepo, log)

	// Создаем репозиторий метаданных
	metadataRepo := repository.NewFileMetadataRepository(db, log)

	// Создаем сервисы
	hashService := service.NewHashService(cfg.Hash.Algorithm)

	uploadService := service.NewUploadService(
		metadataRepo,
		storageRepo,
		hashService,
		log,
		service.UploadConfig{
			MaxUploadSize:  cfg.Server.MaxUploadSize,
			BucketName:     cfg.Storage.BucketName,
			AllowedTypes:   []string{".txt", ".pdf", ".doc", ".docx", ".zip", ".rar"},
			GenerateHash:   true,
			CheckDuplicate: true,
		},
	)

	downloadService := service.NewDownloadService(
		metadataRepo,
		storageRepo,
		log,
		cfg.Storage.BucketName,
	)

	deleteService := service.NewDeleteService(
		metadataRepo,
		storageRepo,
		log,
		cfg.Storage.BucketName,
	)

	// Создаем обработчики
	handler := http.NewHandler(
		uploadService,
		downloadService,
		deleteService,
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
		server: server,
		logger: log,
		config: cfg,
		db:     db,
	}, nil
}

func (a *App) Run() error {
	a.logger.Info().Msgf("Starting file service on %s", a.config.Server.Address)
	return a.server.ListenAndServe()
}

func (a *App) Shutdown(ctx context.Context) error {
	a.logger.Info().Msg("Shutting down file service...")

	// Закрываем соединение с БД
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			a.logger.Error().Err(err).Msg("Failed to close database connection")
		}
	}

	// Останавливаем сервер
	return a.server.Shutdown(ctx)
}
