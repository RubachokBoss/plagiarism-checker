package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/plagiarism-checker/file-service/internal/models"
	"github.com/plagiarism-checker/file-service/internal/repository"
	"github.com/plagiarism-checker/file-service/pkg/hash"
	"github.com/rs/zerolog"
)

type UploadService interface {
	UploadFile(ctx context.Context, fileHeader *multipart.FileHeader, uploadedBy string, metadata []byte) (*models.UploadFileResponse, error)
	UploadFileBytes(ctx context.Context, fileName string, fileBytes []byte, uploadedBy string, metadata []byte) (*models.UploadFileResponse, error)
	CheckDuplicate(ctx context.Context, fileHash string, fileSize int64) ([]*models.FileMetadata, error)
}

type uploadService struct {
	metadataRepo repository.FileMetadataRepository
	storageRepo  repository.StorageRepository
	hashService  HashService
	logger       zerolog.Logger
	config       UploadConfig
}

type UploadConfig struct {
	MaxUploadSize  int64
	BucketName     string
	AllowedTypes   []string
	GenerateHash   bool
	CheckDuplicate bool
}

func NewUploadService(
	metadataRepo repository.FileMetadataRepository,
	storageRepo repository.StorageRepository,
	hashService HashService,
	logger zerolog.Logger,
	config UploadConfig,
) UploadService {
	return &uploadService{
		metadataRepo: metadataRepo,
		storageRepo:  storageRepo,
		hashService:  hashService,
		logger:       logger,
		config:       config,
	}
}

func (s *uploadService) UploadFile(ctx context.Context, fileHeader *multipart.FileHeader, uploadedBy string, metadata []byte) (*models.UploadFileResponse, error) {
	// Открываем файл
	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Читаем содержимое файла
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return s.UploadFileBytes(ctx, fileHeader.Filename, fileBytes, uploadedBy, metadata)
}

func (s *uploadService) UploadFileBytes(ctx context.Context, fileName string, fileBytes []byte, uploadedBy string, metadata []byte) (*models.UploadFileResponse, error) {
	// Проверяем размер файла
	if int64(len(fileBytes)) > s.config.MaxUploadSize {
		return nil, fmt.Errorf("file size exceeds limit: %d bytes", s.config.MaxUploadSize)
	}

	// Определяем MIME тип
	mimeType := s.detectMimeType(fileName, fileBytes)

	// Проверяем разрешенные типы файлов
	if !s.isAllowedType(mimeType, fileName) {
		return nil, fmt.Errorf("file type not allowed: %s", mimeType)
	}

	// Генерируем хэш файла
	fileHash, err := s.hashService.CalculateHash(fileBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate file hash: %w", err)
	}

	// Проверяем дубликаты
	if s.config.CheckDuplicate {
		duplicates, err := s.CheckDuplicate(ctx, fileHash, int64(len(fileBytes)))
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to check for duplicates")
		} else if len(duplicates) > 0 {
			s.logger.Info().
				Str("hash", fileHash).
				Int64("size", int64(len(fileBytes))).
				Int("duplicates", len(duplicates)).
				Msg("Duplicate file found")

			// Возвращаем информацию о существующем файле
			return s.createDuplicateResponse(duplicates[0]), nil
		}
	}

	// Генерируем уникальное имя файла
	uniqueFileName := s.generateUniqueFileName(fileName)

	// Генерируем путь для хранения
	storagePath := s.generateStoragePath(uniqueFileName)

	// Загружаем файл в хранилище
	if err := s.storageRepo.UploadFile(
		ctx,
		s.config.BucketName,
		storagePath,
		strings.NewReader(string(fileBytes)),
		int64(len(fileBytes)),
	); err != nil {
		return nil, fmt.Errorf("failed to upload file to storage: %w", err)
	}

	// Создаем метаданные файла
	fileID := uuid.New().String()
	fileMetadata := &models.FileMetadata{
		ID:              fileID,
		OriginalName:    fileName,
		FileName:        uniqueFileName,
		FileExtension:   strings.ToLower(filepath.Ext(fileName)),
		FileSize:        int64(len(fileBytes)),
		MimeType:        mimeType,
		Hash:            fileHash,
		StorageProvider: "minio",
		StorageBucket:   s.config.BucketName,
		StoragePath:     storagePath,
		UploadStatus:    models.FileStatusUploaded.String(),
		UploadedBy:      uploadedBy,
		UploadedAt:      time.Now(),
		Metadata:        metadata,
	}

	// Сохраняем метаданные в БД
	if err := s.metadataRepo.Create(ctx, fileMetadata); err != nil {
		// Пытаемся удалить файл из хранилища в случае ошибки
		s.storageRepo.DeleteFile(ctx, s.config.BucketName, storagePath)
		return nil, fmt.Errorf("failed to save file metadata: %w", err)
	}

	// Генерируем URL для доступа к файлу
	storageURL := s.generateStorageURL(storagePath)

	s.logger.Info().
		Str("file_id", fileID).
		Str("original_name", fileName).
		Str("hash", fileHash).
		Int64("size", fileMetadata.FileSize).
		Str("mime_type", mimeType).
		Msg("File uploaded successfully")

	return &models.UploadFileResponse{
		FileID:     fileID,
		FileName:   uniqueFileName,
		FileSize:   fileMetadata.FileSize,
		Hash:       fileHash,
		MimeType:   mimeType,
		UploadedAt: fileMetadata.UploadedAt,
		StorageURL: storageURL,
		Metadata:   metadata,
	}, nil
}

