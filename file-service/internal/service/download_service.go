package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/plagiarism-checker/file-service/internal/models"
	"github.com/plagiarism-checker/file-service/internal/repository"
	"github.com/rs/zerolog"
)

type DownloadService interface {
	DownloadFile(ctx context.Context, fileID string) (*models.DownloadFileResponse, error)
	DownloadFileByHash(ctx context.Context, hash string, fileSize int64) (*models.DownloadFileResponse, error)
	GetFileInfo(ctx context.Context, fileID string) (*models.FileInfoResponse, error)
	GetPresignedURL(ctx context.Context, fileID string, expiresIn int64) (string, error)
}

type downloadService struct {
	metadataRepo repository.FileMetadataRepository
	storageRepo  repository.StorageRepository
	logger       zerolog.Logger
	bucketName   string
}

func NewDownloadService(
	metadataRepo repository.FileMetadataRepository,
	storageRepo repository.StorageRepository,
	logger zerolog.Logger,
	bucketName string,
) DownloadService {
	return &downloadService{
		metadataRepo: metadataRepo,
		storageRepo:  storageRepo,
		logger:       logger,
		bucketName:   bucketName,
	}
}

func (s *downloadService) DownloadFile(ctx context.Context, fileID string) (*models.DownloadFileResponse, error) {
	// Получаем метаданные файла
	metadata, err := s.metadataRepo.GetByID(ctx, fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file metadata: %w", err)
	}
	if metadata == nil {
		return nil, errors.New("file not found")
	}

	// Проверяем статус файла
	if metadata.UploadStatus == models.FileStatusDeleted.String() {
		return nil, errors.New("file has been deleted")
	}

	// Скачиваем файл из хранилища
	fileReader, fileSize, err := s.storageRepo.DownloadFile(ctx, s.bucketName, metadata.StoragePath)
	if err != nil {
		return nil, fmt.Errorf("failed to download file from storage: %w", err)
	}

	// Обновляем статистику доступа
	if err := s.metadataRepo.UpdateAccessInfo(ctx, fileID); err != nil {
		s.logger.Error().Err(err).Str("file_id", fileID).Msg("Failed to update access info")
	}

	// Читаем содержимое файла
	fileContent, err := io.ReadAll(fileReader)
	fileReader.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}

	s.logger.Info().
		Str("file_id", fileID).
		Str("file_name", metadata.OriginalName).
		Int64("size", fileSize).
		Int("access_count", metadata.AccessCount+1).
		Msg("File downloaded")

	return &models.DownloadFileResponse{
		Content:     fileContent,
		FileName:    metadata.OriginalName,
		ContentType: metadata.MimeType,
		FileSize:    fileSize,
	}, nil
}

func (s *downloadService) DownloadFileByHash(ctx context.Context, hash string, fileSize int64) (*models.DownloadFileResponse, error) {
	// Находим файлы по хэшу
	files, err := s.metadataRepo.GetByHash(ctx, hash, fileSize)
	if err != nil {
		return nil, fmt.Errorf("failed to find files by hash: %w", err)
	}
	if len(files) == 0 {
		return nil, errors.New("file not found")
	}

	// Берем первый найденный файл
	metadata := files[0]

	// Проверяем статус файла
	if metadata.UploadStatus == models.FileStatusDeleted.String() {
		return nil, errors.New("file has been deleted")
	}

	// Скачиваем файл из хранилища
	fileReader, actualFileSize, err := s.storageRepo.DownloadFile(ctx, s.bucketName, metadata.StoragePath)
	if err != nil {
		return nil, fmt.Errorf("failed to download file from storage: %w", err)
	}

	// Обновляем статистику доступа
	if err := s.metadataRepo.UpdateAccessInfo(ctx, metadata.ID); err != nil {
		s.logger.Error().Err(err).Str("file_id", metadata.ID).Msg("Failed to update access info")
	}

	// Читаем содержимое файла
	fileContent, err := io.ReadAll(fileReader)
	fileReader.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}

	s.logger.Info().
		Str("file_id", metadata.ID).
		Str("file_name", metadata.OriginalName).
		Int64("size", actualFileSize).
		Int("access_count", metadata.AccessCount+1).
		Msg("File downloaded by hash")

	return &models.DownloadFileResponse{
		Content:     fileContent,
		FileName:    metadata.OriginalName,
		ContentType: metadata.MimeType,
		FileSize:    actualFileSize,
	}, nil
}

func (s *downloadService) GetFileInfo(ctx context.Context, fileID string) (*models.FileInfoResponse, error) {
	// Получаем метаданные файла
	metadata, err := s.metadataRepo.GetByID(ctx, fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file metadata: %w", err)
	}
	if metadata == nil {
		return nil, errors.New("file not found")
	}

	// Проверяем статус файла
	if metadata.UploadStatus == models.FileStatusDeleted.String() {
		return nil, errors.New("file has been deleted")
	}

	// Генерируем URL для доступа
	storageURL := fmt.Sprintf("/files/%s", metadata.StoragePath)

	return &models.FileInfoResponse{
		FileID:         metadata.ID,
		OriginalName:   metadata.OriginalName,
		FileSize:       metadata.FileSize,
		MimeType:       metadata.MimeType,
		Hash:           metadata.Hash,
		UploadStatus:   metadata.UploadStatus,
		UploadedAt:     metadata.UploadedAt,
		AccessCount:    metadata.AccessCount,
		LastAccessedAt: metadata.LastAccessedAt,
		StorageURL:     storageURL,
		Metadata:       metadata.Metadata,
	}, nil
}

func (s *downloadService) GetPresignedURL(ctx context.Context, fileID string, expiresIn int64) (string, error) {
	// Получаем метаданные файла
	metadata, err := s.metadataRepo.GetByID(ctx, fileID)
	if err != nil {
		return "", fmt.Errorf("failed to get file metadata: %w", err)
	}
	if metadata == nil {
		return "", errors.New("file not found")
	}

	// Проверяем статус файла
	if metadata.UploadStatus == models.FileStatusDeleted.String() {
		return "", errors.New("file has been deleted")
	}

	// Генерируем предварительно подписанный URL
	url, err := s.storageRepo.GetPresignedURL(ctx, s.bucketName, metadata.StoragePath, expiresIn)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	// Обновляем статистику доступа (только если URL будет использован)
	if err := s.metadataRepo.UpdateAccessInfo(ctx, fileID); err != nil {
		s.logger.Error().Err(err).Str("file_id", fileID).Msg("Failed to update access info")
	}

	s.logger.Info().
		Str("file_id", fileID).
		Str("url", url).
		Int64("expires_in", expiresIn).
		Msg("Generated presigned URL")

	return url, nil
}
