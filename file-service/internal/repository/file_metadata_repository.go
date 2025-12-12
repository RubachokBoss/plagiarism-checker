package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/rs/zerolog"
	"time"

	"github.com/plagiarism-checker/file-service/internal/models"
)

type FileMetadataRepository interface {
	Create(ctx context.Context, metadata *models.FileMetadata) error
	GetByID(ctx context.Context, id string) (*models.FileMetadata, error)
	GetByHash(ctx context.Context, hash string, fileSize int64) ([]*models.FileMetadata, error)
	GetByFileName(ctx context.Context, fileName string) (*models.FileMetadata, error)
	GetAll(ctx context.Context, limit, offset int, status string) ([]*models.FileMetadata, int, error)
	UpdateStatus(ctx context.Context, id, status string) error
	UpdateAccessInfo(ctx context.Context, id string) error
	UpdateMetadata(ctx context.Context, id string, metadata []byte) error
	Delete(ctx context.Context, id string) error
	SoftDelete(ctx context.Context, id string) error
	GetStats(ctx context.Context) (*models.FileStats, error)
	Exists(ctx context.Context, id string) (bool, error)
	SearchByMetadata(ctx context.Context, key, value string) ([]*models.FileMetadata, error)
}

type fileMetadataRepository struct {
	*PostgresRepository
}

func NewFileMetadataRepository(db *sql.DB, logger zerolog.Logger) FileMetadataRepository {
	return &fileMetadataRepository{
		PostgresRepository: NewPostgresRepository(db, logger),
	}
}

func (r *fileMetadataRepository) Create(ctx context.Context, metadata *models.FileMetadata) error {
	query := `
		INSERT INTO file_metadata (
			id, original_name, file_name, file_extension, file_size, mime_type,
			hash, storage_provider, storage_bucket, storage_path, storage_url,
			upload_status, uploaded_by, uploaded_at, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
		)
	`

	_, err := r.db.ExecContext(ctx, query,
		metadata.ID,
		metadata.OriginalName,
		metadata.FileName,
		metadata.FileExtension,
		metadata.FileSize,
		metadata.MimeType,
		metadata.Hash,
		metadata.StorageProvider,
		metadata.StorageBucket,
		metadata.StoragePath,
		metadata.StorageURL,
		metadata.UploadStatus,
		metadata.UploadedBy,
		metadata.UploadedAt,
		metadata.Metadata,
	)

	return err
}

