package repository

import (
	"context"
	"database/sql"
	"github.com/rs/zerolog"

	"github.com/RubachokBoss/plagiarism-checker/work-service/internal/models"
)

type AssignmentRepository interface {
	Create(ctx context.Context, assignment *models.Assignment) error
	GetByID(ctx context.Context, id string) (*models.AssignmentWithStats, error)
	GetAll(ctx context.Context, limit, offset int) ([]models.AssignmentWithStats, int, error)
	Update(ctx context.Context, assignment *models.Assignment) error
	Delete(ctx context.Context, id string) error
	Exists(ctx context.Context, id string) (bool, error)
}

type assignmentRepository struct {
	*PostgresRepository
}

func NewAssignmentRepository(db *sql.DB, logger zerolog.Logger) AssignmentRepository {
	return &assignmentRepository{
		PostgresRepository: NewPostgresRepository(db, logger),
	}
}

func (r *assignmentRepository) Create(ctx context.Context, assignment *models.Assignment) error {
	query := `
		INSERT INTO assignments (id, title, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.ExecContext(ctx, query,
		assignment.ID,
		assignment.Title,
		assignment.Description,
		assignment.CreatedAt,
		assignment.UpdatedAt,
	)

	return err
}

func (r *assignmentRepository) GetByID(ctx context.Context, id string) (*models.AssignmentWithStats, error) {
	query := `
		SELECT 
			a.id, a.title, a.description, a.created_at, a.updated_at,
			COUNT(w.id) as total_works,
			COUNT(CASE WHEN w.status = 'analyzed' THEN 1 END) as analyzed_works,
			COUNT(CASE WHEN w.status IN ('uploaded', 'analyzing') THEN 1 END) as pending_works
		FROM assignments a
		LEFT JOIN works w ON a.id = w.assignment_id
		WHERE a.id = $1
		GROUP BY a.id
	`

	assignment := &models.AssignmentWithStats{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&assignment.ID,
		&assignment.Title,
		&assignment.Description,
		&assignment.CreatedAt,
		&assignment.UpdatedAt,
		&assignment.TotalWorks,
		&assignment.AnalyzedWorks,
		&assignment.PendingWorks,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return assignment, err
}

func (r *assignmentRepository) GetAll(ctx context.Context, limit, offset int) ([]models.AssignmentWithStats, int, error) {
	// Получаем общее количество
	countQuery := `SELECT COUNT(*) FROM assignments`
	var total int
	err := r.db.QueryRowContext(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Получаем задания со статистикой
	query := `
		SELECT 
			a.id, a.title, a.description, a.created_at, a.updated_at,
			COUNT(w.id) as total_works,
			COUNT(CASE WHEN w.status = 'analyzed' THEN 1 END) as analyzed_works,
			COUNT(CASE WHEN w.status IN ('uploaded', 'analyzing') THEN 1 END) as pending_works
		FROM assignments a
		LEFT JOIN works w ON a.id = w.assignment_id
		GROUP BY a.id
		ORDER BY a.created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var assignments []models.AssignmentWithStats
	for rows.Next() {
		var assignment models.AssignmentWithStats
		err := rows.Scan(
			&assignment.ID,
			&assignment.Title,
			&assignment.Description,
			&assignment.CreatedAt,
			&assignment.UpdatedAt,
			&assignment.TotalWorks,
			&assignment.AnalyzedWorks,
			&assignment.PendingWorks,
		)
		if err != nil {
			return nil, 0, err
		}
		assignments = append(assignments, assignment)
	}

	return assignments, total, nil
}

func (r *assignmentRepository) Update(ctx context.Context, assignment *models.Assignment) error {
	query := `
		UPDATE assignments
		SET title = $1, description = $2, updated_at = $3
		WHERE id = $4
	`

	_, err := r.db.ExecContext(ctx, query,
		assignment.Title,
		assignment.Description,
		assignment.UpdatedAt,
		assignment.ID,
	)

	return err
}

func (r *assignmentRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM assignments WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *assignmentRepository) Exists(ctx context.Context, id string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM assignments WHERE id = $1)`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, id).Scan(&exists)
	return exists, err
}
