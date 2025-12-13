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
	// appRouter держит конечные маршруты; chi не позволяет вешать use после их регистрации
	appRouter chi.Router
	// rootRouter нужен для цепочки middleware, сюда монтируется appRouter
	rootRouter *chi.Mux
	mounted    bool
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
		logger:     logger,
		appRouter:  router,
		rootRouter: chi.NewRouter(),
	}

	s.server = &http.Server{
		Addr:         cfg.Address,
		Handler:      s.rootRouter,
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
	s.rootRouter.Use(middleware.RequestID)
	s.rootRouter.Use(middleware.RealIP)
	s.rootRouter.Use(middleware.StripSlashes)
	s.rootRouter.Use(middleware.CleanPath)
	s.rootRouter.Use(middleware.GetHead)
	s.rootRouter.Use(middleware.Compress(5))

	if corsMiddleware != nil {
		s.rootRouter.Use(corsMiddleware) // cors ставится первым
	}

	if timeoutMiddleware != nil {
		s.rootRouter.Use(timeoutMiddleware) // таймаут перед логированием
	}

	if loggerMiddleware != nil {
		s.rootRouter.Use(loggerMiddleware) // логирование после таймаута
	}

	if recoveryMiddleware != nil {
		s.rootRouter.Use(recoveryMiddleware) // recovery ближе к обработчику
	}

	if !s.mounted {
		// монтируем после навешивания middleware
		s.rootRouter.Mount("/", s.appRouter)
		s.mounted = true
	}
}
