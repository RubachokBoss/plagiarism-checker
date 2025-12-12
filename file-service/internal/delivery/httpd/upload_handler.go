package httpd

import (
	"encoding/json"
	"io"
	"net/http"
)

func (h *Handler) UploadFile(w http.ResponseWriter, r *http.Request) {
	// Проверяем Content-Type
	contentType := r.Header.Get("Content-Type")
	if contentType != "multipart/form-data" && !contains(contentType, "multipart/form-data") {
		writeError(w, http.StatusBadRequest, "Content-Type must be multipart/form-data")
		return
	}

	// Парсим multipart форму
	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB
		writeError(w, http.StatusBadRequest, "Failed to parse form data")
		return
	}

	// Получаем файл
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "File is required")
		return
	}
	defer file.Close()

	// Читаем содержимое файла
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to read file")
		return
	}

	// Получаем дополнительные параметры
	uploadedBy := r.FormValue("uploaded_by")
	metadataStr := r.FormValue("metadata")

	var metadata []byte
	if metadataStr != "" {
		var metadataMap map[string]interface{}
		if err := json.Unmarshal([]byte(metadataStr), &metadataMap); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid metadata format")
			return
		}
		metadata, _ = json.Marshal(metadataMap)
	}

	// Загружаем файл
	ctx := r.Context()
	response, err := h.uploadService.UploadFileBytes(ctx, fileHeader.Filename, fileBytes, uploadedBy, metadata)
	if err != nil {
		h.handleUploadError(w, err)
		return
	}

	writeSuccess(w, response)
}

func (h *Handler) handleUploadError(w http.ResponseWriter, err error) {
	errMsg := err.Error()

	switch {
	case contains(errMsg, "file size exceeds limit"):
		writeError(w, http.StatusRequestEntityTooLarge, errMsg)
	case contains(errMsg, "file type not allowed"):
		writeError(w, http.StatusUnsupportedMediaType, errMsg)
	case contains(errMsg, "failed to calculate file hash"):
		h.logger.Error().Err(err).Msg("Hash calculation error")
		writeError(w, http.StatusInternalServerError, "Failed to process file")
	case contains(errMsg, "failed to upload file to storage"):
		h.logger.Error().Err(err).Msg("Storage upload error")
		writeError(w, http.StatusInternalServerError, "Failed to store file")
	case contains(errMsg, "failed to save file metadata"):
		h.logger.Error().Err(err).Msg("Database error")
		writeError(w, http.StatusInternalServerError, "Failed to save file information")
	default:
		h.logger.Error().Err(err).Msg("Upload error")
		writeError(w, http.StatusInternalServerError, "Failed to upload file")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr)))
}

// UploadBytesHandler для загрузки файлов из байтов (используется другими сервисами)
func (h *Handler) UploadBytes(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FileName   string          `json:"file_name"`
		FileBytes  []byte          `json:"file_bytes"`
		UploadedBy string          `json:"uploaded_by,omitempty"`
		Metadata   json.RawMessage `json:"metadata,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.FileName == "" || len(req.FileBytes) == 0 {
		writeError(w, http.StatusBadRequest, "file_name and file_bytes are required")
		return
	}

	ctx := r.Context()
	response, err := h.uploadService.UploadFileBytes(ctx, req.FileName, req.FileBytes, req.UploadedBy, req.Metadata)
	if err != nil {
		h.handleUploadError(w, err)
		return
	}

	writeSuccess(w, response)
}
