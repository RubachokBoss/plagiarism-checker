package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/RubachokBoss/plagiarism-checker/work-service/internal/service/integration"

	"github.com/RubachokBoss/plagiarism-checker/work-service/internal/models"
	"github.com/RubachokBoss/plagiarism-checker/work-service/internal/repository"
	"github.com/rs/zerolog"
)

type ReportService interface {
	GetWorkReport(ctx context.Context, workID string) (*models.ReportResponse, error)
}

type reportService struct {
	workRepo       repository.WorkRepository
	studentRepo    repository.StudentRepository
	assignmentRepo repository.AssignmentRepository
	analysisClient integration.AnalysisClient
	logger         zerolog.Logger
}

func NewReportService(
	workRepo repository.WorkRepository,
	studentRepo repository.StudentRepository,
	assignmentRepo repository.AssignmentRepository,
	analysisClient integration.AnalysisClient,
	logger zerolog.Logger,
) ReportService {
	return &reportService{
		workRepo:       workRepo,
		studentRepo:    studentRepo,
		assignmentRepo: assignmentRepo,
		analysisClient: analysisClient,
		logger:         logger,
	}
}

func (s *reportService) GetWorkReport(ctx context.Context, workID string) (*models.ReportResponse, error) {
	work, err := s.workRepo.GetByID(ctx, workID)
	if err != nil {
		return nil, fmt.Errorf("failed to get work: %w", err)
	}
	if work == nil {
		return nil, errors.New("work not found")
	}

	analysisReport, err := s.analysisClient.GetReport(ctx, workID)
	if err != nil {
		s.logger.Error().Err(err).Str("work_id", workID).Msg("Failed to get analysis report")
		if analysisReport == nil {
			return &models.ReportResponse{
				WorkID:       work.ID,
				StudentID:    work.StudentID,
				AssignmentID: work.AssignmentID,
				Status:       work.Status,
				CreatedAt:    work.CreatedAt,
			}, nil
		}
	}

	report := &models.ReportResponse{
		WorkID:          work.ID,
		StudentID:       work.StudentID,
		AssignmentID:    work.AssignmentID,
		Status:          work.Status,
		PlagiarismFlag:  analysisReport.PlagiarismFlag,
		OriginalWorkID:  analysisReport.OriginalWorkID,
		MatchPercentage: analysisReport.MatchPercentage,
		AnalyzedAt:      analysisReport.AnalyzedAt,
		CreatedAt:       work.CreatedAt,
	}

	return report, nil
}
