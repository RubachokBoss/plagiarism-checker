package httpd

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "file_id")
	if fileID == "" {
		writeError(w, http.StatusBadRequest, "File ID is required")
		return
	}

	hardDelete := r.URL.Query().Get("hard") == "true"

	ctx := r.Context()
	response, err := h.deleteService.DeleteFile(ctx, fileID, hardDelete)
	if err != nil {
		h.handleDeleteError(w, err)
		return
	}

	writeSuccess(w, response)
}

func (h *Handler) CleanupFiles(w http.ResponseWriter, r *http.Request) {
	daysOld := getIntQueryParam(r, "days", 30)
	hardDelete := r.URL.Query().Get("hard") == "true"

	ctx := r.Context()
	count, err := h.deleteService.CleanupExpiredFiles(ctx, daysOld)
	if err != nil {
		h.logger.Error().Err(err).Msg("Cleanup error")
		writeError(w, http.StatusInternalServerError, "Failed to cleanup files")
		return
	}

	response := map[string]interface{}{
		"cleaned":     count,
		"days_old":    daysOld,
		"hard_delete": hardDelete,
		"message":     "Cleanup completed",
	}

	writeSuccess(w, response)
}

func (h *Handler) handleDeleteError(w http.ResponseWriter, err error) {
	errMsg := err.Error()

	switch {
	case contains(errMsg, "file not found"):
		writeError(w, http.StatusNotFound, "File not found")
	case contains(errMsg, "failed to delete file from storage"):
		h.logger.Error().Err(err).Msg("Storage delete error")
		writeError(w, http.StatusInternalServerError, "Failed to delete file from storage")
	case contains(errMsg, "failed to delete file metadata"):
		h.logger.Error().Err(err).Msg("Database delete error")
		writeError(w, http.StatusInternalServerError, "Failed to delete file information")
	default:
		h.logger.Error().Err(err).Msg("Delete error")
		writeError(w, http.StatusInternalServerError, "Failed to delete file")
	}
}
