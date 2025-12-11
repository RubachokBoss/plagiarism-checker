package handler

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

type Handler struct {
	router      *chi.Mux
	logger      zerolog.Logger
	proxyConfig ProxyConfig
}

type ProxyConfig struct {
	Timeout         time.Duration
	MaxIdleConns    int
	IdleConnTimeout time.Duration
}

type ServiceProxy struct {
	TargetURL  *url.URL
	Proxy      *httputil.ReverseProxy
	PathPrefix string
}

func NewHandler(logger zerolog.Logger, proxyConfig ProxyConfig) *Handler {
	h := &Handler{
		router:      chi.NewRouter(),
		logger:      logger,
		proxyConfig: proxyConfig,
	}

	h.setupRoutes()
	return h
}

func (h *Handler) setupRoutes() {
	// Health check
	h.router.Get("/health", h.HealthCheck)
	h.router.Get("/ready", h.ReadyCheck)
	h.router.Get("/live", h.LiveCheck)
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"service":   "api-gateway",
		"version":   "1.0.0",
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) ReadyCheck(w http.ResponseWriter, r *http.Request) {
	// Здесь можно добавить проверку зависимостей
	response := map[string]interface{}{
		"status":    "ready",
		"timestamp": time.Now().UTC(),
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) LiveCheck(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "alive",
		"timestamp": time.Now().UTC(),
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) CreateServiceProxy(targetURL, pathPrefix string) (*ServiceProxy, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// Настраиваем транспорт
	transport := &http.Transport{
		MaxIdleConns:       h.proxyConfig.MaxIdleConns,
		IdleConnTimeout:    h.proxyConfig.IdleConnTimeout,
		DisableCompression: true,
	}

	proxy.Transport = transport

	// Модифицируем запрос
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.Host = target.Host

		// Убираем префикс API Gateway
		if pathPrefix != "" {
			req.URL.Path = req.URL.Path[len(pathPrefix):]
		}

		// Добавляем заголовки
		req.Header.Set("X-Forwarded-Host", req.Host)
		req.Header.Set("X-Real-IP", r.RemoteAddr)

		h.logger.Debug().
			Str("method", req.Method).
			Str("path", req.URL.Path).
			Str("target", target.String()).
			Msg("Proxying request")
	}

	// Обработка ошибок прокси
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		h.logger.Error().
			Err(err).
			Str("url", r.URL.String()).
			Str("target", target.String()).
			Msg("Proxy error")

		errorResponse := map[string]interface{}{
			"error":   "Service unavailable",
			"message": "The service is temporarily unavailable. Please try again later.",
			"code":    "SERVICE_UNAVAILABLE",
		}

		writeJSON(w, http.StatusServiceUnavailable, errorResponse)
	}

	return &ServiceProxy{
		TargetURL:  target,
		Proxy:      proxy,
		PathPrefix: pathPrefix,
	}, nil
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// В реальной реализации здесь нужно использовать json.NewEncoder
	// Для простоты оставляем заглушку
	w.Write([]byte(`{"message": "JSON response"}`))
}

func (h *Handler) GetRouter() *chi.Mux {
	return h.router
}
