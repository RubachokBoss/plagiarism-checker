package httpd

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/service"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) GetWordCloudPNG(w http.ResponseWriter, r *http.Request) {
	workID := chi.URLParam(r, "work_id")
	workID = strings.TrimSpace(workID)
	if workID == "" {
		writeError(w, http.StatusBadRequest, "Work ID is required")
		return
	}

	width := getIntQueryParam(r, "width", 800)
	height := getIntQueryParam(r, "height", 600)
	maxWords := getIntQueryParam(r, "max_words", 200)
	minLen := getIntQueryParam(r, "min_len", 2)

	removeStopwords := false
	if v := r.URL.Query().Get("remove_stopwords"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			removeStopwords = b
		}
	}

	lang := r.URL.Query().Get("lang")

	img, err := h.wordCloudService.RenderWorkWordCloudPNG(r.Context(), workID, service.WordCloudOptions{
		Width:           width,
		Height:          height,
		MaxNumWords:     maxWords,
		MinWordLength:   minLen,
		RemoveStopwords: removeStopwords,
		Language:        lang,
	})
	if err != nil {
		h.handleWordCloudError(w, err)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.WriteHeader(http.StatusOK)
	w.Write(img)
}

func (h *Handler) handleWordCloudError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidWorkID):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrReportNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrFileContentEmpty):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrFileIDEmpty):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrFileServiceError):
		// Ошибка зависимого микросервиса (file-service).
		writeError(w, http.StatusBadGateway, err.Error())
	case errors.Is(err, service.ErrQuickChartError):
		// Ошибка внешнего API (quickchart).
		writeError(w, http.StatusBadGateway, err.Error())
	default:
		// Для отладки возвращаем сообщение ошибки, но оставляем 500 как признак внутреннего сбоя.
		h.logger.Error().Err(err).Msg("Word cloud error")
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}
