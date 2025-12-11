package app

import (
	"context"

	"github.com/plagiarism-checker/api-gateway/internal/config"
	"github.com/plagiarism-checker/api-gateway/internal/handler"
	"github.com/plagiarism-checker/api-gateway/internal/middleware"
	"github.com/plagiarism-checker/api-gateway/internal/server"
	"github.com/plagiarism-checker/api-gateway/pkg/logger"
	"github.com/rs/zerolog"
)

type App struct {
	server *server.Server
	logger zerolog.Logger
	config *config.Config
}

func New(cfg *config.Config, log zerolog.Logger) (*App, error) {
	// Создаем обработчик
	h := handler.NewHandler(log, handler.ProxyConfig{
		Timeout:         cfg.Proxy.Timeout,
		MaxIdleConns:    cfg.Proxy.MaxIdleConns,
		IdleConnTimeout: cfg.Proxy.IdleConnTimeout,
	})

	// Создаем прокси для сервисов
	workProxy, err := h.CreateServiceProxy(cfg.Services.Work.URL, "/api/v1/works")
	if err != nil {
		return nil, err
	}

	fileProxy, err := h.CreateServiceProxy(cfg.Services.File.URL, "/api/v1/files")
	if err != nil {
		return nil, err
	}

	analysisProxy, err := h.CreateServiceProxy(cfg.Services.Analysis.URL, "/api/v1/analysis")
	if err != nil {
		return nil, err
	}

	// Настраиваем маршруты прокси
	h.SetupProxyRoutes(workProxy, fileProxy, analysisProxy)

	// Получаем router
	router := h.GetRouter()

	// Создаем сервер
	srv := server.NewServer(server.ServerConfig{
		Address:         cfg.Server.Address,
		ReadTimeout:     cfg.Server.ReadTimeout,
		WriteTimeout:    cfg.Server.WriteTimeout,
		IdleTimeout:     cfg.Server.IdleTimeout,
		ShutdownTimeout: cfg.Server.ShutdownTimeout,
	}, router, log)

	// Настраиваем middleware
	srv.SetupMiddleware(
		middleware.NewCORS(
			cfg.CORS.AllowedOrigins,
			cfg.CORS.AllowedMethods,
			cfg.CORS.AllowedHeaders,
			cfg.CORS.ExposedHeaders,
			cfg.CORS.AllowCredentials,
			cfg.CORS.MaxAge,
		),
		middleware.RequestLogger(log),
		middleware.Recovery(log),
		middleware.Timeout(cfg.Proxy.Timeout),
	)

	return &App{
		server: srv,
		logger: log,
		config: cfg,
	}, nil
}

func (a *App) Run() error {
	return a.server.Start()
}

func (a *App) Shutdown(ctx context.Context) error {
	return a.server.Shutdown(ctx)
}
