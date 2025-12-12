package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/models"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/repository"
	"github.com/rs/zerolog"
)

type ReportService interface {
	GetReport(ctx context.Context, reportID string) (*models.GetReportResponse, error)
	GetReportByWorkID(ctx context.Context, workID string) (*models.GetReportResponse, error)
	SearchReports(ctx context.Context, filters models.SearchReportsRequest) (*models.SearchReportsResponse, error)
	GetAssignmentStats(ctx context.Context, assignmentID string) (*models.GetAssignmentStatsResponse, error)
	GetStudentStats(ctx context.Context, studentID string) (*models.GetStudentStatsResponse, error)
	GetAllStats(ctx context.Context) (*models.AnalysisStats, error)
	ExportReports(ctx context.Context, filters map[string]interface{}, format string) ([]byte, error)
}

type reportService struct {
	reportRepo     repository.ReportRepository
	plagiarismRepo repository.PlagiarismRepository
	logger         zerolog.Logger
}

func NewReportService(
	reportRepo repository.ReportRepository,
	plagiarismRepo repository.PlagiarismRepository,
	logger zerolog.Logger,
) ReportService {
	return &reportService{
		reportRepo:     reportRepo,
		plagiarismRepo: plagiarismRepo,
		logger:         logger,
	}
}

func (s *reportService) GetReport(ctx context.Context, reportID string) (*models.GetReportResponse, error) {
	report, err := s.reportRepo.GetByID(ctx, reportID)
	if err != nil {
		return nil, fmt.Errorf("failed to get report: %w", err)
	}

	if report == nil {
		return nil, errors.New("report not found")
	}

	return s.convertToResponse(report), nil
}

func (s *reportService) GetReportByWorkID(ctx context.Context, workID string) (*models.GetReportResponse, error) {
	report, err := s.reportRepo.GetByWorkID(ctx, workID)
	if err != nil {
		return nil, fmt.Errorf("failed to get report by work ID: %w", err)
	}

	if report == nil {
		return nil, errors.New("report not found for this work")
	}

	return s.convertToResponse(report), nil
}

func (s *reportService) SearchReports(ctx context.Context, filters models.SearchReportsRequest) (*models.SearchReportsResponse, error) {
	// Convert request filters to repository filters
	repoFilters := make(map[string]interface{})

	if filters.WorkID != nil && *filters.WorkID != "" {
		repoFilters["work_id"] = *filters.WorkID
	}

	if filters.AssignmentID != nil && *filters.AssignmentID != "" {
		repoFilters["assignment_id"] = *filters.AssignmentID
	}

	if filters.StudentID != nil && *filters.StudentID != "" {
		repoFilters["student_id"] = *filters.StudentID
	}

	if filters.Status != nil && *filters.Status != "" {
		repoFilters["status"] = *filters.Status
	}

	if filters.PlagiarismFlag != nil {
		repoFilters["plagiarism_flag"] = *filters.PlagiarismFlag
	}

	if filters.DateFrom != nil && *filters.DateFrom != "" {
		if date, err := time.Parse(time.RFC3339, *filters.DateFrom); err == nil {
			repoFilters["date_from"] = date
		}
	}

	if filters.DateTo != nil && *filters.DateTo != "" {
		if date, err := time.Parse(time.RFC3339, *filters.DateTo); err == nil {
			repoFilters["date_to"] = date
		}
	}

	// Calculate offset
	offset := (filters.Page - 1) * filters.Limit

	// Search reports
	reports, total, err := s.reportRepo.Search(ctx, repoFilters, filters.Limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to search reports: %w", err)
	}

	// Convert to response
	responseReports := make([]models.GetReportResponse, 0, len(reports))
	for _, report := range reports {
		responseReports = append(responseReports, *s.convertToResponse(&report))
	}

	// Calculate total pages
	totalPages := total / filters.Limit
	if total%filters.Limit > 0 {
		totalPages++
	}

	return &models.SearchReportsResponse{
		Reports:    responseReports,
		Total:      total,
		Page:       filters.Page,
		Limit:      filters.Limit,
		TotalPages: totalPages,
	}, nil
}

func (s *reportService) GetAssignmentStats(ctx context.Context, assignmentID string) (*models.GetAssignmentStatsResponse, error) {
	// Get assignment statistics
	stats, err := s.reportRepo.GetAssignmentStats(ctx, assignmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignment stats: %w", err)
	}

	if stats == nil {
		return nil, errors.New("assignment not found or no reports available")
	}

	// Get recent reports for this assignment
	reports, _, err := s.reportRepo.GetByAssignmentID(ctx, assignmentID, 10, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignment reports: %w", err)
	}

	// Convert reports to response
	responseReports := make([]models.GetReportResponse, 0, len(reports))
	for _, report := range reports {
		responseReports = append(responseReports, *s.convertToResponse(&report))
	}

	// Get plagiarism patterns
	patterns, err := s.plagiarismRepo.GetPlagiarismPatterns(ctx, assignmentID)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get plagiarism patterns")
	}

	// Prepare statistics
	statistics := map[string]interface{}{
		"total_reports":         stats.TotalWorks,
		"analyzed_percentage":   float64(stats.AnalyzedWorks) / float64(stats.TotalWorks) * 100,
		"plagiarism_percentage": float64(stats.PlagiarizedWorks) / float64(stats.AnalyzedWorks) * 100,
		"avg_match_percentage":  stats.AvgMatchPercentage,
		"plagiarism_patterns":   patterns,
	}

	return &models.GetAssignmentStatsResponse{
		AssignmentID:       stats.AssignmentID,
		TotalWorks:         stats.TotalWorks,
		AnalyzedWorks:      stats.AnalyzedWorks,
		PlagiarizedWorks:   stats.PlagiarizedWorks,
		AvgMatchPercentage: stats.AvgMatchPercentage,
		Reports:            responseReports,
		Statistics:         statistics,
		LastAnalyzedAt:     stats.LastAnalyzedAt,
	}, nil
}

