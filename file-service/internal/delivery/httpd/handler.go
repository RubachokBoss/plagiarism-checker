package httpd

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/RubachokBoss/plagiarism-checker/file-service/internal/models"
	"github.com/RubachokBoss/plagiarism-checker/file-service/internal/repository"
	"github.com/RubachokBoss/plagiarism-checker/file-service/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

type Handler struct {
	uploadService   service.UploadService
	downloadService service.DownloadService
	deleteService   service.DeleteService
	metadataRepo    repository.FileMetadataRepository
	storageRepo     repository.StorageRepository
	logger          zerolog.Logger
}

func NewHandler(
	uploadService service.UploadService,
	downloadService service.DownloadService,
	deleteService service.DeleteService,
	metadataRepo repository.FileMetadataRepository,
	storageRepo repository.StorageRepository,
	logger zerolog.Logger,
) *Handler {
	return &Handler{
		uploadService:   uploadService,
		downloadService: downloadService,
		deleteService:   deleteService,
		metadataRepo:    metadataRepo,
		storageRepo:     storageRepo,
		logger:          logger,
	}
}

func (h *Handler) RegisterRoutes(router chi.Router) {
	// Health check
	router.Get("/health", h.HealthCheck)
	router.Get("/ready", h.ReadyCheck)
	router.Get("/stats", h.GetStats)

	// Versioned API
	router.Route("/api/v1", func(api chi.Router) {
		// File operations
		api.Route("/files", func(r chi.Router) {
			r.Post("/upload", h.UploadFile)
			r.Post("/upload/bytes", h.UploadBytes) // Новый эндпоинт
			r.Get("/{file_id}", h.DownloadFile)
			r.Get("/{file_id}/info", h.GetFileInfo)
			r.Get("/{file_id}/url", h.GetFileURL)
			r.Delete("/{file_id}", h.DeleteFile)
			r.Get("/download/by-hash", h.DownloadByHash) // Новый эндпоинт
		})

		// Admin operations
		api.Route("/admin/files", func(r chi.Router) {
			r.Get("/", h.ListFiles)
			r.Get("/search", h.SearchFiles)
			r.Delete("/cleanup", h.CleanupFiles)
			r.Get("/associations/{file_id}", h.GetFileAssociations) // Новый эндпоинт
			r.Post("/associate", h.AssociateFile)                   // Новый эндпоинт
		})
	})
}

// Новые методы для ассоциаций файлов
func (h *Handler) GetFileAssociations(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "file_id")
	if fileID == "" {
		writeError(w, http.StatusBadRequest, "File ID is required")
		return
	}

	// В реальности нужно реализовать репозиторий для ассоциаций
	// Здесь заглушка для примера
	associations := []map[string]interface{}{
		{
			"entity_type": "work",
			"entity_id":   "12345",
			"association": "primary",
			"created_at":  time.Now().UTC().Format(time.RFC3339),
		},
	}

	writeSuccess(w, map[string]interface{}{
		"file_id":      fileID,
		"associations": associations,
		"count":        len(associations),
	})
}

func (h *Handler) AssociateFile(w http.ResponseWriter, r *http.Request) {
	var req models.AssociateFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Проверяем существование файла
	exists, err := h.metadataRepo.Exists(r.Context(), req.FileID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to check file existence")
		writeError(w, http.StatusInternalServerError, "Failed to check file")
		return
	}

	if !exists {
		writeError(w, http.StatusNotFound, "File not found")
		return
	}

	// В реальности сохраняем ассоциацию в БД
	// Здесь заглушка
	writeSuccess(w, map[string]interface{}{
		"success":          true,
		"file_id":          req.FileID,
		"entity_type":      req.EntityType,
		"entity_id":        req.EntityID,
		"association_type": req.AssociationType,
		"associated_at":    time.Now().UTC().Format(time.RFC3339),
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