func (r *fileMetadataRepository) GetByID(ctx context.Context, id string) (*models.FileMetadata, error) {
	query := `
		SELECT 
			id, original_name, file_name, file_extension, file_size, mime_type,
			hash, storage_provider, storage_bucket, storage_path, storage_url,
			upload_status, uploaded_by, uploaded_at, access_count, 
			last_accessed_at, metadata
		FROM file_metadata
		WHERE id = $1 AND upload_status != 'deleted'
	`

	metadata := &models.FileMetadata{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&metadata.ID,
		&metadata.OriginalName,
		&metadata.FileName,
		&metadata.FileExtension,
		&metadata.FileSize,
		&metadata.MimeType,
		&metadata.Hash,
		&metadata.StorageProvider,
		&metadata.StorageBucket,
		&metadata.StoragePath,
		&metadata.StorageURL,
		&metadata.UploadStatus,
		&metadata.UploadedBy,
		&metadata.UploadedAt,
		&metadata.AccessCount,
		&metadata.LastAccessedAt,
		&metadata.Metadata,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return metadata, err
}

func (r *fileMetadataRepository) GetByHash(ctx context.Context, hash string, fileSize int64) ([]*models.FileMetadata, error) {
	query := `
		SELECT 
			id, original_name, file_name, file_extension, file_size, mime_type,
			hash, storage_provider, storage_bucket, storage_path, storage_url,
			upload_status, uploaded_by, uploaded_at, access_count, 
			last_accessed_at, metadata
		FROM file_metadata
		WHERE hash = $1 AND file_size = $2 AND upload_status != 'deleted'
		ORDER BY uploaded_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, hash, fileSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*models.FileMetadata
	for rows.Next() {
		metadata := &models.FileMetadata{}
		err := rows.Scan(
			&metadata.ID,
			&metadata.OriginalName,
			&metadata.FileName,
			&metadata.FileExtension,
			&metadata.FileSize,
			&metadata.MimeType,
			&metadata.Hash,
			&metadata.StorageProvider,
			&metadata.StorageBucket,
			&metadata.StoragePath,
			&metadata.StorageURL,
			&metadata.UploadStatus,
			&metadata.UploadedBy,
			&metadata.UploadedAt,
			&metadata.AccessCount,
			&metadata.LastAccessedAt,
			&metadata.Metadata,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, metadata)
	}

	return files, nil
}

func (r *fileMetadataRepository) GetByFileName(ctx context.Context, fileName string) (*models.FileMetadata, error) {
	query := `
		SELECT 
			id, original_name, file_name, file_extension, file_size, mime_type,
			hash, storage_provider, storage_bucket, storage_path, storage_url,
			upload_status, uploaded_by, uploaded_at, access_count, 
			last_accessed_at, metadata
		FROM file_metadata
		WHERE file_name = $1 AND upload_status != 'deleted'
	`

	metadata := &models.FileMetadata{}
	err := r.db.QueryRowContext(ctx, query, fileName).Scan(
		&metadata.ID,
		&metadata.OriginalName,
		&metadata.FileName,
		&metadata.FileExtension,
		&metadata.FileSize,
		&metadata.MimeType,
		&metadata.Hash,
		&metadata.StorageProvider,
		&metadata.StorageBucket,
		&metadata.StoragePath,
		&metadata.StorageURL,
		&metadata.UploadStatus,
		&metadata.UploadedBy,
		&metadata.UploadedAt,
		&metadata.AccessCount,
		&metadata.LastAccessedAt,
		&metadata.Metadata,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return metadata, err
}

func (r *fileMetadataRepository) GetAll(ctx context.Context, limit, offset int, status string) ([]*models.FileMetadata, int, error) {
	// Получаем общее количество
	countQuery := `SELECT COUNT(*) FROM file_metadata WHERE upload_status != 'deleted'`
	var countArgs []interface{}

	if status != "" {
		countQuery += ` AND upload_status = $1`
		countArgs = append(countArgs, status)
	}

	var total int
	err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Получаем файлы
	query := `
		SELECT 
			id, original_name, file_name, file_extension, file_size, mime_type,
			hash, storage_provider, storage_bucket, storage_path, storage_url,
			upload_status, uploaded_by, uploaded_at, access_count, 
			last_accessed_at, metadata
		FROM file_metadata
		WHERE upload_status != 'deleted'
	`

	var queryArgs []interface{}
	argCount := 1

	if status != "" {
		query += ` AND upload_status = $1`
		queryArgs = append(queryArgs, status)
		argCount++
	}

	query += ` ORDER BY uploaded_at DESC LIMIT $` + fmt.Sprint(argCount) + ` OFFSET $` + fmt.Sprint(argCount+1)
	queryArgs = append(queryArgs, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var files []*models.FileMetadata
	for rows.Next() {
		metadata := &models.FileMetadata{}
		err := rows.Scan(
			&metadata.ID,
			&metadata.OriginalName,
			&metadata.FileName,
			&metadata.FileExtension,
			&metadata.FileSize,
			&metadata.MimeType,
			&metadata.Hash,
			&metadata.StorageProvider,
			&metadata.StorageBucket,
			&metadata.StoragePath,
			&metadata.StorageURL,
			&metadata.UploadStatus,
			&metadata.UploadedBy,
			&metadata.UploadedAt,
			&metadata.AccessCount,
			&metadata.LastAccessedAt,
			&metadata.Metadata,
		)
		if err != nil {
			return nil, 0, err
		}
		files = append(files, metadata)
	}

	return files, total, nil
}

func (r *fileMetadataRepository) UpdateStatus(ctx context.Context, id, status string) error {
	query := `
		UPDATE file_metadata
		SET upload_status = $1
		WHERE id = $2
	`

	_, err := r.db.ExecContext(ctx, query, status, id)
	return err
}

func (r *fileMetadataRepository) UpdateAccessInfo(ctx context.Context, id string) error {
	query := `
		UPDATE file_metadata
		SET access_count = access_count + 1, last_accessed_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *fileMetadataRepository) UpdateMetadata(ctx context.Context, id string, metadata []byte) error {
	query := `
		UPDATE file_metadata
		SET metadata = $1
		WHERE id = $2
	`

	_, err := r.db.ExecContext(ctx, query, metadata, id)
	return err
}

func (r *fileMetadataRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM file_metadata WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *fileMetadataRepository) SoftDelete(ctx context.Context, id string) error {
	query := `
		UPDATE file_metadata
		SET upload_status = 'deleted'
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *fileMetadataRepository) GetStats(ctx context.Context) (*models.FileStats, error) {
	stats := &models.FileStats{}

	// Общая статистика
	totalQuery := `
		SELECT 
			COUNT(*) as total_files,
			COALESCE(SUM(file_size), 0) as total_size,
			COALESCE(AVG(file_size), 0) as avg_size
		FROM file_metadata
		WHERE upload_status != 'deleted'
	`

	err := r.db.QueryRowContext(ctx, totalQuery).Scan(
		&stats.TotalFiles,
		&stats.TotalSize,
		&stats.AverageFileSize,
	)
	if err != nil {
		return nil, err
	}

	// Файлы за сегодня
	todayQuery := `
		SELECT COUNT(*)
		FROM file_metadata
		WHERE upload_status != 'deleted' 
		AND DATE(uploaded_at) = CURRENT_DATE
	`

	err = r.db.QueryRowContext(ctx, todayQuery).Scan(&stats.UploadedToday)
	if err != nil {
		return nil, err
	}

	// Топ расширений файлов
	extQuery := `
		SELECT 
			file_extension,
			COUNT(*) as count,
			SUM(file_size) as total_size
		FROM file_metadata
		WHERE upload_status != 'deleted'
		GROUP BY file_extension
		ORDER BY count DESC
		LIMIT 10
	`

	rows, err := r.db.QueryContext(ctx, extQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var extStat models.FileExtensionStat
		err := rows.Scan(&extStat.Extension, &extStat.Count, &extStat.TotalSize)
		if err != nil {
			return nil, err
		}
		stats.TopExtensions = append(stats.TopExtensions, extStat)
	}

	return stats, nil
}

func (r *fileMetadataRepository) Exists(ctx context.Context, id string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM file_metadata WHERE id = $1 AND upload_status != 'deleted')`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, id).Scan(&exists)
	return exists, err
}

func (r *fileMetadataRepository) SearchByMetadata(ctx context.Context, key, value string) ([]*models.FileMetadata, error) {
	query := `
		SELECT 
			id, original_name, file_name, file_extension, file_size, mime_type,
			hash, storage_provider, storage_bucket, storage_path, storage_url,
			upload_status, uploaded_by, uploaded_at, access_count, 
			last_accessed_at, metadata
		FROM file_metadata
		WHERE upload_status != 'deleted' 
		AND metadata->>$1 = $2
		ORDER BY uploaded_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, key, value)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*models.FileMetadata
	for rows.Next() {
		metadata := &models.FileMetadata{}
		err := rows.Scan(
			&metadata.ID,
			&metadata.OriginalName,
			&metadata.FileName,
			&metadata.FileExtension,
			&metadata.FileSize,
			&metadata.MimeType,
			&metadata.Hash,
			&metadata.StorageProvider,
			&metadata.StorageBucket,
			&metadata.StoragePath,
			&metadata.StorageURL,
			&metadata.UploadStatus,
			&metadata.UploadedBy,
			&metadata.UploadedAt,
			&metadata.AccessCount,
			&metadata.LastAccessedAt,
			&metadata.Metadata,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, metadata)
	}

	return files, nil
}
