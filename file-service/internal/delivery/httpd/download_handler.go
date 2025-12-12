package httpd

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "file_id")
	if fileID == "" {
		writeError(w, http.StatusBadRequest, "File ID is required")
		return
	}

	ctx := r.Context()
	response, err := h.downloadService.DownloadFile(ctx, fileID)
	if err != nil {
		h.handleDownloadError(w, err)
		return
	}

	// Устанавливаем заголовки для скачивания
	w.Header().Set("Content-Type", response.ContentType)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+response.FileName+"\"")
	w.Header().Set("Content-Length", strconv.FormatInt(response.FileSize, 10))
	w.Header().Set("Cache-Control", "private, max-age=86400")

	// Отправляем файл
	w.WriteHeader(http.StatusOK)
	w.Write(response.Content)
}

func (h *Handler) GetFileInfo(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "file_id")
	if fileID == "" {
		writeError(w, http.StatusBadRequest, "File ID is required")
		return
	}

	ctx := r.Context()
	info, err := h.downloadService.GetFileInfo(ctx, fileID)
	if err != nil {
		h.handleDownloadError(w, err)
		return
	}

	writeSuccess(w, info)
}

func (h *Handler) GetFileURL(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "file_id")
	if fileID == "" {
		writeError(w, http.StatusBadRequest, "File ID is required")
		return
	}

	expiresIn := getInt64QueryParam(r, "expires", 3600) // По умолчанию 1 час

	ctx := r.Context()
	url, err := h.downloadService.GetPresignedURL(ctx, fileID, expiresIn)
	if err != nil {
		h.handleDownloadError(w, err)
		return
	}

	response := map[string]interface{}{
		"url":        url,
		"expires_in": expiresIn,
		"file_id":    fileID,
	}

	writeSuccess(w, response)
}

func (h *Handler) handleDownloadError(w http.ResponseWriter, err error) {
	errMsg := err.Error()

	switch {
	case contains(errMsg, "file not found"):
		writeError(w, http.StatusNotFound, "File not found")
	case contains(errMsg, "file has been deleted"):
		writeError(w, http.StatusGone, "File has been deleted")
	case contains(errMsg, "failed to download file from storage"):
		h.logger.Error().Err(err).Msg("Storage download error")
		writeError(w, http.StatusInternalServerError, "Failed to retrieve file")
	default:
		h.logger.Error().Err(err).Msg("Download error")
		writeError(w, http.StatusInternalServerError, "Failed to download file")
	}
}

// DownloadByHashHandler для скачивания файла по хэшу (используется Analysis Service)
func (h *Handler) DownloadByHash(w http.ResponseWriter, r *http.Request) {
	hash := r.URL.Query().Get("hash")
	fileSize := getInt64QueryParam(r, "size", 0)

	if hash == "" || fileSize == 0 {
		writeError(w, http.StatusBadRequest, "hash and size parameters are required")
		return
	}

	ctx := r.Context()
	response, err := h.downloadService.DownloadFileByHash(ctx, hash, fileSize)
	if err != nil {
		h.handleDownloadError(w, err)
		return
	}

	// Устанавливаем заголовки для скачивания
	w.Header().Set("Content-Type", response.ContentType)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+response.FileName+"\"")
	w.Header().Set("Content-Length", strconv.FormatInt(response.FileSize, 10))
	w.Header().Set("Cache-Control", "private, max-age=86400")

	// Отправляем файл
	w.WriteHeader(http.StatusOK)
	w.Write(response.Content)
}
