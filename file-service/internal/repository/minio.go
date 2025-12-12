package repository

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sync"
	"time"

	"github.com/RubachokBoss/plagiarism-checker/file-service/internal/models"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog"
)

type MinIORepository struct {
	client *minio.Client
	bucket string
	region string
	logger zerolog.Logger

	ensureMu      sync.Mutex
	bucketEnsured bool
}

func NewMinIORepository(endpoint, accessKey, secretKey, bucket, region string, useSSL bool, connectTimeout time.Duration, logger zerolog.Logger) (*MinIORepository, error) {
	// Инициализация клиента MinIO
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	repo := &MinIORepository{
		client: client,
		bucket: bucket,
		region: region,
		logger: logger,
	}

	// Best-effort bootstrap: на старте НЕ валим весь сервис, если MinIO ещё не готов.
	// Это особенно важно при запуске всего docker-compose или при нехватке места на диске.
	if connectTimeout <= 0 {
		connectTimeout = 30 * time.Second
	}
	if connectTimeout < 2*time.Minute {
		connectTimeout = 2 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()
	if err := repo.ensureBucket(ctx); err != nil {
		logger.Error().Err(err).
			Str("endpoint", endpoint).
			Str("bucket", bucket).
			Msg("MinIO not ready during startup; file-service will keep running and retry on demand")
	}

	logger.Info().
		Str("endpoint", endpoint).
		Str("bucket", bucket).
		Bool("ssl", useSSL).
		Msg("Connected to MinIO")

	return repo, nil
}

func (r *MinIORepository) ensureBucket(ctx context.Context) error {
	r.ensureMu.Lock()
	defer r.ensureMu.Unlock()
	if r.bucketEnsured {
		return nil
	}

	// Если MinIO ещё не отвечает — ретраим до дедлайна ctx.
	backoff := 500 * time.Millisecond
	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("minio not ready: %w", err)
		}

		// API readiness check
		if _, err := r.client.ListBuckets(ctx); err != nil {
			time.Sleep(backoff)
			continue
		}

		exists, err := r.client.BucketExists(ctx, r.bucket)
		if err != nil {
			time.Sleep(backoff)
			continue
		}

		if !exists {
			if err := r.client.MakeBucket(ctx, r.bucket, minio.MakeBucketOptions{Region: r.region}); err != nil {
				time.Sleep(backoff)
				continue
			}
			r.logger.Info().Str("bucket", r.bucket).Msg("Created new bucket")
		}

		r.bucketEnsured = true
		return nil
	}
}

func (r *MinIORepository) UploadFile(ctx context.Context, bucket, fileName string, file io.Reader, size int64) error {
	if err := r.ensureBucket(ctx); err != nil {
		return err
	}
	// Загружаем файл
	uploadInfo, err := r.client.PutObject(ctx, bucket, fileName, file, size, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	r.logger.Debug().
		Str("bucket", bucket).
		Str("file", fileName).
		Str("etag", uploadInfo.ETag).
		Int64("size", size).
		Msg("File uploaded to MinIO")

	return nil
}

func (r *MinIORepository) DownloadFile(ctx context.Context, bucket, fileName string) (io.ReadCloser, int64, error) {
	if err := r.ensureBucket(ctx); err != nil {
		return nil, 0, err
	}
	// Получаем информацию о файле
	objInfo, err := r.client.StatObject(ctx, bucket, fileName, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return nil, 0, errors.New("file not found")
		}
		return nil, 0, fmt.Errorf("failed to stat file: %w", err)
	}

	// Скачиваем файл
	object, err := r.client.GetObject(ctx, bucket, fileName, minio.GetObjectOptions{})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get file: %w", err)
	}

	r.logger.Debug().
		Str("bucket", bucket).
		Str("file", fileName).
		Int64("size", objInfo.Size).
		Msg("File downloaded from MinIO")

	return object, objInfo.Size, nil
}

func (r *MinIORepository) DeleteFile(ctx context.Context, bucket, fileName string) error {
	if err := r.ensureBucket(ctx); err != nil {
		return err
	}
	// Удаляем файл
	err := r.client.RemoveObject(ctx, bucket, fileName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	r.logger.Debug().
		Str("bucket", bucket).
		Str("file", fileName).
		Msg("File deleted from MinIO")

	return nil
}

func (r *MinIORepository) FileExists(ctx context.Context, bucket, fileName string) (bool, error) {
	if err := r.ensureBucket(ctx); err != nil {
		return false, err
	}
	_, err := r.client.StatObject(ctx, bucket, fileName, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return false, nil
		}
		return false, fmt.Errorf("failed to check file existence: %w", err)
	}

	return true, nil
}

func (r *MinIORepository) GetFileInfo(ctx context.Context, bucket, fileName string) (*models.FileInfoResponse, error) {
	if err := r.ensureBucket(ctx); err != nil {
		return nil, err
	}
	objInfo, err := r.client.StatObject(ctx, bucket, fileName, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return nil, errors.New("file not found")
		}
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	return &models.FileInfoResponse{
		OriginalName: fileName,
		FileSize:     objInfo.Size,
		MimeType:     objInfo.ContentType,
	}, nil
}

func (r *MinIORepository) GetPresignedURL(ctx context.Context, bucket, fileName string, expiresIn int64) (string, error) {
	if err := r.ensureBucket(ctx); err != nil {
		return "", err
	}
	// Создаем предварительно подписанный URL
	url, err := r.client.PresignedGetObject(ctx, bucket, fileName, time.Duration(expiresIn)*time.Second, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return url.String(), nil
}

func (r *MinIORepository) ListFiles(ctx context.Context, bucket, prefix string) ([]string, error) {
	if err := r.ensureBucket(ctx); err != nil {
		return nil, err
	}
	var files []string

	// Получаем список объектов
	objectCh := r.client.ListObjects(ctx, bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	for object := range objectCh {
		if object.Err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", object.Err)
		}
		files = append(files, object.Key)
	}

	return files, nil
}

func (r *MinIORepository) GetBucketStats(ctx context.Context, bucket string) (*models.StorageInfo, error) {
	if err := r.ensureBucket(ctx); err != nil {
		return nil, err
	}
	var totalSize int64
	var fileCount int64

	// Получаем список всех объектов в бакете
	objectCh := r.client.ListObjects(ctx, bucket, minio.ListObjectsOptions{
		Recursive: true,
	})

	for object := range objectCh {
		if object.Err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", object.Err)
		}
		totalSize += object.Size
		fileCount++
	}

	return &models.StorageInfo{
		Provider:   "minio",
		BucketName: bucket,
		Region:     r.region,
		UsedSpace:  totalSize,
		FileCount:  fileCount,
	}, nil
}

// GenerateFileName генерирует уникальное имя файла
func (r *MinIORepository) GenerateFileName(originalName string) string {
	ext := filepath.Ext(originalName)
	name := filepath.Base(originalName)
	name = name[:len(name)-len(ext)]

	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("%s_%d%s", name, timestamp, ext)
}

// GenerateStoragePath генерирует путь для хранения файла
func (r *MinIORepository) GenerateStoragePath(fileName string) string {
	now := time.Now()
	return fmt.Sprintf("%d/%02d/%s", now.Year(), now.Month(), fileName)
}
