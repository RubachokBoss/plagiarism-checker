package app

import (
	"context"

	"github.com/RubachokBoss/plagiarism-checker/api-gateway/internal/config"
	"github.com/RubachokBoss/plagiarism-checker/api-gateway/internal/handler"
	"github.com/RubachokBoss/plagiarism-checker/api-gateway/internal/middleware"
	"github.com/RubachokBoss/plagiarism-checker/api-gateway/internal/server"
	"github.com/rs/zerolog"
)

type App struct {
	server *server.Server
	logger zerolog.Logger
	config *config.Config
}

func New(cfg *config.Config, log zerolog.Logger) (*App, error) {
	h := handler.NewHandler(log, handler.ProxyConfig{
		Timeout:         cfg.Proxy.Timeout,
		MaxIdleConns:    cfg.Proxy.MaxIdleConns,
		IdleConnTimeout: cfg.Proxy.IdleConnTimeout,
	})

	router := h.GetRouter()

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

	// важно: middleware должны быть навешаны до регистрации роутов
	h.SetupBaseRoutes()

	workProxy, err := h.CreateServiceProxy(cfg.Services.Work.URL, "")
	if err != nil {
		return nil, err
	}

	fileProxy, err := h.CreateServiceProxy(cfg.Services.File.URL, "")
	if err != nil {
		return nil, err
	}

	analysisProxy, err := h.CreateServiceProxy(cfg.Services.Analysis.URL, "")
	if err != nil {
		return nil, err
	}

	// Настраиваем маршруты прокси
	h.SetupProxyRoutes(workProxy, fileProxy, analysisProxy)

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
