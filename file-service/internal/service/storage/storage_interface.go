package storage

import (
	"context"
	"io"
)

// StorageInterface определяет контракт для работы с различными хранилищами
type StorageInterface interface {
	// Основные операции
	Upload(ctx context.Context, bucket, key string, data io.Reader, size int64) error
	Download(ctx context.Context, bucket, key string) (io.ReadCloser, int64, error)
	Delete(ctx context.Context, bucket, key string) error
	Exists(ctx context.Context, bucket, key string) (bool, error)

	// Информация
	GetInfo(ctx context.Context, bucket, key string) (*FileInfo, error)
	GetURL(bucket, key string) string

	// Управление
	List(ctx context.Context, bucket, prefix string) ([]string, error)
	Copy(ctx context.Context, srcBucket, srcKey, dstBucket, dstKey string) error
	Move(ctx context.Context, srcBucket, srcKey, dstBucket, dstKey string) error

	// Права доступа
	GeneratePresignedURL(ctx context.Context, bucket, key string, expiresIn int64) (string, error)
	SetPublicAccess(ctx context.Context, bucket, key string, public bool) error
}

// FileInfo содержит информацию о файле в хранилище
type FileInfo struct {
	Key          string
	Size         int64
	ContentType  string
	LastModified int64
	ETag         string
	Metadata     map[string]string
}

// StorageConfig содержит конфигурацию хранилища
type StorageConfig struct {
	Provider  string
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Region    string
	UseSSL    bool
	Timeout   int
}

// StorageFactory создает экземпляры хранилищ
type StorageFactory interface {
	CreateStorage(config StorageConfig) (StorageInterface, error)
}
