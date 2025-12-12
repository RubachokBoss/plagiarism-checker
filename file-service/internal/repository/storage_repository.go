package repository

import (
	"context"
	"io"

	"github.com/RubachokBoss/plagiarism-checker/file-service/internal/models"
	"github.com/rs/zerolog"
)

type StorageRepository interface {
	UploadFile(ctx context.Context, bucket, fileName string, file io.Reader, size int64) error
	DownloadFile(ctx context.Context, bucket, fileName string) (io.ReadCloser, int64, error)
	DeleteFile(ctx context.Context, bucket, fileName string) error
	FileExists(ctx context.Context, bucket, fileName string) (bool, error)
	GetFileInfo(ctx context.Context, bucket, fileName string) (*models.FileInfoResponse, error)
	GetPresignedURL(ctx context.Context, bucket, fileName string, expiresIn int64) (string, error)
	ListFiles(ctx context.Context, bucket, prefix string) ([]string, error)
	GetBucketStats(ctx context.Context, bucket string) (*models.StorageInfo, error)
}

type storageRepository struct {
	provider StorageRepository
	logger   zerolog.Logger
}

func NewStorageRepository(provider StorageRepository, logger zerolog.Logger) StorageRepository {
	return &storageRepository{
		provider: provider,
		logger:   logger,
	}
}

func (r *storageRepository) UploadFile(ctx context.Context, bucket, fileName string, file io.Reader, size int64) error {
	return r.provider.UploadFile(ctx, bucket, fileName, file, size)
}

func (r *storageRepository) DownloadFile(ctx context.Context, bucket, fileName string) (io.ReadCloser, int64, error) {
	return r.provider.DownloadFile(ctx, bucket, fileName)
}

func (r *storageRepository) DeleteFile(ctx context.Context, bucket, fileName string) error {
	return r.provider.DeleteFile(ctx, bucket, fileName)
}

func (r *storageRepository) FileExists(ctx context.Context, bucket, fileName string) (bool, error) {
	return r.provider.FileExists(ctx, bucket, fileName)
}

func (r *storageRepository) GetFileInfo(ctx context.Context, bucket, fileName string) (*models.FileInfoResponse, error) {
	return r.provider.GetFileInfo(ctx, bucket, fileName)
}

func (r *storageRepository) GetPresignedURL(ctx context.Context, bucket, fileName string, expiresIn int64) (string, error) {
	return r.provider.GetPresignedURL(ctx, bucket, fileName, expiresIn)
}

func (r *storageRepository) ListFiles(ctx context.Context, bucket, prefix string) ([]string, error) {
	return r.provider.ListFiles(ctx, bucket, prefix)
}

func (r *storageRepository) GetBucketStats(ctx context.Context, bucket string) (*models.StorageInfo, error) {
	return r.provider.GetBucketStats(ctx, bucket)
}
