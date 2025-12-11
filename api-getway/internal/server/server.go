package server

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
)

type Server struct {
	server *http.Server
	logger zerolog.Logger
	router chi.Router
}

type ServerConfig struct {
	Address         string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

func NewServer(cfg ServerConfig, router chi.Router, logger zerolog.Logger) *Server {
	s := &Server{
		logger: logger,
		router: router,
	}

	// Создаем HTTP сервер
	s.server = &http.Server{
		Addr:         cfg.Address,
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	return s
}

func (s *Server) Start() error {
	s.logger.Info().Str("address", s.server.Addr).Msg("Starting server")
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info().Msg("Shutting down server")
	return s.server.Shutdown(ctx)
}

func (s *Server) SetupMiddleware(
	corsMiddleware func(http.Handler) http.Handler,
	loggerMiddleware func(http.Handler) http.Handler,
	recoveryMiddleware func(http.Handler) http.Handler,
	timeoutMiddleware func(http.Handler) http.Handler,
) {
	// Базовые middleware
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Recoverer)

	// Кастомные middleware
	if recoveryMiddleware != nil {
		s.router.Use(recoveryMiddleware)
	}

	if loggerMiddleware != nil {
		s.router.Use(loggerMiddleware)
	}

	if corsMiddleware != nil {
		s.router.Use(corsMiddleware)
	}

	if timeoutMiddleware != nil {
		s.router.Use(timeoutMiddleware)
	}

	// Дополнительные middleware
	s.router.Use(middleware.Compress(5))
	s.router.Use(middleware.CleanPath)
	s.router.Use(middleware.GetHead)
	s.router.Use(middleware.StripSlashes)
}
