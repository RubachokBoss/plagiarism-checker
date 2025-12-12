package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

type AnalysisClient interface {
	GetReport(ctx context.Context, workID string) (*AnalysisReport, error)
}

type analysisClient struct {
	baseURL         string
	reportsEndpoint string
	timeout         time.Duration
	retryCount      int
	retryDelay      time.Duration
	client          *http.Client
	logger          zerolog.Logger
}

type AnalysisReport struct {
	WorkID          string     `json:"work_id"`
	Status          string     `json:"status"`
	PlagiarismFlag  bool       `json:"plagiarism_flag"`
	OriginalWorkID  *string    `json:"original_work_id,omitempty"`
	MatchPercentage int        `json:"match_percentage"`
	AnalyzedAt      *time.Time `json:"analyzed_at,omitempty"`
}

func NewAnalysisClient(baseURL, reportsEndpoint string, timeout time.Duration, retryCount int, retryDelay time.Duration, logger zerolog.Logger) AnalysisClient {
	return &analysisClient{
		baseURL:         baseURL,
		reportsEndpoint: reportsEndpoint,
		timeout:         timeout,
		retryCount:      retryCount,
		retryDelay:      retryDelay,
		client: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

func (c *analysisClient) GetReport(ctx context.Context, workID string) (*AnalysisReport, error) {
	url := fmt.Sprintf("%s%s/%s", c.baseURL, c.reportsEndpoint, workID)

	var report *AnalysisReport
	var lastErr error

	for i := 0; i <= c.retryCount; i++ {
		if i > 0 {
			c.logger.Warn().Int("attempt", i).Msg("Retrying analysis report fetch")
			time.Sleep(c.retryDelay * time.Duration(i))
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to get report: %w", err)
			if resp != nil {
				resp.Body.Close()
			}
			continue
		}

		if resp.StatusCode == http.StatusOK {
			if err := json.NewDecoder(resp.Body).Decode(&report); err != nil {
				resp.Body.Close()
				lastErr = fmt.Errorf("failed to decode response: %w", err)
				continue
			}
			resp.Body.Close()
			return report, nil
		}

		if resp.StatusCode == http.StatusNotFound {
			resp.Body.Close()
			return nil, nil // Отчет еще не готов
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		lastErr = fmt.Errorf("analysis service returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil, fmt.Errorf("failed to get analysis report after %d attempts: %w", c.retryCount+1, lastErr)
}
