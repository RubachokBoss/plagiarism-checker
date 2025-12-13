package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/repository"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/service/integration"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type WordCloudService interface {
	RenderWorkWordCloudPNG(ctx context.Context, workID string, opts WordCloudOptions) ([]byte, error)
}

type WordCloudOptions struct {
	Width           int
	Height          int
	MaxNumWords     int
	MinWordLength   int
	RemoveStopwords bool
	Language        string
}

type wordCloudService struct {
	reportRepo repository.ReportRepository
	fileClient integration.FileClient
	httpClient *http.Client
	logger     zerolog.Logger
}

func NewWordCloudService(reportRepo repository.ReportRepository, fileClient integration.FileClient, logger zerolog.Logger) WordCloudService {
	return &wordCloudService{
		reportRepo: reportRepo,
		fileClient: fileClient,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logger:     logger,
	}
}

func (s *wordCloudService) RenderWorkWordCloudPNG(ctx context.Context, workID string, opts WordCloudOptions) ([]byte, error) {
	workID = strings.TrimSpace(workID)
	if workID == "" {
		return nil, fmt.Errorf("work id is required")
	}
	if _, err := uuid.Parse(workID); err != nil {
		return nil, fmt.Errorf("invalid work_id")
	}

	report, err := s.reportRepo.GetByWorkID(ctx, workID)
	if err != nil {
		return nil, fmt.Errorf("failed to get report: %w", err)
	}
	if report == nil {
		return nil, fmt.Errorf("report not found for this work")
	}
	if strings.TrimSpace(report.FileID) == "" {
		return nil, fmt.Errorf("file_id is empty for this work")
	}

	content, err := s.fileClient.GetFileContent(ctx, report.FileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file content: %w", err)
	}

	text := strings.TrimSpace(string(content))
	if text == "" {
		return nil, fmt.Errorf("file content is empty")
	}

	width := opts.Width
	if width <= 0 {
		width = 800
	}
	height := opts.Height
	if height <= 0 {
		height = 600
	}
	maxWords := opts.MaxNumWords
	if maxWords <= 0 {
		maxWords = 200
	}
	minLen := opts.MinWordLength
	if minLen <= 0 {
		minLen = 2
	}
	lang := strings.TrimSpace(opts.Language)
	if lang == "" {
		lang = "ru"
	}

	payload := map[string]interface{}{
		"text":            text,
		"format":          "png",
		"width":           width,
		"height":          height,
		"maxNumWords":     maxWords,
		"minWordLength":   minLen,
		"removeStopwords": opts.RemoveStopwords,
		"language":        lang,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal quickchart payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://quickchart.io/wordcloud", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create quickchart request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("quickchart request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("quickchart returned status %d: %s", resp.StatusCode, string(b))
	}

	img, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read quickchart response: %w", err)
	}
	if len(img) == 0 {
		return nil, fmt.Errorf("quickchart returned empty image")
	}

	s.logger.Info().
		Str("work_id", workID).
		Str("file_id", report.FileID).
		Int("png_size", len(img)).
		Msg("Word cloud generated")

	return img, nil
}

