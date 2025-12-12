package httpd

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/RubachokBoss/plagiarism-checker/file-service/internal/models"
	"math"
	"net/http"
	"time"
)

func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Получаем статистику файлов
	fileStats, err := h.metadataRepo.GetStats(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get file stats")
		writeError(w, http.StatusInternalServerError, "Failed to retrieve statistics")
		return
	}

	// Получаем статистику хранилища
	storageInfo, err := h.storageRepo.GetBucketStats(ctx, h.uploadService.GetConfig().BucketName)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get storage stats")
		// Продолжаем без статистики хранилища
	}

	// Форматируем размер для читабельности
	formatSize := func(bytes int64) string {
		const unit = 1024
		if bytes < unit {
			return fmt.Sprintf("%d B", bytes)
		}
		div, exp := int64(unit), 0
		for n := bytes / unit; n >= unit; n /= unit {
			div *= unit
			exp++
		}
		return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
	}

	stats := map[string]interface{}{
		"service":        "file-service",
		"timestamp":      time.Now().UTC(),
		"total_files":    fileStats.TotalFiles,
		"total_size":     fileStats.TotalSize,
		"formatted_size": formatSize(fileStats.TotalSize),
		"uploaded_today": fileStats.UploadedToday,
		"average_size":   fileStats.AverageFileSize,
		"active_files":   fileStats.TotalFiles, // В реальности нужно вычитать удаленные
	}

	// Добавляем статистику хранилища если доступна
	if storageInfo != nil {
		stats["storage_provider"] = storageInfo.Provider
		stats["storage_bucket"] = storageInfo.BucketName
		stats["storage_files"] = storageInfo.FileCount
		stats["storage_used"] = formatSize(storageInfo.UsedSpace)
	}

	writeSuccess(w, stats)
}

func (h *Handler) ListFiles(w http.ResponseWriter, r *http.Request) {
	page := getIntQueryParam(r, "page", 1)
	limit := getIntQueryParam(r, "limit", 20)
	status := r.URL.Query().Get("status")

	offset := (page - 1) * limit

	ctx := r.Context()
	files, total, err := h.metadataRepo.GetAll(ctx, limit, offset, status)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list files")
		writeError(w, http.StatusInternalServerError, "Failed to list files")
		return
	}

	// Преобразуем в формат для ответа
	fileResponses := make([]map[string]interface{}, len(files))
	for i, file := range files {
		fileResponses[i] = map[string]interface{}{
			"id":               file.ID,
			"original_name":    file.OriginalName,
			"stored_name":      file.FileName,
			"size":             file.FileSize,
			"mime_type":        file.MimeType,
			"hash":             file.Hash,
			"upload_status":    file.UploadStatus,
			"uploaded_at":      file.UploadedAt.Format(time.RFC3339),
			"uploaded_by":      file.UploadedBy,
			"access_count":     file.AccessCount,
			"last_accessed_at": formatTime(file.LastAccessedAt),
			"extension":        file.FileExtension,
		}
	}

	response := map[string]interface{}{
		"files": fileResponses,
		"pagination": map[string]interface{}{
			"page":     page,
			"limit":    limit,
			"total":    total,
			"pages":    int(math.Ceil(float64(total) / float64(limit))),
			"has_next": page*limit < total,
			"has_prev": page > 1,
		},
	}

	writeSuccess(w, response)
}

func (h *Handler) SearchFiles(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		writeError(w, http.StatusBadRequest, "Search query is required")
		return
	}

	page := getIntQueryParam(r, "page", 1)
	limit := getIntQueryParam(r, "limit", 20)

	ctx := r.Context()

	// Ищем по имени файла
	filesByName, err := h.metadataRepo.GetByFileName(ctx, query)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		h.logger.Error().Err(err).Msg("Failed to search files by name")
		writeError(w, http.StatusInternalServerError, "Failed to search files")
		return
	}

	// Ищем по метаданным (пример для ключа "tags")
	filesByMetadata, err := h.metadataRepo.SearchByMetadata(ctx, "tags", query)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to search files by metadata")
		// Продолжаем без результатов поиска по метаданным
	}

	// Объединяем результаты
	allFiles := make(map[string]*models.FileMetadata)

	if filesByName != nil {
		allFiles[filesByName.ID] = filesByName
	}

	for _, file := range filesByMetadata {
		allFiles[file.ID] = file
	}

	// Применяем пагинацию
	var paginatedFiles []*models.FileMetadata
	start := (page - 1) * limit
	end := start + limit
	i := 0

	for _, file := range allFiles {
		if i >= start && i < end {
			paginatedFiles = append(paginatedFiles, file)
		}
		i++
		if i >= end {
			break
		}
	}

	// Преобразуем результаты
	results := make([]map[string]interface{}, len(paginatedFiles))
	for i, file := range paginatedFiles {
		results[i] = map[string]interface{}{
			"id":            file.ID,
			"original_name": file.OriginalName,
			"size":          file.FileSize,
			"mime_type":     file.MimeType,
			"uploaded_at":   file.UploadedAt.Format(time.RFC3339),
			"hash":          file.Hash,
			"match_type":    "name", // В реальности нужно определить тип совпадения
		}
	}

	writeSuccess(w, map[string]interface{}{
		"query":   query,
		"results": results,
		"count":   len(allFiles),
		"page":    page,
		"limit":   limit,
	})
}

// Вспомогательные функции
func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}
