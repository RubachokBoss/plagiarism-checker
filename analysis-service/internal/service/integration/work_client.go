package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/models"
	"github.com/rs/zerolog"
)

type WorkClient interface {
	GetPreviousWorks(ctx context.Context, assignmentID, excludeWorkID string) ([]models.SimilarWork, error)
	GetWorkInfo(ctx context.Context, workID string) (*models.SimilarWork, error)
	UpdateWorkStatus(ctx context.Context, workID, status string) error
}

type workClient struct {
	baseURL    string
	timeout    time.Duration
	retryCount int
	retryDelay time.Duration
	client     *http.Client
	logger     zerolog.Logger
}

func NewWorkClient(baseURL string, timeout time.Duration, retryCount int, retryDelay time.Duration, logger zerolog.Logger) WorkClient {
	return &workClient{
		baseURL:    baseURL,
		timeout:    timeout,
		retryCount: retryCount,
		retryDelay: retryDelay,
		client: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

func (c *workClient) GetPreviousWorks(ctx context.Context, assignmentID, excludeWorkID string) ([]models.SimilarWork, error) {
	// In real implementation, this would call Work Service API
	// For now, return mock data or implement actual HTTP call

	c.logger.Debug().
		Str("assignment_id", assignmentID).
		Str("exclude_work_id", excludeWorkID).
		Msg("Getting previous works")

	// This is a placeholder - in real implementation, make HTTP request
	return []models.SimilarWork{}, nil
}

func (c *workClient) GetWorkInfo(ctx context.Context, workID string) (*models.SimilarWork, error) {
	url := fmt.Sprintf("%s/api/v1/works/%s", c.baseURL, workID)

	var workInfo *models.SimilarWork
	var lastErr error

	for i := 0; i <= c.retryCount; i++ {
		if i > 0 {
			c.logger.Warn().Int("attempt", i).Msg("Retrying work info fetch")
			time.Sleep(c.retryDelay * time.Duration(i))
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to get work info: %w", err)
			if resp != nil {
				resp.Body.Close()
			}
			continue
		}

		if resp.StatusCode == http.StatusOK {
			if err := json.NewDecoder(resp.Body).Decode(&workInfo); err != nil {
				resp.Body.Close()
				lastErr = fmt.Errorf("failed to decode response: %w", err)
				continue
			}
			resp.Body.Close()
			return workInfo, nil
		}

		if resp.StatusCode == http.StatusNotFound {
			resp.Body.Close()
			return nil, nil
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		lastErr = fmt.Errorf("work service returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil, fmt.Errorf("failed to get work info after %d attempts: %w", c.retryCount+1, lastErr)
}

func (c *workClient) UpdateWorkStatus(ctx context.Context, workID, status string) error {
	url := fmt.Sprintf("%s/api/v1/works/%s/status", c.baseURL, workID)

	updateReq := map[string]string{
		"status": status,
	}

	body, err := json.Marshal(updateReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	var lastErr error
	for i := 0; i <= c.retryCount; i++ {
		if i > 0 {
			c.logger.Warn().Int("attempt", i).Msg("Retrying work status update")
			time.Sleep(c.retryDelay * time.Duration(i))
		}

		req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(body))
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to update work status: %w", err)
			if resp != nil {
				resp.Body.Close()
			}
			continue
		}

		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
			resp.Body.Close()
			c.logger.Info().
				Str("work_id", workID).
				Str("status", status).
				Msg("Work status updated")
			return nil
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		lastErr = fmt.Errorf("work service returned status %d: %s", resp.StatusCode, string(body))
	}

	return fmt.Errorf("failed to update work status after %d attempts: %w", c.retryCount+1, lastErr)
}
