package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/plagiarism-checker/file-service/internal/service"
	"github.com/rs/zerolog"
)

type Handler struct {
	uploadService   service.UploadService
	downloadService service.DownloadService
	deleteService   service.DeleteService
	logger          zerolog.Logger
}

func NewHandler(
	uploadService service.UploadService,
	downloadService service.DownloadService,
	deleteService service.DeleteService,
	logger zerolog.Logger,
) *Handler {
	return &Handler{
		uploadService:   uploadService,
		downloadService: downloadService,
		deleteService:   deleteService,
		logger:          logger,
	}
}

func (h *Handler) RegisterRoutes(router chi.Router) {
	// Health check
	router.Get("/health", h.HealthCheck)
	router.Get("/ready", h.ReadyCheck)
	router.Get("/stats", h.GetStats)

	// File operations
	router.Route("/files", func(r chi.Router) {
		r.Post("/upload", h.UploadFile)
		r.Get("/{file_id}", h.DownloadFile)
		r.Get("/{file_id}/info", h.GetFileInfo)
		r.Get("/{file_id}/url", h.GetFileURL)
		r.Delete("/{file_id}", h.DeleteFile)
	})

	// Admin operations
	router.Route("/admin/files", func(r chi.Router) {
		r.Get("/", h.ListFiles)
		r.Get("/search", h.SearchFiles)
		r.Delete("/cleanup", h.CleanupFiles)
	})
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"service":   "file-service",
		"timestamp": time.Now().UTC(),
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) ReadyCheck(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "ready",
		"timestamp": time.Now().UTC(),
	}

	writeJSON(w, http.StatusOK, response)
}

// Вспомогательные функции
func getIntQueryParam(r *http.Request, key string, defaultValue int) int {
	value := r.URL.Query().Get(key)
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return intValue
}

func getInt64QueryParam(r *http.Request, key string, defaultValue int64) int64 {
	value := r.URL.Query().Get(key)
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return defaultValue
	}

	return intValue
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]interface{}{
		"error":   http.StatusText(status),
		"message": message,
	})
}

func writeSuccess(w http.ResponseWriter, data interface{}) {
	response := map[string]interface{}{
		"success": true,
		"data":    data,
	}
	writeJSON(w, http.StatusOK, response)
}
