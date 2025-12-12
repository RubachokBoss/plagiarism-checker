package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/RubachokBoss/plagiarism-checker/file-service/internal/models"
	"github.com/RubachokBoss/plagiarism-checker/file-service/internal/repository"
	"github.com/rs/zerolog"
)

type DeleteService interface {
	DeleteFile(ctx context.Context, fileID string, hardDelete bool) (*models.DeleteFileResponse, error)
	DeleteFileByHash(ctx context.Context, hash string, fileSize int64, hardDelete bool) ([]*models.DeleteFileResponse, error)
	CleanupExpiredFiles(ctx context.Context, daysOld int) (int, error)
}

type deleteService struct {
	metadataRepo repository.FileMetadataRepository
	storageRepo  repository.StorageRepository
	logger       zerolog.Logger
	bucketName   string
}

func NewDeleteService(
	metadataRepo repository.FileMetadataRepository,
	storageRepo repository.StorageRepository,
	logger zerolog.Logger,
	bucketName string,
) DeleteService {
	return &deleteService{
		metadataRepo: metadataRepo,
		storageRepo:  storageRepo,
		logger:       logger,
		bucketName:   bucketName,
	}
}

func (s *deleteService) DeleteFile(ctx context.Context, fileID string, hardDelete bool) (*models.DeleteFileResponse, error) {
	// Получаем метаданные файла
	metadata, err := s.metadataRepo.GetByID(ctx, fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file metadata: %w", err)
	}
	if metadata == nil {
		return nil, errors.New("file not found")
	}

	// Проверяем, не удален ли уже файл
	if metadata.UploadStatus == models.FileStatusDeleted.String() {
		return &models.DeleteFileResponse{
			FileID:  fileID,
			Deleted: false,
			Message: "File already deleted",
		}, nil
	}

	if hardDelete {
		// Полное удаление: удаляем файл из хранилища и метаданные из БД
		if err := s.storageRepo.DeleteFile(ctx, s.bucketName, metadata.StoragePath); err != nil {
			return nil, fmt.Errorf("failed to delete file from storage: %w", err)
		}

		if err := s.metadataRepo.Delete(ctx, fileID); err != nil {
			return nil, fmt.Errorf("failed to delete file metadata: %w", err)
		}

		s.logger.Info().
			Str("file_id", fileID).
			Str("storage_path", metadata.StoragePath).
			Msg("File hard deleted")

		return &models.DeleteFileResponse{
			FileID:  fileID,
			Deleted: true,
			Message: "File permanently deleted",
		}, nil
	} else {
		// Мягкое удаление: только меняем статус
		if err := s.metadataRepo.SoftDelete(ctx, fileID); err != nil {
			return nil, fmt.Errorf("failed to soft delete file: %w", err)
		}

		s.logger.Info().
			Str("file_id", fileID).
			Msg("File soft deleted")

		return &models.DeleteFileResponse{
			FileID:  fileID,
			Deleted: true,
			Message: "File marked as deleted",
		}, nil
	}
}

func (s *deleteService) DeleteFileByHash(ctx context.Context, hash string, fileSize int64, hardDelete bool) ([]*models.DeleteFileResponse, error) {
	// Находим файлы по хэшу
	files, err := s.metadataRepo.GetByHash(ctx, hash, fileSize)
	if err != nil {
		return nil, fmt.Errorf("failed to find files by hash: %w", err)
	}

	var responses []*models.DeleteFileResponse
	for _, file := range files {
		response, err := s.DeleteFile(ctx, file.ID, hardDelete)
		if err != nil {
			s.logger.Error().
				Err(err).
				Str("file_id", file.ID).
				Msg("Failed to delete file by hash")
			continue
		}
		responses = append(responses, response)
	}

	return responses, nil
}

func (s *deleteService) CleanupExpiredFiles(ctx context.Context, daysOld int) (int, error) {
	// Этот метод требует реализации дополнительного запроса в репозитории
	// для поиска файлов старше указанного количества дней
	// В текущей реализации возвращаем заглушку

	s.logger.Warn().
		Int("days_old", daysOld).
		Msg("Cleanup expired files not implemented yet")

	return 0, nil
}
