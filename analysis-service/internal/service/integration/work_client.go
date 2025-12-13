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
	fileClient FileClient
	logger     zerolog.Logger
}

func NewWorkClient(baseURL string, timeout time.Duration, retryCount int, retryDelay time.Duration, fileClient FileClient, logger zerolog.Logger) WorkClient {
	return &workClient{
		baseURL:    baseURL,
		timeout:    timeout,
		retryCount: retryCount,
		retryDelay: retryDelay,
		client: &http.Client{
			Timeout: timeout,
		},
		fileClient: fileClient,
		logger:     logger,
	}
}

func (c *workClient) GetPreviousWorks(ctx context.Context, assignmentID, excludeWorkID string) ([]models.SimilarWork, error) {
	if c.fileClient == nil {
		return nil, fmt.Errorf("file client is not configured")
	}

	page := 1
	limit := 100
	var allWorks []models.SimilarWork

	for {
		url := fmt.Sprintf("%s/api/v1/assignments/%s/works?page=%d&limit=%d", c.baseURL, assignmentID, page, limit)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := c.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to get previous works: %w", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			resp.Body.Close()
			return []models.SimilarWork{}, nil
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("work service returned status %d: %s", resp.StatusCode, string(body))
		}

		var worksResp struct {
			Success bool `json:"success"`
			Data    struct {
				Works []struct {
					ID           string    `json:"id"`
					StudentID    string    `json:"student_id"`
					AssignmentID string    `json:"assignment_id"`
					FileID       string    `json:"file_id"`
					CreatedAt    time.Time `json:"created_at"`
				} `json:"works"`
				Total int `json:"total"`
				Page  int `json:"page"`
				Limit int `json:"limit"`
			} `json:"data"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&worksResp); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decode work service response: %w", err)
		}
		resp.Body.Close()

		for _, w := range worksResp.Data.Works {
			if w.ID == "" || w.ID == excludeWorkID || w.FileID == "" {
				continue
			}

			fileHash, _, err := c.fileClient.GetFileHash(ctx, w.FileID)
			if err != nil {
				c.logger.Warn().
					Err(err).
					Str("work_id", w.ID).
					Str("file_id", w.FileID).
					Msg("Failed to fetch hash for previous work, skipping")
				continue
			}

			allWorks = append(allWorks, models.SimilarWork{
				WorkID:      w.ID,
				StudentID:   w.StudentID,
				FileHash:    fileHash,
				SubmittedAt: w.CreatedAt,
			})
		}

		if len(worksResp.Data.Works) == 0 || page*limit >= worksResp.Data.Total {
			break
		}
		page++
	}

	return allWorks, nil
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
