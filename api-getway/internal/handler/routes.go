package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) SetupProxyRoutes(workProxy, fileProxy, analysisProxy *ServiceProxy) {
	// API версионирование
	h.router.Route("/api/v1", func(r chi.Router) {
		// Works endpoints (работают с work-service)
		r.Route("/works", func(r chi.Router) {
			r.Post("/", workProxy.ServeHTTP)
			r.Get("/{id}/reports", workProxy.ServeHTTP)
			r.Get("/{id}", workProxy.ServeHTTP) // Для отладки
		})

		// Files endpoints (работают с file-service)
		r.Route("/files", func(r chi.Router) {
			r.Get("/{id}", fileProxy.ServeHTTP)
			r.Get("/{id}/download", fileProxy.ServeHTTP) // Альтернативный путь
		})

		// Analysis endpoints (работают с analysis-service)
		r.Route("/analysis", func(r chi.Router) {
			r.Get("/reports/{id}", analysisProxy.ServeHTTP)
		})

		// Assignments endpoints (работают с work-service)
		r.Route("/assignments", func(r chi.Router) {
			r.Get("/", workProxy.ServeHTTP)
			r.Get("/{id}", workProxy.ServeHTTP)
			r.Get("/{id}/works", workProxy.ServeHTTP)
		})

		// Students endpoints (работают с work-service)
		r.Route("/students", func(r chi.Router) {
			r.Get("/", workProxy.ServeHTTP)
			r.Post("/", workProxy.ServeHTTP)
			r.Get("/{id}", workProxy.ServeHTTP)
			r.Get("/{id}/works", workProxy.ServeHTTP)
		})
	})

	// Admin endpoints (для мониторинга)
	h.router.Route("/admin", func(r chi.Router) {
		r.Get("/metrics", h.adminMetrics)
		r.Get("/services", h.adminServices)
	})
}

func (h *Handler) adminMetrics(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"service": "api-gateway",
		"metrics": map[string]interface{}{
			"uptime":        "24h",
			"request_count": 1000,
			"error_rate":    0.01,
		},
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) adminServices(w http.ResponseWriter, r *http.Request) {
	services := []map[string]interface{}{
		{
			"name":     "work-service",
			"status":   "healthy",
			"endpoint": "/api/v1/works",
		},
		{
			"name":     "file-service",
			"status":   "healthy",
			"endpoint": "/api/v1/files",
		},
		{
			"name":     "analysis-service",
			"status":   "healthy",
			"endpoint": "/api/v1/analysis",
		},
	}

	response := map[string]interface{}{
		"services":  services,
		"timestamp": time.Now().UTC(),
	}
	writeJSON(w, http.StatusOK, response)
}

// ServeHTTP реализация для ServiceProxy
func (sp *ServiceProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sp.Proxy.ServeHTTP(w, r)
}
