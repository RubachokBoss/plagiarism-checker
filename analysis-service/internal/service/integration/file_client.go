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

type FileClient interface {
	GetFileHash(ctx context.Context, fileID string) (string, int64, error)
	GetFileContent(ctx context.Context, fileID string) ([]byte, error)
	GetFileInfo(ctx context.Context, fileID string) (*FileInfoResponse, error)
}

type fileClient struct {
	baseURL    string
	timeout    time.Duration
	retryCount int
	retryDelay time.Duration
	client     *http.Client
	logger     zerolog.Logger
}

type FileInfoResponse struct {
	FileID   string `json:"file_id"`
	Hash     string `json:"hash"`
	Size     int64  `json:"size"`
	FileName string `json:"file_name"`
	MimeType string `json:"mime_type"`
}

func NewFileClient(baseURL string, timeout time.Duration, retryCount int, retryDelay time.Duration, logger zerolog.Logger) FileClient {
	return &fileClient{
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

func (c *fileClient) GetFileHash(ctx context.Context, fileID string) (string, int64, error) {
	url := fmt.Sprintf("%s/files/%s/info", c.baseURL, fileID)

	var fileInfo *FileInfoResponse
	var lastErr error

	for i := 0; i <= c.retryCount; i++ {
		if i > 0 {
			c.logger.Warn().Int("attempt", i).Msg("Retrying file hash fetch")
			time.Sleep(c.retryDelay * time.Duration(i))
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to get file hash: %w", err)
			if resp != nil {
				resp.Body.Close()
			}
			continue
		}

		if resp.StatusCode == http.StatusOK {
			if err := json.NewDecoder(resp.Body).Decode(&fileInfo); err != nil {
				resp.Body.Close()
				lastErr = fmt.Errorf("failed to decode response: %w", err)
				continue
			}
			resp.Body.Close()

			c.logger.Debug().
				Str("file_id", fileID).
				Str("hash", fileInfo.Hash).
				Int64("size", fileInfo.Size).
				Msg("Got file hash")

			return fileInfo.Hash, fileInfo.Size, nil
		}

		if resp.StatusCode == http.StatusNotFound {
			resp.Body.Close()
			return "", 0, fmt.Errorf("file not found: %s", fileID)
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		lastErr = fmt.Errorf("file service returned status %d: %s", resp.StatusCode, string(body))
	}

	return "", 0, fmt.Errorf("failed to get file hash after %d attempts: %w", c.retryCount+1, lastErr)
}

func (c *fileClient) GetFileContent(ctx context.Context, fileID string) ([]byte, error) {
	url := fmt.Sprintf("%s/files/%s", c.baseURL, fileID)

	var lastErr error

	for i := 0; i <= c.retryCount; i++ {
		if i > 0 {
			c.logger.Warn().Int("attempt", i).Msg("Retrying file content fetch")
			time.Sleep(c.retryDelay * time.Duration(i))
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to get file content: %w", err)
			if resp != nil {
				resp.Body.Close()
			}
			continue
		}

		if resp.StatusCode == http.StatusOK {
			content, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				lastErr = fmt.Errorf("failed to read response body: %w", err)
				continue
			}

			c.logger.Debug().
				Str("file_id", fileID).
				Int("content_size", len(content)).
				Msg("Got file content")

			return content, nil
		}

		if resp.StatusCode == http.StatusNotFound {
			resp.Body.Close()
			return nil, fmt.Errorf("file not found: %s", fileID)
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		lastErr = fmt.Errorf("file service returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil, fmt.Errorf("failed to get file content after %d attempts: %w", c.retryCount+1, lastErr)
}

func (c *fileClient) GetFileInfo(ctx context.Context, fileID string) (*FileInfoResponse, error) {
	url := fmt.Sprintf("%s/files/%s/info", c.baseURL, fileID)

	var fileInfo *FileInfoResponse
	var lastErr error

	for i := 0; i <= c.retryCount; i++ {
		if i > 0 {
			c.logger.Warn().Int("attempt", i).Msg("Retrying file info fetch")
			time.Sleep(c.retryDelay * time.Duration(i))
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to get file info: %w", err)
			if resp != nil {
				resp.Body.Close()
			}
			continue
		}

		if resp.StatusCode == http.StatusOK {
			if err := json.NewDecoder(resp.Body).Decode(&fileInfo); err != nil {
				resp.Body.Close()
				lastErr = fmt.Errorf("failed to decode response: %w", err)
				continue
			}
			resp.Body.Close()
			return fileInfo, nil
		}

		if resp.StatusCode == http.StatusNotFound {
			resp.Body.Close()
			return nil, nil
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		lastErr = fmt.Errorf("file service returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil, fmt.Errorf("failed to get file info after %d attempts: %w", c.retryCount+1, lastErr)
}
