package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

type FileClient interface {
	UploadFile(ctx context.Context, fileContent []byte, fileName string) (*UploadResponse, error)
	GetFile(ctx context.Context, fileID string) ([]byte, error)
	DeleteFile(ctx context.Context, fileID string) error
}

type fileClient struct {
	baseURL        string
	uploadEndpoint string
	timeout        time.Duration
	retryCount     int
	retryDelay     time.Duration
	client         *http.Client
	logger         zerolog.Logger
}

type UploadResponse struct {
	FileID string
	Hash   string
	Size   int64
}

func NewFileClient(baseURL, uploadEndpoint string, timeout time.Duration, retryCount int, retryDelay time.Duration, logger zerolog.Logger) FileClient {
	return &fileClient{
		baseURL:        baseURL,
		uploadEndpoint: uploadEndpoint,
		timeout:        timeout,
		retryCount:     retryCount,
		retryDelay:     retryDelay,
		client: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

func (c *fileClient) UploadFile(ctx context.Context, fileContent []byte, fileName string) (*UploadResponse, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, bytes.NewReader(fileContent)); err != nil {
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	body := buf.Bytes()
	contentType := writer.FormDataContentType()

	var resp *http.Response
	var lastErr error

	for i := 0; i <= c.retryCount; i++ {
		if i > 0 {
			c.logger.Warn().Int("attempt", i).Msg("Retrying file upload")
			time.Sleep(c.retryDelay * time.Duration(i))
		}

		req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+c.uploadEndpoint, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", contentType)

		resp, err = c.client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}

		if resp != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			lastErr = fmt.Errorf("file service returned status %d", resp.StatusCode)
		} else {
			lastErr = err
		}
	}

	if resp == nil || resp.StatusCode != http.StatusOK {
		if lastErr != nil {
			return nil, fmt.Errorf("failed to upload file after %d attempts: %w", c.retryCount+1, lastErr)
		}
		return nil, fmt.Errorf("failed to upload file after %d attempts", c.retryCount+1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("file service returned status %d: %s", resp.StatusCode, string(body))
	}

	var envelope struct {
		Data struct {
			FileID   string `json:"file_id"`
			Hash     string `json:"hash"`
			FileSize int64  `json:"file_size"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	uploadResp := UploadResponse{
		FileID: envelope.Data.FileID,
		Hash:   envelope.Data.Hash,
		Size:   envelope.Data.FileSize,
	}

	c.logger.Info().
		Str("file_id", uploadResp.FileID).
		Str("hash", uploadResp.Hash).
		Int64("size", uploadResp.Size).
		Msg("File uploaded successfully")

	return &uploadResp, nil
}

func (c *fileClient) GetFile(ctx context.Context, fileID string) ([]byte, error) {
	url := fmt.Sprintf("%s/api/v1/files/%s", c.baseURL, fileID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("file service returned status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (c *fileClient) DeleteFile(ctx context.Context, fileID string) error {
	url := fmt.Sprintf("%s/api/v1/files/%s", c.baseURL, fileID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("file service returned status %d", resp.StatusCode)
	}

	return nil
}
