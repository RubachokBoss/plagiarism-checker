package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOStorage struct {
	client *minio.Client
	bucket string
	region string
}

func NewMinIOStorage(config StorageConfig) (*MinIOStorage, error) {
	client, err := minio.New(config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKey, config.SecretKey, ""),
		Secure: config.UseSSL,
		Region: config.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Timeout)*time.Second)
	defer cancel()

	_, err = client.ListBuckets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MinIO: %w", err)
	}

	exists, err := client.BucketExists(ctx, config.Bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		err = client.MakeBucket(ctx, config.Bucket, minio.MakeBucketOptions{
			Region: config.Region,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return &MinIOStorage{
		client: client,
		bucket: config.Bucket,
		region: config.Region,
	}, nil
}

func (s *MinIOStorage) Upload(ctx context.Context, bucket, key string, data io.Reader, size int64) error {
	_, err := s.client.PutObject(ctx, bucket, key, data, size, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	return err
}

func (s *MinIOStorage) Download(ctx context.Context, bucket, key string) (io.ReadCloser, int64, error) {
	obj, err := s.client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, 0, err
	}

	stat, err := obj.Stat()
	if err != nil {
		obj.Close()
		return nil, 0, err
	}

	return obj, stat.Size, nil
}

func (s *MinIOStorage) Delete(ctx context.Context, bucket, key string) error {
	return s.client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{})
}

func (s *MinIOStorage) Exists(ctx context.Context, bucket, key string) (bool, error) {
	_, err := s.client.StatObject(ctx, bucket, key, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *MinIOStorage) GetInfo(ctx context.Context, bucket, key string) (*FileInfo, error) {
	obj, err := s.client.StatObject(ctx, bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return nil, err
	}

	return &FileInfo{
		Key:          key,
		Size:         obj.Size,
		ContentType:  obj.ContentType,
		LastModified: obj.LastModified.Unix(),
		ETag:         obj.ETag,
		Metadata:     obj.UserMetadata,
	}, nil
}

func (s *MinIOStorage) GetURL(bucket, key string) string {
	return fmt.Sprintf("%s/%s/%s", s.client.EndpointURL(), bucket, key)
}

func (s *MinIOStorage) List(ctx context.Context, bucket, prefix string) ([]string, error) {
	var keys []string

	objectCh := s.client.ListObjects(ctx, bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	for obj := range objectCh {
		if obj.Err != nil {
			return nil, obj.Err
		}
		keys = append(keys, obj.Key)
	}

	return keys, nil
}

func (s *MinIOStorage) Copy(ctx context.Context, srcBucket, srcKey, dstBucket, dstKey string) error {
	src := minio.CopySrcOptions{
		Bucket: srcBucket,
		Object: srcKey,
	}

	dst := minio.CopyDestOptions{
		Bucket: dstBucket,
		Object: dstKey,
	}

	_, err := s.client.CopyObject(ctx, dst, src)
	return err
}

func (s *MinIOStorage) Move(ctx context.Context, srcBucket, srcKey, dstBucket, dstKey string) error {
	if err := s.Copy(ctx, srcBucket, srcKey, dstBucket, dstKey); err != nil {
		return err
	}

	return s.Delete(ctx, srcBucket, srcKey)
}

func (s *MinIOStorage) GeneratePresignedURL(ctx context.Context, bucket, key string, expiresIn int64) (string, error) {
	url, err := s.client.PresignedGetObject(ctx, bucket, key, time.Duration(expiresIn)*time.Second, nil)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}

func (s *MinIOStorage) SetPublicAccess(ctx context.Context, bucket, key string, public bool) error {
	return nil
}
