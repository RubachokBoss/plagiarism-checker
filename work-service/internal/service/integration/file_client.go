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
	FileID string `json:"file_id"`
	Hash   string `json:"hash"`
	Size   int64  `json:"size"`
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
	// Создаем multipart запрос
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Добавляем файл
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, bytes.NewReader(fileContent)); err != nil {
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}

	writer.Close()

	// Создаем запрос
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+c.uploadEndpoint, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Выполняем запрос с повторными попытками
	var resp *http.Response
	var lastErr error

	for i := 0; i <= c.retryCount; i++ {
		if i > 0 {
			c.logger.Warn().Int("attempt", i).Msg("Retrying file upload")
			time.Sleep(c.retryDelay * time.Duration(i))
		}

		resp, err = c.client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}

		if resp != nil {
			resp.Body.Close()
		}
		lastErr = err
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to upload file after %d attempts: %w", c.retryCount+1, lastErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("file service returned status %d: %s", resp.StatusCode, string(body))
	}

	// Парсим ответ
	var uploadResp UploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.Info().
		Str("file_id", uploadResp.FileID).
		Str("hash", uploadResp.Hash).
		Int64("size", uploadResp.Size).
		Msg("File uploaded successfully")

	return &uploadResp, nil
}

func (c *fileClient) GetFile(ctx context.Context, fileID string) ([]byte, error) {
	url := fmt.Sprintf("%s/files/%s", c.baseURL, fileID)

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
	url := fmt.Sprintf("%s/files/%s", c.baseURL, fileID)

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
