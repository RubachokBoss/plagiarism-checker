package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/RubachokBoss/plagiarism-checker/work-service/internal/models"
	"github.com/RubachokBoss/plagiarism-checker/work-service/internal/repository"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type StudentService interface {
	CreateStudent(ctx context.Context, req *models.CreateStudentRequest) (*models.Student, error)
	GetStudentByID(ctx context.Context, id string) (*models.StudentWithStats, error)
	GetStudentByEmail(ctx context.Context, email string) (*models.Student, error)
	GetAllStudents(ctx context.Context, page, limit int) ([]models.StudentWithStats, int, error)
	UpdateStudent(ctx context.Context, id string, req *models.CreateStudentRequest) error
	DeleteStudent(ctx context.Context, id string) error
}

type studentService struct {
	studentRepo repository.StudentRepository
	logger      zerolog.Logger
}

func NewStudentService(studentRepo repository.StudentRepository, logger zerolog.Logger) StudentService {
	return &studentService{
		studentRepo: studentRepo,
		logger:      logger,
	}
}

func (s *studentService) CreateStudent(ctx context.Context, req *models.CreateStudentRequest) (*models.Student, error) {
	existingStudent, err := s.studentRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing student: %w", err)
	}
	if existingStudent != nil {
		return nil, errors.New("student with this email already exists")
	}

	student := &models.Student{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Email:     req.Email,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.studentRepo.Create(ctx, student); err != nil {
		return nil, fmt.Errorf("failed to create student: %w", err)
	}

	s.logger.Info().
		Str("student_id", student.ID).
		Str("email", student.Email).
		Msg("Student created")

	return student, nil
}

func (s *studentService) GetStudentByID(ctx context.Context, id string) (*models.StudentWithStats, error) {
	student, err := s.studentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get student: %w", err)
	}
	if student == nil {
		return nil, errors.New("student not found")
	}

	return student, nil
}

func (s *studentService) GetStudentByEmail(ctx context.Context, email string) (*models.Student, error) {
	student, err := s.studentRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get student by email: %w", err)
	}
	if student == nil {
		return nil, errors.New("student not found")
	}

	return student, nil
}

func (s *studentService) GetAllStudents(ctx context.Context, page, limit int) ([]models.StudentWithStats, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	students, total, err := s.studentRepo.GetAll(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get all students: %w", err)
	}

	return students, total, nil
}

func (s *studentService) UpdateStudent(ctx context.Context, id string, req *models.CreateStudentRequest) error {
	student, err := s.studentRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get student: %w", err)
	}
	if student == nil {
		return errors.New("student not found")
	}

	if req.Email != student.Email {
		existingStudent, err := s.studentRepo.GetByEmail(ctx, req.Email)
		if err != nil {
			return fmt.Errorf("failed to check email availability: %w", err)
		}
		if existingStudent != nil {
			return errors.New("email already in use by another student")
		}
	}

	student.Name = req.Name
	student.Email = req.Email
	student.UpdatedAt = time.Now()

	return s.studentRepo.Update(ctx, &student.Student)
}

func (s *studentService) DeleteStudent(ctx context.Context, id string) error {
	student, err := s.studentRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get student: %w", err)
	}
	if student == nil {
		return errors.New("student not found")
	}

	if student.TotalWorks > 0 {
		return errors.New("cannot delete student with existing works")
	}

	return s.studentRepo.Delete(ctx, id)
}
