package repository

import (
	"context"
	"database/sql"
	"github.com/rs/zerolog"
	"time"

	"github.com/RubachokBoss/plagiarism-checker/work-service/internal/models"
)

type WorkRepository interface {
	Create(ctx context.Context, work *models.Work) error
	GetByID(ctx context.Context, id string) (*models.Work, error)
	GetByStudentAndAssignment(ctx context.Context, studentID, assignmentID string) (*models.Work, error)
	GetByAssignmentID(ctx context.Context, assignmentID string, limit, offset int) ([]models.WorkWithDetails, int, error)
	GetByStudentID(ctx context.Context, studentID string, limit, offset int) ([]models.WorkWithDetails, int, error)
	GetAll(ctx context.Context, limit, offset int) ([]models.WorkWithDetails, int, error)
	UpdateStatus(ctx context.Context, id, status string) error
	UpdateFileID(ctx context.Context, id, fileID string) error
	Delete(ctx context.Context, id string) error
	GetPreviousWorks(ctx context.Context, assignmentID, excludeWorkID string) ([]models.Work, error)
}

type workRepository struct {
	*PostgresRepository
}

func NewWorkRepository(db *sql.DB, logger zerolog.Logger) WorkRepository {
	return &workRepository{
		PostgresRepository: NewPostgresRepository(db, logger),
	}
}

func (r *workRepository) Create(ctx context.Context, work *models.Work) error {
	query := `
		INSERT INTO works (id, student_id, assignment_id, file_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.ExecContext(ctx, query,
		work.ID,
		work.StudentID,
		work.AssignmentID,
		work.FileID,
		work.Status,
		work.CreatedAt,
		work.UpdatedAt,
	)

	return err
}

func (r *workRepository) GetByID(ctx context.Context, id string) (*models.Work, error) {
	query := `
		SELECT id, student_id, assignment_id, file_id, status, created_at, updated_at
		FROM works
		WHERE id = $1
	`

	work := &models.Work{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&work.ID,
		&work.StudentID,
		&work.AssignmentID,
		&work.FileID,
		&work.Status,
		&work.CreatedAt,
		&work.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return work, err
}

func (r *workRepository) GetByStudentAndAssignment(ctx context.Context, studentID, assignmentID string) (*models.Work, error) {
	query := `
		SELECT id, student_id, assignment_id, file_id, status, created_at, updated_at
		FROM works
		WHERE student_id = $1 AND assignment_id = $2
	`

	work := &models.Work{}
	err := r.db.QueryRowContext(ctx, query, studentID, assignmentID).Scan(
		&work.ID,
		&work.StudentID,
		&work.AssignmentID,
		&work.FileID,
		&work.Status,
		&work.CreatedAt,
		&work.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return work, err
}

func (r *workRepository) GetByAssignmentID(ctx context.Context, assignmentID string, limit, offset int) ([]models.WorkWithDetails, int, error) {
	countQuery := `SELECT COUNT(*) FROM works WHERE assignment_id = $1`
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, assignmentID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT 
			w.id, w.student_id, w.assignment_id, w.file_id, w.status, w.created_at, w.updated_at,
			s.name as student_name, s.email as student_email,
			a.title as assignment_title
		FROM works w
		JOIN students s ON w.student_id = s.id
		JOIN assignments a ON w.assignment_id = a.id
		WHERE w.assignment_id = $1
		ORDER BY w.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, assignmentID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var works []models.WorkWithDetails
	for rows.Next() {
		var work models.WorkWithDetails
		err := rows.Scan(
			&work.ID,
			&work.StudentID,
			&work.AssignmentID,
			&work.FileID,
			&work.Status,
			&work.CreatedAt,
			&work.UpdatedAt,
			&work.StudentName,
			&work.StudentEmail,
			&work.AssignmentTitle,
		)
		if err != nil {
			return nil, 0, err
		}
		works = append(works, work)
	}

	return works, total, nil
}

func (r *workRepository) GetByStudentID(ctx context.Context, studentID string, limit, offset int) ([]models.WorkWithDetails, int, error) {
	countQuery := `SELECT COUNT(*) FROM works WHERE student_id = $1`
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, studentID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT 
			w.id, w.student_id, w.assignment_id, w.file_id, w.status, w.created_at, w.updated_at,
			s.name as student_name, s.email as student_email,
			a.title as assignment_title
		FROM works w
		JOIN students s ON w.student_id = s.id
		JOIN assignments a ON w.assignment_id = a.id
		WHERE w.student_id = $1
		ORDER BY w.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, studentID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var works []models.WorkWithDetails
	for rows.Next() {
		var work models.WorkWithDetails
		err := rows.Scan(
			&work.ID,
			&work.StudentID,
			&work.AssignmentID,
			&work.FileID,
			&work.Status,
			&work.CreatedAt,
			&work.UpdatedAt,
			&work.StudentName,
			&work.StudentEmail,
			&work.AssignmentTitle,
		)
		if err != nil {
			return nil, 0, err
		}
		works = append(works, work)
	}

	return works, total, nil
}

func (r *workRepository) GetAll(ctx context.Context, limit, offset int) ([]models.WorkWithDetails, int, error) {
	countQuery := `SELECT COUNT(*) FROM works`
	var total int
	err := r.db.QueryRowContext(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT 
			w.id, w.student_id, w.assignment_id, w.file_id, w.status, w.created_at, w.updated_at,
			s.name as student_name, s.email as student_email,
			a.title as assignment_title
		FROM works w
		JOIN students s ON w.student_id = s.id
		JOIN assignments a ON w.assignment_id = a.id
		ORDER BY w.created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var works []models.WorkWithDetails
	for rows.Next() {
		var work models.WorkWithDetails
		err := rows.Scan(
			&work.ID,
			&work.StudentID,
			&work.AssignmentID,
			&work.FileID,
			&work.Status,
			&work.CreatedAt,
			&work.UpdatedAt,
			&work.StudentName,
			&work.StudentEmail,
			&work.AssignmentTitle,
		)
		if err != nil {
			return nil, 0, err
		}
		works = append(works, work)
	}

	return works, total, nil
}

func (r *workRepository) UpdateStatus(ctx context.Context, id, status string) error {
	query := `
		UPDATE works
		SET status = $1, updated_at = $2
		WHERE id = $3
	`

	_, err := r.db.ExecContext(ctx, query, status, time.Now(), id)
	return err
}

func (r *workRepository) UpdateFileID(ctx context.Context, id, fileID string) error {
	query := `
		UPDATE works
		SET file_id = $1, updated_at = $2
		WHERE id = $3
	`

	_, err := r.db.ExecContext(ctx, query, fileID, time.Now(), id)
	return err
}

func (r *workRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM works WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *workRepository) GetPreviousWorks(ctx context.Context, assignmentID, excludeWorkID string) ([]models.Work, error) {
	query := `
		SELECT id, student_id, assignment_id, file_id, status, created_at, updated_at
		FROM works
		WHERE assignment_id = $1 AND id != $2
		ORDER BY created_at
	`

	rows, err := r.db.QueryContext(ctx, query, assignmentID, excludeWorkID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var works []models.Work
	for rows.Next() {
		var work models.Work
		err := rows.Scan(
			&work.ID,
			&work.StudentID,
			&work.AssignmentID,
			&work.FileID,
			&work.Status,
			&work.CreatedAt,
			&work.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		works = append(works, work)
	}

	return works, nil
}
