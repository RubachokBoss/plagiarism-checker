package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/RubachokBoss/plagiarism-checker/work-service/internal/service/integration"
	"time"

	"github.com/RubachokBoss/plagiarism-checker/work-service/internal/models"
	"github.com/RubachokBoss/plagiarism-checker/work-service/internal/repository"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type WorkService interface {
	CreateWork(ctx context.Context, req *models.CreateWorkRequest) (*models.CreateWorkResponse, error)
	UploadWork(ctx context.Context, req *models.UploadWorkRequest) (*models.CreateWorkResponse, error)
	GetWorkByID(ctx context.Context, id string) (*models.WorkWithDetails, error)
	GetWorksByAssignment(ctx context.Context, assignmentID string, page, limit int) (*models.WorksResponse, error)
	GetWorksByStudent(ctx context.Context, studentID string, page, limit int) (*models.WorksResponse, error)
	GetAllWorks(ctx context.Context, page, limit int) (*models.WorksResponse, error)
	UpdateWorkStatus(ctx context.Context, id, status string) error
	DeleteWork(ctx context.Context, id string) error
	GetPreviousWorks(ctx context.Context, assignmentID, excludeWorkID string) ([]models.Work, error)
}

type workService struct {
	workRepo       repository.WorkRepository
	studentRepo    repository.StudentRepository
	assignmentRepo repository.AssignmentRepository
	fileClient     integration.FileClient
	rabbitmqClient integration.RabbitMQClient
	logger         zerolog.Logger
}

func NewWorkService(
	workRepo repository.WorkRepository,
	studentRepo repository.StudentRepository,
	assignmentRepo repository.AssignmentRepository,
	fileClient integration.FileClient,
	rabbitmqClient integration.RabbitMQClient,
	logger zerolog.Logger,
) WorkService {
	return &workService{
		workRepo:       workRepo,
		studentRepo:    studentRepo,
		assignmentRepo: assignmentRepo,
		fileClient:     fileClient,
		rabbitmqClient: rabbitmqClient,
		logger:         logger,
	}
}

