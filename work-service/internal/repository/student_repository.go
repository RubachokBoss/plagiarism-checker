package repository

import (
	"context"
	"database/sql"
	"github.com/rs/zerolog"

	"github.com/RubachokBoss/plagiarism-checker/work-service/internal/models"
)

type StudentRepository interface {
	Create(ctx context.Context, student *models.Student) error
	GetByID(ctx context.Context, id string) (*models.StudentWithStats, error)
	GetByEmail(ctx context.Context, email string) (*models.Student, error)
	GetAll(ctx context.Context, limit, offset int) ([]models.StudentWithStats, int, error)
	Update(ctx context.Context, student *models.Student) error
	Delete(ctx context.Context, id string) error
	Exists(ctx context.Context, id string) (bool, error)
}

type studentRepository struct {
	*PostgresRepository
}

func NewStudentRepository(db *sql.DB, logger zerolog.Logger) StudentRepository {
	return &studentRepository{
		PostgresRepository: NewPostgresRepository(db, logger),
	}
}

func (r *studentRepository) Create(ctx context.Context, student *models.Student) error {
	query := `
		INSERT INTO students (id, name, email, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.ExecContext(ctx, query,
		student.ID,
		student.Name,
		student.Email,
		student.CreatedAt,
		student.UpdatedAt,
	)

	return err
}

func (r *studentRepository) GetByID(ctx context.Context, id string) (*models.StudentWithStats, error) {
	query := `
		SELECT 
			s.id, s.name, s.email, s.created_at, s.updated_at,
			COUNT(w.id) as total_works,
			COUNT(CASE WHEN w.status = 'analyzed' THEN 1 END) as analyzed_works,
			COUNT(CASE WHEN w.status IN ('uploaded', 'analyzing') THEN 1 END) as pending_works
		FROM students s
		LEFT JOIN works w ON s.id = w.student_id
		WHERE s.id = $1
		GROUP BY s.id
	`

	student := &models.StudentWithStats{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&student.ID,
		&student.Name,
		&student.Email,
		&student.CreatedAt,
		&student.UpdatedAt,
		&student.TotalWorks,
		&student.AnalyzedWorks,
		&student.PendingWorks,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return student, err
}

func (r *studentRepository) GetByEmail(ctx context.Context, email string) (*models.Student, error) {
	query := `
		SELECT id, name, email, created_at, updated_at
		FROM students
		WHERE email = $1
	`

	student := &models.Student{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&student.ID,
		&student.Name,
		&student.Email,
		&student.CreatedAt,
		&student.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return student, err
}

func (r *studentRepository) GetAll(ctx context.Context, limit, offset int) ([]models.StudentWithStats, int, error) {
	// Получаем общее количество
	countQuery := `SELECT COUNT(*) FROM students`
	var total int
	err := r.db.QueryRowContext(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Получаем студентов со статистикой
	query := `
		SELECT 
			s.id, s.name, s.email, s.created_at, s.updated_at,
			COUNT(w.id) as total_works,
			COUNT(CASE WHEN w.status = 'analyzed' THEN 1 END) as analyzed_works,
			COUNT(CASE WHEN w.status IN ('uploaded', 'analyzing') THEN 1 END) as pending_works
		FROM students s
		LEFT JOIN works w ON s.id = w.student_id
		GROUP BY s.id
		ORDER BY s.created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var students []models.StudentWithStats
	for rows.Next() {
		var student models.StudentWithStats
		err := rows.Scan(
			&student.ID,
			&student.Name,
			&student.Email,
			&student.CreatedAt,
			&student.UpdatedAt,
			&student.TotalWorks,
			&student.AnalyzedWorks,
			&student.PendingWorks,
		)
		if err != nil {
			return nil, 0, err
		}
		students = append(students, student)
	}

	return students, total, nil
}

func (r *studentRepository) Update(ctx context.Context, student *models.Student) error {
	query := `
		UPDATE students
		SET name = $1, email = $2, updated_at = $3
		WHERE id = $4
	`

	_, err := r.db.ExecContext(ctx, query,
		student.Name,
		student.Email,
		student.UpdatedAt,
		student.ID,
	)

	return err
}

func (r *studentRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM students WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *studentRepository) Exists(ctx context.Context, id string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM students WHERE id = $1)`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, id).Scan(&exists)
	return exists, err
}
