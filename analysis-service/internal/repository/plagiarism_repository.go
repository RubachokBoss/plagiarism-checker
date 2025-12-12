package repository

import (
	"context"
	"database/sql"

	"github.com/rs/zerolog"

	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/models"
	"github.com/lib/pq"
)

type PlagiarismRepository interface {
	FindSimilarWorks(ctx context.Context, fileHash string, assignmentID, excludeWorkID string) ([]models.SimilarWork, error)
	GetWorksByAssignment(ctx context.Context, assignmentID string, excludeWorkID string) ([]models.SimilarWork, error)
	GetFileHashesByAssignment(ctx context.Context, assignmentID string) (map[string]string, error) // file_id -> hash
	SaveComparisonResult(ctx context.Context, workID string, comparedWith []string, results []models.ComparisonResult) error
	GetComparisonHistory(ctx context.Context, workID string) ([]models.ComparisonResult, error)
	GetTopPlagiarizedWorks(ctx context.Context, limit int) ([]models.Report, error)
	GetPlagiarismPatterns(ctx context.Context, assignmentID string) ([]models.ComparisonResult, error)
}

type plagiarismRepository struct {
	*PostgresRepository
}

func NewPlagiarismRepository(db *sql.DB, logger zerolog.Logger) PlagiarismRepository {
	return &plagiarismRepository{
		PostgresRepository: NewPostgresRepository(db, logger),
	}
}

func (r *plagiarismRepository) FindSimilarWorks(ctx context.Context, fileHash string, assignmentID, excludeWorkID string) ([]models.SimilarWork, error) {
	query := `
		SELECT 
			r.work_id,
			r.student_id,
			r.match_percentage,
			r.file_hash,
			r.created_at
		FROM reports r
		WHERE r.assignment_id = $1
			AND r.work_id != $2
			AND r.file_hash = $3
			AND r.status = 'completed'
		ORDER BY r.match_percentage DESC, r.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, assignmentID, excludeWorkID, fileHash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var works []models.SimilarWork
	for rows.Next() {
		var work models.SimilarWork
		err := rows.Scan(
			&work.WorkID,
			&work.StudentID,
			&work.MatchPercentage,
			&work.FileHash,
			&work.SubmittedAt,
		)
		if err != nil {
			return nil, err
		}
		works = append(works, work)
	}

	return works, nil
}

func (r *plagiarismRepository) GetWorksByAssignment(ctx context.Context, assignmentID string, excludeWorkID string) ([]models.SimilarWork, error) {
	query := `
		SELECT 
			r.work_id,
			r.student_id,
			r.file_hash,
			r.created_at
		FROM reports r
		WHERE r.assignment_id = $1
			AND r.work_id != $2
			AND r.status = 'completed'
		ORDER BY r.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, assignmentID, excludeWorkID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var works []models.SimilarWork
	for rows.Next() {
		var work models.SimilarWork
		err := rows.Scan(
			&work.WorkID,
			&work.StudentID,
			&work.FileHash,
			&work.SubmittedAt,
		)
		if err != nil {
			return nil, err
		}
		works = append(works, work)
	}

	return works, nil
}

func (r *plagiarismRepository) GetFileHashesByAssignment(ctx context.Context, assignmentID string) (map[string]string, error) {
	query := `
		SELECT work_id, file_hash
		FROM reports
		WHERE assignment_id = $1
			AND status = 'completed'
			AND file_hash IS NOT NULL
	`

	rows, err := r.db.QueryContext(ctx, query, assignmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	hashes := make(map[string]string)
	for rows.Next() {
		var workID, fileHash string
		err := rows.Scan(&workID, &fileHash)
		if err != nil {
			return nil, err
		}
		hashes[workID] = fileHash
	}

	return hashes, nil
}

func (r *plagiarismRepository) SaveComparisonResult(ctx context.Context, workID string, comparedWith []string, results []models.ComparisonResult) error {
	// Start transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update compared_hashes in reports table
	updateQuery := `
		UPDATE reports
		SET compared_hashes = $1, compared_files_count = $2
		WHERE work_id = $3
	`

	_, err = tx.ExecContext(ctx, updateQuery, pq.Array(comparedWith), len(comparedWith), workID)
	if err != nil {
		return err
	}

	// Save detailed comparison results (if needed)
	// This could be saved in a separate table for detailed analysis
	// For now, we'll just update the reports table

	return tx.Commit()
}

func (r *plagiarismRepository) GetComparisonHistory(ctx context.Context, workID string) ([]models.ComparisonResult, error) {
	query := `
		SELECT 
			details->'comparison_results'
		FROM reports
		WHERE work_id = $1
			AND details IS NOT NULL
	`

	var resultsJSON []byte
	err := r.db.QueryRowContext(ctx, query, workID).Scan(&resultsJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return []models.ComparisonResult{}, nil
		}
		return nil, err
	}

	// Parse JSON results
	// This is simplified - in real implementation you'd parse the JSON
	// For now, return empty slice
	return []models.ComparisonResult{}, nil
}

func (r *plagiarismRepository) GetTopPlagiarizedWorks(ctx context.Context, limit int) ([]models.Report, error) {
	query := `
		SELECT 
			id, work_id, file_id, assignment_id, student_id, status,
			plagiarism_flag, original_work_id, match_percentage, file_hash,
			compared_hashes, details, processing_time_ms, compared_files_count,
			created_at, started_at, completed_at, updated_at
		FROM reports
		WHERE plagiarism_flag = TRUE
		ORDER BY match_percentage DESC, created_at DESC
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []models.Report
	for rows.Next() {
		report, err := r.scanReport(rows)
		if err != nil {
			return nil, err
		}
		reports = append(reports, *report)
	}

	return reports, nil
}

func (r *plagiarismRepository) GetPlagiarismPatterns(ctx context.Context, assignmentID string) ([]models.ComparisonResult, error) {
	// This would analyze patterns of plagiarism within an assignment
	// For now, return empty results
	return []models.ComparisonResult{}, nil
}

func (r *plagiarismRepository) scanReport(rows *sql.Rows) (*models.Report, error) {
	report := &models.Report{}
	var comparedHashes []sql.NullString
	var originalWorkID sql.NullString
	var processingTimeMs sql.NullInt64

	err := rows.Scan(
		&report.ID,
		&report.WorkID,
		&report.FileID,
		&report.AssignmentID,
		&report.StudentID,
		&report.Status,
		&report.PlagiarismFlag,
		&originalWorkID,
		&report.MatchPercentage,
		&report.FileHash,
		pq.Array(&comparedHashes),
		&report.Details,
		&processingTimeMs,
		&report.ComparedFilesCount,
		&report.CreatedAt,
		&report.StartedAt,
		&report.CompletedAt,
		&report.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	// Convert nullable fields
	if originalWorkID.Valid {
		report.OriginalWorkID = &originalWorkID.String
	}

	if processingTimeMs.Valid {
		timeMs := int(processingTimeMs.Int64)
		report.ProcessingTimeMs = &timeMs
	}

	// Convert compared hashes
	for _, hash := range comparedHashes {
		if hash.Valid {
			report.ComparedHashes = append(report.ComparedHashes, hash.String)
		}
	}

	return report, nil
}
