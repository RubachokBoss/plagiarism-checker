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

type AssignmentService interface {
	CreateAssignment(ctx context.Context, req *models.CreateAssignmentRequest) (*models.Assignment, error)
	GetAssignmentByID(ctx context.Context, id string) (*models.AssignmentWithStats, error)
	GetAllAssignments(ctx context.Context, page, limit int) ([]models.AssignmentWithStats, int, error)
	UpdateAssignment(ctx context.Context, id string, req *models.CreateAssignmentRequest) error
	DeleteAssignment(ctx context.Context, id string) error
}

type assignmentService struct {
	assignmentRepo repository.AssignmentRepository
	logger         zerolog.Logger
}

func NewAssignmentService(assignmentRepo repository.AssignmentRepository, logger zerolog.Logger) AssignmentService {
	return &assignmentService{
		assignmentRepo: assignmentRepo,
		logger:         logger,
	}
}

func (s *assignmentService) CreateAssignment(ctx context.Context, req *models.CreateAssignmentRequest) (*models.Assignment, error) {
	assignment := &models.Assignment{
		ID:          uuid.New().String(),
		Title:       req.Title,
		Description: req.Description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.assignmentRepo.Create(ctx, assignment); err != nil {
		return nil, fmt.Errorf("failed to create assignment: %w", err)
	}

	s.logger.Info().
		Str("assignment_id", assignment.ID).
		Str("title", assignment.Title).
		Msg("Assignment created")

	return assignment, nil
}

func (s *assignmentService) GetAssignmentByID(ctx context.Context, id string) (*models.AssignmentWithStats, error) {
	assignment, err := s.assignmentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignment: %w", err)
	}
	if assignment == nil {
		return nil, errors.New("assignment not found")
	}

	return assignment, nil
}

func (s *assignmentService) GetAllAssignments(ctx context.Context, page, limit int) ([]models.AssignmentWithStats, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	assignments, total, err := s.assignmentRepo.GetAll(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get all assignments: %w", err)
	}

	return assignments, total, nil
}

func (s *assignmentService) UpdateAssignment(ctx context.Context, id string, req *models.CreateAssignmentRequest) error {
	assignment, err := s.assignmentRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get assignment: %w", err)
	}
	if assignment == nil {
		return errors.New("assignment not found")
	}

	assignment.Title = req.Title
	assignment.Description = req.Description
	assignment.UpdatedAt = time.Now()

	return s.assignmentRepo.Update(ctx, &assignment.Assignment)
}

func (s *assignmentService) DeleteAssignment(ctx context.Context, id string) error {
	// Проверяем, существует ли задание
	assignment, err := s.assignmentRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get assignment: %w", err)
	}
	if assignment == nil {
		return errors.New("assignment not found")
	}

	// Проверяем, есть ли связанные работы
	if assignment.TotalWorks > 0 {
		return errors.New("cannot delete assignment with existing works")
	}

	return s.assignmentRepo.Delete(ctx, id)
}
