package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/models"
	"github.com/lib/pq"
)

type ReportRepository interface {
	Create(ctx context.Context, report *models.Report) error
	GetByID(ctx context.Context, id string) (*models.Report, error)
	GetByWorkID(ctx context.Context, workID string) (*models.Report, error)
	GetByAssignmentID(ctx context.Context, assignmentID string, limit, offset int) ([]models.Report, int, error)
	GetByStudentID(ctx context.Context, studentID string, limit, offset int) ([]models.Report, int, error)
	GetAll(ctx context.Context, limit, offset int) ([]models.Report, int, error)
	Update(ctx context.Context, report *models.Report) error
	UpdateStatus(ctx context.Context, id, status string) error
	UpdateResult(ctx context.Context, id string, plagiarismFlag bool, originalWorkID *string, matchPercentage int, details []byte) error
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]models.Report, int, error)
	GetStats(ctx context.Context) (*models.AnalysisStats, error)
	GetAssignmentStats(ctx context.Context, assignmentID string) (*models.AssignmentStats, error)
	GetStudentStats(ctx context.Context, studentID string) (*models.StudentStats, error)
	GetRecentReports(ctx context.Context, limit int) ([]models.Report, error)
	GetReportsByStatus(ctx context.Context, status string, limit int) ([]models.Report, error)
	Exists(ctx context.Context, workID string) (bool, error)
	Ping(ctx context.Context) error
}

type reportRepository struct {
	*PostgresRepository
}

func NewReportRepository(db *sql.DB, logger zerolog.Logger) ReportRepository {
	return &reportRepository{
		PostgresRepository: NewPostgresRepository(db, logger),
	}
}

func (r *reportRepository) Create(ctx context.Context, report *models.Report) error {
	query := `
		INSERT INTO reports (
			id, work_id, file_id, assignment_id, student_id, status,
			plagiarism_flag, original_work_id, match_percentage, file_hash,
			compared_hashes, details, processing_time_ms, compared_files_count,
			created_at, started_at, completed_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
		)
	`

	_, err := r.db.ExecContext(ctx, query,
		report.ID,
		report.WorkID,
		report.FileID,
		report.AssignmentID,
		report.StudentID,
		report.Status,
		report.PlagiarismFlag,
		report.OriginalWorkID,
		report.MatchPercentage,
		report.FileHash,
		pq.Array(report.ComparedHashes),
		report.Details,
		report.ProcessingTimeMs,
		report.ComparedFilesCount,
		report.CreatedAt,
		report.StartedAt,
		report.CompletedAt,
		report.UpdatedAt,
	)

	return err
}