func (s *uploadService) CheckDuplicate(ctx context.Context, fileHash string, fileSize int64) ([]*models.FileMetadata, error) {
	return s.metadataRepo.GetByHash(ctx, fileHash, fileSize)
}

func (s *uploadService) createDuplicateResponse(existingFile *models.FileMetadata) *models.UploadFileResponse {
	storageURL := s.generateStorageURL(existingFile.StoragePath)

	return &models.UploadFileResponse{
		FileID:     existingFile.ID,
		FileName:   existingFile.FileName,
		FileSize:   existingFile.FileSize,
		Hash:       existingFile.Hash,
		MimeType:   existingFile.MimeType,
		UploadedAt: existingFile.UploadedAt,
		StorageURL: storageURL,
		Metadata:   existingFile.Metadata,
	}
}

func (s *uploadService) detectMimeType(fileName string, fileBytes []byte) string {
	// Определяем MIME тип по расширению файла
	ext := strings.ToLower(filepath.Ext(fileName))

	mimeTypes := map[string]string{
		".txt":  "text/plain",
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".ppt":  "application/vnd.ms-powerpoint",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		".zip":  "application/zip",
		".rar":  "application/x-rar-compressed",
		".7z":   "application/x-7z-compressed",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".bmp":  "image/bmp",
		".mp3":  "audio/mpeg",
		".mp4":  "video/mp4",
		".avi":  "video/x-msvideo",
		".mov":  "video/quicktime",
	}

	if mimeType, ok := mimeTypes[ext]; ok {
		return mimeType
	}

	// По умолчанию возвращаем binary
	return "application/octet-stream"
}

func (s *uploadService) isAllowedType(mimeType, fileName string) bool {
	if len(s.config.AllowedTypes) == 0 {
		return true
	}

	ext := strings.ToLower(filepath.Ext(fileName))
	for _, allowed := range s.config.AllowedTypes {
		if strings.HasPrefix(allowed, ".") {
			// Проверка по расширению
			if ext == allowed {
				return true
			}
		} else {
			// Проверка по MIME типу
			if strings.HasPrefix(mimeType, allowed) {
				return true
			}
		}
	}

	return false
}

func (s *uploadService) generateUniqueFileName(originalName string) string {
	ext := filepath.Ext(originalName)
	name := strings.TrimSuffix(originalName, ext)

	// Удаляем небезопасные символы
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "..", "")

	timestamp := time.Now().UnixNano()
	uuid := uuid.New().String()[:8]

	return fmt.Sprintf("%s_%d_%s%s", name, timestamp, uuid, ext)
}

func (s *uploadService) generateStoragePath(fileName string) string {
	now := time.Now()
	return fmt.Sprintf("%d/%02d/%02d/%s", now.Year(), now.Month(), now.Day(), fileName)
}

func (s *uploadService) generateStorageURL(storagePath string) string {
	return fmt.Sprintf("/files/%s", storagePath)
}