func (s *reportService) GetStudentStats(ctx context.Context, studentID string) (*models.GetStudentStatsResponse, error) {
	// Get student statistics
	stats, err := s.reportRepo.GetStudentStats(ctx, studentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get student stats: %w", err)
	}

	if stats == nil {
		return nil, errors.New("student not found or no reports available")
	}

	// Get recent reports for this student
	reports, _, err := s.reportRepo.GetByStudentID(ctx, studentID, 10, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get student reports: %w", err)
	}

	// Convert reports to response
	responseReports := make([]models.GetReportResponse, 0, len(reports))
	for _, report := range reports {
		responseReports = append(responseReports, *s.convertToResponse(&report))
	}

	// Get comparison history for the student's works
	var comparisonHistory []models.ComparisonResult
	for _, report := range reports {
		history, err := s.plagiarismRepo.GetComparisonHistory(ctx, report.WorkID)
		if err != nil {
			s.logger.Error().Err(err).Str("work_id", report.WorkID).Msg("Failed to get comparison history")
			continue
		}
		comparisonHistory = append(comparisonHistory, history...)
	}

	// Prepare statistics
	statistics := map[string]interface{}{
		"total_reports":         stats.TotalWorks,
		"analyzed_percentage":   float64(stats.AnalyzedWorks) / float64(stats.TotalWorks) * 100,
		"plagiarism_percentage": float64(stats.PlagiarizedWorks) / float64(stats.AnalyzedWorks) * 100,
		"avg_match_percentage":  stats.AvgMatchPercentage,
		"comparison_history":    comparisonHistory,
	}

	return &models.GetStudentStatsResponse{
		StudentID:          stats.StudentID,
		TotalWorks:         stats.TotalWorks,
		AnalyzedWorks:      stats.AnalyzedWorks,
		PlagiarizedWorks:   stats.PlagiarizedWorks,
		AvgMatchPercentage: stats.AvgMatchPercentage,
		Reports:            responseReports,
		Statistics:         statistics,
		LastAnalyzedAt:     stats.LastAnalyzedAt,
	}, nil
}

func (s *reportService) GetAllStats(ctx context.Context) (*models.AnalysisStats, error) {
	return s.reportRepo.GetStats(ctx)
}

func (s *reportService) ExportReports(ctx context.Context, filters map[string]interface{}, format string) ([]byte, error) {
	// Get reports based on filters
	reports, _, err := s.reportRepo.Search(ctx, filters, 1000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get reports for export: %w", err)
	}

	// Export based on format
	switch format {
	case "json":
		return s.exportJSON(reports)
	case "csv":
		return s.exportCSV(reports)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

func (s *reportService) exportJSON(reports []models.Report) ([]byte, error) {
	// Convert to response format first
	responseReports := make([]models.GetReportResponse, 0, len(reports))
	for _, report := range reports {
		responseReports = append(responseReports, *s.convertToResponse(&report))
	}

	return json.MarshalIndent(responseReports, "", "  ")
}

func (s *reportService) exportCSV(reports []models.Report) ([]byte, error) {
	// CSV header
	csvData := "Report ID,Work ID,Assignment ID,Student ID,Status,Plagiarism,Match %,Processing Time (ms),Compared Files,Created At,Completed At\n"

	// CSV rows
	for _, report := range reports {
		completedAt := ""
		if report.CompletedAt != nil {
			completedAt = report.CompletedAt.Format(time.RFC3339)
		}

		csvData += fmt.Sprintf("%s,%s,%s,%s,%s,%v,%d,%d,%d,%s,%s\n",
			report.ID,
			report.WorkID,
			report.AssignmentID,
			report.StudentID,
			report.Status,
			report.PlagiarismFlag,
			report.MatchPercentage,
			func() int {
				if report.ProcessingTimeMs != nil {
					return *report.ProcessingTimeMs
				}
				return 0
			}(),
			report.ComparedFilesCount,
			report.CreatedAt.Format(time.RFC3339),
			completedAt,
		)
	}

	return []byte(csvData), nil
}

func (s *reportService) convertToResponse(report *models.Report) *models.GetReportResponse {
	response := &models.GetReportResponse{
		ReportID:           report.ID,
		WorkID:             report.WorkID,
		FileID:             report.FileID,
		AssignmentID:       report.AssignmentID,
		StudentID:          report.StudentID,
		Status:             report.Status,
		PlagiarismFlag:     report.PlagiarismFlag,
		OriginalWorkID:     report.OriginalWorkID,
		MatchPercentage:    report.MatchPercentage,
		FileHash:           report.FileHash,
		ComparedFilesCount: report.ComparedFilesCount,
		CreatedAt:          report.CreatedAt,
		StartedAt:          report.StartedAt,
		CompletedAt:        report.CompletedAt,
	}

	// Parse details if available
	if report.Details != nil && len(report.Details) > 0 {
		var details map[string]interface{}
		if err := json.Unmarshal(report.Details, &details); err == nil {
			response.Details = details
		}
	}

	// Convert processing time
	if report.ProcessingTimeMs != nil {
		processingTime := *report.ProcessingTimeMs
		response.ProcessingTimeMs = &processingTime
	}

	return response
}