func (r *reportRepository) GetByID(ctx context.Context, id string) (*models.Report, error) {
	query := `
		SELECT 
			id, work_id, file_id, assignment_id, student_id, status,
			plagiarism_flag, original_work_id, match_percentage, file_hash,
			compared_hashes, details, processing_time_ms, compared_files_count,
			created_at, started_at, completed_at, updated_at
		FROM reports
		WHERE id = $1
	`

	report := &models.Report{}
	var comparedHashes []sql.NullString
	var originalWorkID sql.NullString
	var processingTimeMs sql.NullInt64

	err := r.db.QueryRowContext(ctx, query, id).Scan(
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

	if err == sql.ErrNoRows {
		return nil, nil
	}
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

func (r *reportRepository) GetByWorkID(ctx context.Context, workID string) (*models.Report, error) {
	query := `
		SELECT 
			id, work_id, file_id, assignment_id, student_id, status,
			plagiarism_flag, original_work_id, match_percentage, file_hash,
			compared_hashes, details, processing_time_ms, compared_files_count,
			created_at, started_at, completed_at, updated_at
		FROM reports
		WHERE work_id = $1
	`

	report := &models.Report{}
	var comparedHashes []sql.NullString
	var originalWorkID sql.NullString
	var processingTimeMs sql.NullInt64

	err := r.db.QueryRowContext(ctx, query, workID).Scan(
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

	if err == sql.ErrNoRows {
		return nil, nil
	}
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

func (r *reportRepository) GetByAssignmentID(ctx context.Context, assignmentID string, limit, offset int) ([]models.Report, int, error) {
	// Get total count
	countQuery := `SELECT COUNT(*) FROM reports WHERE assignment_id = $1`
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, assignmentID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get reports
	query := `
		SELECT 
			id, work_id, file_id, assignment_id, student_id, status,
			plagiarism_flag, original_work_id, match_percentage, file_hash,
			compared_hashes, details, processing_time_ms, compared_files_count,
			created_at, started_at, completed_at, updated_at
		FROM reports
		WHERE assignment_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, assignmentID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var reports []models.Report
	for rows.Next() {
		report, err := r.scanReport(rows)
		if err != nil {
			return nil, 0, err
		}
		reports = append(reports, *report)
	}

	return reports, total, nil
}

func (r *reportRepository) GetByStudentID(ctx context.Context, studentID string, limit, offset int) ([]models.Report, int, error) {
	// Get total count
	countQuery := `SELECT COUNT(*) FROM reports WHERE student_id = $1`
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, studentID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get reports
	query := `
		SELECT 
			id, work_id, file_id, assignment_id, student_id, status,
			plagiarism_flag, original_work_id, match_percentage, file_hash,
			compared_hashes, details, processing_time_ms, compared_files_count,
			created_at, started_at, completed_at, updated_at
		FROM reports
		WHERE student_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, studentID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var reports []models.Report
	for rows.Next() {
		report, err := r.scanReport(rows)
		if err != nil {
			return nil, 0, err
		}
		reports = append(reports, *report)
	}

	return reports, total, nil
}

func (r *reportRepository) GetAll(ctx context.Context, limit, offset int) ([]models.Report, int, error) {
	// Get total count
	countQuery := `SELECT COUNT(*) FROM reports`
	var total int
	err := r.db.QueryRowContext(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get reports
	query := `
		SELECT 
			id, work_id, file_id, assignment_id, student_id, status,
			plagiarism_flag, original_work_id, match_percentage, file_hash,
			compared_hashes, details, processing_time_ms, compared_files_count,
			created_at, started_at, completed_at, updated_at
		FROM reports
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var reports []models.Report
	for rows.Next() {
		report, err := r.scanReport(rows)
		if err != nil {
			return nil, 0, err
		}
		reports = append(reports, *report)
	}

	return reports, total, nil
}

func (r *reportRepository) Update(ctx context.Context, report *models.Report) error {
	query := `
		UPDATE reports
		SET 
			status = $1,
			plagiarism_flag = $2,
			original_work_id = $3,
			match_percentage = $4,
			file_hash = $5,
			compared_hashes = $6,
			details = $7,
			processing_time_ms = $8,
			compared_files_count = $9,
			started_at = $10,
			completed_at = $11,
			updated_at = $12
		WHERE id = $13
	`

	_, err := r.db.ExecContext(ctx, query,
		report.Status,
		report.PlagiarismFlag,
		report.OriginalWorkID,
		report.MatchPercentage,
		report.FileHash,
		pq.Array(report.ComparedHashes),
		report.Details,
		report.ProcessingTimeMs,
		report.ComparedFilesCount,
		report.StartedAt,
		report.CompletedAt,
		report.UpdatedAt,
		report.ID,
	)

	return err
}

func (r *reportRepository) UpdateStatus(ctx context.Context, id, status string) error {
	query := `
		UPDATE reports
		SET status = $1, updated_at = $2
		WHERE id = $3
	`

	_, err := r.db.ExecContext(ctx, query, status, time.Now(), id)
	return err
}

func (r *reportRepository) UpdateResult(ctx context.Context, id string, plagiarismFlag bool, originalWorkID *string, matchPercentage int, details []byte) error {
	query := `
		UPDATE reports
		SET 
			plagiarism_flag = $1,
			original_work_id = $2,
			match_percentage = $3,
			details = $4,
			status = 'completed',
			completed_at = $5,
			updated_at = $6
		WHERE id = $7
	`

	_, err := r.db.ExecContext(ctx, query,
		plagiarismFlag,
		originalWorkID,
		matchPercentage,
		details,
		time.Now(),
		time.Now(),
		id,
	)

	return err
}

func (r *reportRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM reports WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *reportRepository) Search(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]models.Report, int, error) {
	// Build WHERE clause
	whereClauses := []string{}
	args := []interface{}{}
	argCount := 1

	for key, value := range filters {
		if value != nil {
			switch key {
			case "work_id", "assignment_id", "student_id", "status":
				whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", key, argCount))
				args = append(args, value)
				argCount++
			case "plagiarism_flag":
				whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", key, argCount))
				args = append(args, value)
				argCount++
			case "date_from":
				whereClauses = append(whereClauses, fmt.Sprintf("created_at >= $%d", argCount))
				args = append(args, value)
				argCount++
			case "date_to":
				whereClauses = append(whereClauses, fmt.Sprintf("created_at <= $%d", argCount))
				args = append(args, value)
				argCount++
			}
		}
	}

	// Build query
	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM reports %s", whereSQL)
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get reports
	query := fmt.Sprintf(`
		SELECT 
			id, work_id, file_id, assignment_id, student_id, status,
			plagiarism_flag, original_work_id, match_percentage, file_hash,
			compared_hashes, details, processing_time_ms, compared_files_count,
			created_at, started_at, completed_at, updated_at
		FROM reports
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argCount, argCount+1)

	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var reports []models.Report
	for rows.Next() {
		report, err := r.scanReport(rows)
		if err != nil {
			return nil, 0, err
		}
		reports = append(reports, *report)
	}

	return reports, total, nil
}

func (r *reportRepository) GetStats(ctx context.Context) (*models.AnalysisStats, error) {
	stats := &models.AnalysisStats{}

	// Basic stats
	query := `
		SELECT 
			COUNT(*) as total_reports,
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed_reports,
			COUNT(CASE WHEN status IN ('pending', 'processing') THEN 1 END) as pending_reports,
			COUNT(CASE WHEN plagiarism_flag = TRUE THEN 1 END) as plagiarized_works,
			COALESCE(AVG(processing_time_ms), 0) as avg_processing_time
		FROM reports
	`

	err := r.db.QueryRowContext(ctx, query).Scan(
		&stats.TotalReports,
		&stats.CompletedReports,
		&stats.PendingReports,
		&stats.PlagiarizedWorks,
		&stats.AvgProcessingTime,
	)
	if err != nil {
		return nil, err
	}

	// Top assignments
	assignmentQuery := `
		SELECT 
			assignment_id,
			total_works,
			analyzed_works,
			plagiarized_works,
			avg_match_percentage
		FROM assignment_stats
		ORDER BY total_works DESC
		LIMIT 10
	`

	rows, err := r.db.QueryContext(ctx, assignmentQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var stat models.AssignmentStats
		err := rows.Scan(
			&stat.AssignmentID,
			&stat.TotalWorks,
			&stat.AnalyzedWorks,
			&stat.PlagiarizedWorks,
			&stat.AvgMatchPercentage,
		)
		if err != nil {
			return nil, err
		}
		stats.TopAssignments = append(stats.TopAssignments, stat)
	}

	// Top students
	studentQuery := `
		SELECT 
			student_id,
			total_works,
			analyzed_works,
			plagiarized_works,
			avg_match_percentage
		FROM student_stats
		ORDER BY total_works DESC
		LIMIT 10
	`

	rows, err = r.db.QueryContext(ctx, studentQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var stat models.StudentStats
		err := rows.Scan(
			&stat.StudentID,
			&stat.TotalWorks,
			&stat.AnalyzedWorks,
			&stat.PlagiarizedWorks,
			&stat.AvgMatchPercentage,
		)
		if err != nil {
			return nil, err
		}
		stats.TopStudents = append(stats.TopStudents, stat)
	}

	// Recent activity
	recentQuery := `
		SELECT 
			id, work_id, file_id, assignment_id, student_id, status,
			plagiarism_flag, original_work_id, match_percentage, file_hash,
			compared_hashes, details, processing_time_ms, compared_files_count,
			created_at, started_at, completed_at, updated_at
		FROM reports
		ORDER BY created_at DESC
		LIMIT 10
	`

	rows, err = r.db.QueryContext(ctx, recentQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		report, err := r.scanReport(rows)
		if err != nil {
			return nil, err
		}
		stats.RecentActivity = append(stats.RecentActivity, *report)
	}

	return stats, nil
}

func (r *reportRepository) GetAssignmentStats(ctx context.Context, assignmentID string) (*models.AssignmentStats, error) {
	query := `
		SELECT 
			assignment_id,
			total_works,
			analyzed_works,
			plagiarized_works,
			avg_match_percentage,
			last_analyzed_at,
			updated_at
		FROM assignment_stats
		WHERE assignment_id = $1
	`

	stats := &models.AssignmentStats{}
	err := r.db.QueryRowContext(ctx, query, assignmentID).Scan(
		&stats.AssignmentID,
		&stats.TotalWorks,
		&stats.AnalyzedWorks,
		&stats.PlagiarizedWorks,
		&stats.AvgMatchPercentage,
		&stats.LastAnalyzedAt,
		&stats.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return stats, err
}

func (r *reportRepository) GetStudentStats(ctx context.Context, studentID string) (*models.StudentStats, error) {
	query := `
		SELECT 
			student_id,
			total_works,
			analyzed_works,
			plagiarized_works,
			avg_match_percentage,
			last_analyzed_at,
			updated_at
		FROM student_stats
		WHERE student_id = $1
	`

	stats := &models.StudentStats{}
	err := r.db.QueryRowContext(ctx, query, studentID).Scan(
		&stats.StudentID,
		&stats.TotalWorks,
		&stats.AnalyzedWorks,
		&stats.PlagiarizedWorks,
		&stats.AvgMatchPercentage,
		&stats.LastAnalyzedAt,
		&stats.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return stats, err
}

func (r *reportRepository) GetRecentReports(ctx context.Context, limit int) ([]models.Report, error) {
	query := `
		SELECT 
			id, work_id, file_id, assignment_id, student_id, status,
			plagiarism_flag, original_work_id, match_percentage, file_hash,
			compared_hashes, details, processing_time_ms, compared_files_count,
			created_at, started_at, completed_at, updated_at
		FROM reports
		ORDER BY created_at DESC
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

func (r *reportRepository) GetReportsByStatus(ctx context.Context, status string, limit int) ([]models.Report, error) {
	query := `
		SELECT 
			id, work_id, file_id, assignment_id, student_id, status,
			plagiarism_flag, original_work_id, match_percentage, file_hash,
			compared_hashes, details, processing_time_ms, compared_files_count,
			created_at, started_at, completed_at, updated_at
		FROM reports
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, status, limit)
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

func (r *reportRepository) Exists(ctx context.Context, workID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM reports WHERE work_id = $1)`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, workID).Scan(&exists)
	return exists, err
}

func (r *reportRepository) Ping(ctx context.Context) error {
	return r.PostgresRepository.Ping(ctx)
}

func (r *reportRepository) scanReport(rows *sql.Rows) (*models.Report, error) {
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