func (s *workService) CreateWork(ctx context.Context, req *models.CreateWorkRequest) (*models.CreateWorkResponse, error) {
	// Проверяем существование студента
	studentExists, err := s.studentRepo.Exists(ctx, req.StudentID)
	if err != nil {
		return nil, fmt.Errorf("failed to check student existence: %w", err)
	}
	if !studentExists {
		return nil, errors.New("student not found")
	}

	// Проверяем существование задания
	assignmentExists, err := s.assignmentRepo.Exists(ctx, req.AssignmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to check assignment existence: %w", err)
	}
	if !assignmentExists {
		return nil, errors.New("assignment not found")
	}

	// Проверяем, не сдавал ли уже студент работу по этому заданию
	existingWork, err := s.workRepo.GetByStudentAndAssignment(ctx, req.StudentID, req.AssignmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing work: %w", err)
	}
	if existingWork != nil {
		return nil, errors.New("work already submitted for this assignment")
	}

	// Создаем работу с временным file_id (будет обновлен позже)
	workID := uuid.New().String()
	work := &models.Work{
		ID:           workID,
		StudentID:    req.StudentID,
		AssignmentID: req.AssignmentID,
		FileID:       "pending", // Временное значение
		Status:       models.WorkStatusUploaded.String(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Сохраняем работу в БД
	if err := s.workRepo.Create(ctx, work); err != nil {
		return nil, fmt.Errorf("failed to create work: %w", err)
	}

	s.logger.Info().
		Str("work_id", workID).
		Str("student_id", req.StudentID).
		Str("assignment_id", req.AssignmentID).
		Msg("Work created")

	return &models.CreateWorkResponse{
		ID:        workID,
		Status:    work.Status,
		CreatedAt: work.CreatedAt,
	}, nil
}

func (s *workService) UploadWork(ctx context.Context, req *models.UploadWorkRequest) (*models.CreateWorkResponse, error) {
	// Сначала создаем запись работы
	createReq := &models.CreateWorkRequest{
		StudentID:    req.StudentID,
		AssignmentID: req.AssignmentID,
	}

	workResponse, err := s.CreateWork(ctx, createReq)
	if err != nil {
		return nil, err
	}

	// Загружаем файл в File Service
	uploadResp, err := s.fileClient.UploadFile(ctx, req.FileContent, req.FileName)
	if err != nil {
		// Если загрузка файла не удалась, нужно удалить созданную запись работы
		s.workRepo.Delete(ctx, workResponse.ID)
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	// Обновляем работу с полученным file_id
	if err := s.workRepo.UpdateFileID(ctx, workResponse.ID, uploadResp.FileID); err != nil {
		// Если не удалось обновить, удаляем запись и файл
		s.workRepo.Delete(ctx, workResponse.ID)
		s.fileClient.DeleteFile(ctx, uploadResp.FileID)
		return nil, fmt.Errorf("failed to update work with file id: %w", err)
	}

	// Отправляем событие в RabbitMQ для запуска анализа
	event := &models.WorkCreatedEvent{
		WorkID:       workResponse.ID,
		FileID:       uploadResp.FileID,
		StudentID:    req.StudentID,
		AssignmentID: req.AssignmentID,
		Timestamp:    time.Now().Unix(),
	}

	if err := s.rabbitmqClient.PublishWorkCreated(ctx, event); err != nil {
		s.logger.Error().Err(err).Msg("Failed to publish work created event")
		// Не прерываем выполнение, только логируем ошибку
	}

	// Обновляем статус работы на "analyzing"
	if err := s.workRepo.UpdateStatus(ctx, workResponse.ID, models.WorkStatusAnalyzing.String()); err != nil {
		s.logger.Error().Err(err).Msg("Failed to update work status to analyzing")
	}

	s.logger.Info().
		Str("work_id", workResponse.ID).
		Str("file_id", uploadResp.FileID).
		Msg("Work uploaded and analysis started")

	workResponse.FileID = uploadResp.FileID
	return workResponse, nil
}

func (s *workService) GetWorkByID(ctx context.Context, id string) (*models.WorkWithDetails, error) {
	work, err := s.workRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get work: %w", err)
	}
	if work == nil {
		return nil, errors.New("work not found")
	}

	// Получаем детальную информацию
	works, _, err := s.workRepo.GetAll(ctx, 1, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get work details: %w", err)
	}

	for _, w := range works {
		if w.ID == id {
			return &w, nil
		}
	}

	return nil, errors.New("work details not found")
}

func (s *workService) GetWorksByAssignment(ctx context.Context, assignmentID string, page, limit int) (*models.WorksResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	works, total, err := s.workRepo.GetByAssignmentID(ctx, assignmentID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get works by assignment: %w", err)
	}

	return &models.WorksResponse{
		Works: works,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (s *workService) GetWorksByStudent(ctx context.Context, studentID string, page, limit int) (*models.WorksResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	works, total, err := s.workRepo.GetByStudentID(ctx, studentID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get works by student: %w", err)
	}

	return &models.WorksResponse{
		Works: works,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (s *workService) GetAllWorks(ctx context.Context, page, limit int) (*models.WorksResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	works, total, err := s.workRepo.GetAll(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get all works: %w", err)
	}

	return &models.WorksResponse{
		Works: works,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (s *workService) UpdateWorkStatus(ctx context.Context, id, status string) error {
	if !models.IsValidWorkStatus(status) {
		return errors.New("invalid work status")
	}

	return s.workRepo.UpdateStatus(ctx, id, status)
}

func (s *workService) DeleteWork(ctx context.Context, id string) error {
	work, err := s.workRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get work: %w", err)
	}
	if work == nil {
		return errors.New("work not found")
	}

	// Удаляем файл из File Service
	if work.FileID != "" && work.FileID != "pending" {
		if err := s.fileClient.DeleteFile(ctx, work.FileID); err != nil {
			s.logger.Error().Err(err).Str("file_id", work.FileID).Msg("Failed to delete file")
		}
	}

	// Удаляем запись из БД
	return s.workRepo.Delete(ctx, id)
}

func (s *workService) GetPreviousWorks(ctx context.Context, assignmentID, excludeWorkID string) ([]models.Work, error) {
	return s.workRepo.GetPreviousWorks(ctx, assignmentID, excludeWorkID)
}
