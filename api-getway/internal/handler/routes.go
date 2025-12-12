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
			r.Put("/{id}/status", workProxy.ServeHTTP)
		})

		// Files endpoints (работают с file-service)
		r.Route("/files", func(r chi.Router) {
			r.Post("/upload", fileProxy.ServeHTTP)
			r.Post("/upload/bytes", fileProxy.ServeHTTP)
			r.Get("/{id}", fileProxy.ServeHTTP)
			r.Get("/{id}/info", fileProxy.ServeHTTP)
			r.Get("/{id}/url", fileProxy.ServeHTTP)
			r.Delete("/{id}", fileProxy.ServeHTTP)
			r.Get("/download/by-hash", fileProxy.ServeHTTP)
		})

		// Analysis endpoints (работают с analysis-service)
		r.Route("/analysis", func(r chi.Router) {
			r.Post("/", analysisProxy.ServeHTTP)
			r.Post("/batch", analysisProxy.ServeHTTP)
			r.Post("/async", analysisProxy.ServeHTTP)
			r.Get("/{work_id}", analysisProxy.ServeHTTP)
			r.Post("/retry", analysisProxy.ServeHTTP)
		})

		// Reports endpoints (работают с analysis-service)
		r.Route("/reports", func(r chi.Router) {
			r.Get("/", analysisProxy.ServeHTTP)
			r.Get("/{report_id}", analysisProxy.ServeHTTP)
			r.Get("/work/{work_id}", analysisProxy.ServeHTTP)
			r.Get("/assignment/{assignment_id}", analysisProxy.ServeHTTP)
			r.Get("/student/{student_id}", analysisProxy.ServeHTTP)
			r.Get("/export", analysisProxy.ServeHTTP)
		})

		// Assignments endpoints (работают с work-service)
		r.Route("/assignments", func(r chi.Router) {
			r.Get("/", workProxy.ServeHTTP)
			r.Post("/", workProxy.ServeHTTP)
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
