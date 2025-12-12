package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/models"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/repository"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/service/analyzer"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/service/integration"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/worker/queue"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type AnalysisService interface {
	AnalyzeWork(ctx context.Context, workID, fileID, assignmentID, studentID string) (*models.AnalysisResult, error)
	AnalyzeWorkAsync(ctx context.Context, workID, fileID, assignmentID, studentID string) (string, error)
	GetAnalysisResult(ctx context.Context, workID string) (*models.AnalysisResult, error)
	BatchAnalyze(ctx context.Context, workIDs []string) (*models.BatchAnalysisResponse, error)
	GetServiceStatus(ctx context.Context) (*models.HealthCheckResponse, error)
	RetryFailedAnalyses(ctx context.Context, limit int) (int, error)
}

type analysisService struct {
	reportRepo        repository.ReportRepository
	plagiarismRepo    repository.PlagiarismRepository
	workClient        integration.WorkClient
	fileClient        integration.FileClient
	plagiarismChecker analyzer.PlagiarismChecker
	messageHandler    queue.MessageHandler
	rabbitMQPublisher queue.RabbitMQPublisher
	logger            zerolog.Logger
	config            AnalysisConfig
	mu                sync.RWMutex
}

type AnalysisConfig struct {
	HashAlgorithm       string
	SimilarityThreshold int
	EnableDeepAnalysis  bool
	Timeout             time.Duration
	MaxRetries          int
	BatchSize           int
}

func NewAnalysisService(
	reportRepo repository.ReportRepository,
	plagiarismRepo repository.PlagiarismRepository,
	workClient integration.WorkClient,
	fileClient integration.FileClient,
	plagiarismChecker analyzer.PlagiarismChecker,
	messageHandler queue.MessageHandler,
	rabbitMQPublisher queue.RabbitMQPublisher,
	logger zerolog.Logger,
	config AnalysisConfig,
) AnalysisService {
	return &analysisService{
		reportRepo:        reportRepo,
		plagiarismRepo:    plagiarismRepo,
		workClient:        workClient,
		fileClient:        fileClient,
		plagiarismChecker: plagiarismChecker,
		messageHandler:    messageHandler,
		rabbitMQPublisher: rabbitMQPublisher,
		logger:            logger,
		config:            config,
	}
}

func (s *analysisService) AnalyzeWork(ctx context.Context, workID, fileID, assignmentID, studentID string) (*models.AnalysisResult, error) {
	startTime := time.Now()

	// Check if analysis already exists
	existingReport, err := s.reportRepo.GetByWorkID(ctx, workID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing report: %w", err)
	}

	if existingReport != nil && existingReport.Status == models.ReportStatusCompleted.String() {
		s.logger.Info().Str("work_id", workID).Msg("Analysis already completed, returning cached result")
		return s.convertReportToResult(existingReport), nil
	}

	// Create or update report
	report := &models.Report{
		ID:           uuid.New().String(),
		WorkID:       workID,
		FileID:       fileID,
		AssignmentID: assignmentID,
		StudentID:    studentID,
		Status:       models.ReportStatusProcessing.String(),
		StartedAt:    &startTime,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if existingReport != nil {
		report.ID = existingReport.ID
		report.Status = models.ReportStatusProcessing.String()
		report.StartedAt = &startTime
		report.UpdatedAt = time.Now()

		if err := s.reportRepo.UpdateStatus(ctx, report.ID, report.Status); err != nil {
			return nil, fmt.Errorf("failed to update report status: %w", err)
		}
	} else {
		if err := s.reportRepo.Create(ctx, report); err != nil {
			return nil, fmt.Errorf("failed to create report: %w", err)
		}
	}

	// Update work status in Work Service
	if err := s.workClient.UpdateWorkStatus(ctx, workID, "analyzing"); err != nil {
		s.logger.Error().Err(err).Str("work_id", workID).Msg("Failed to update work status")
	}

	// Perform plagiarism check
	result, err := s.plagiarismChecker.CheckPlagiarism(ctx, workID, fileID, assignmentID, studentID)
	if err != nil {
		// Update report with failure
		report.Status = models.ReportStatusFailed.String()
		report.UpdatedAt = time.Now()
		if updateErr := s.reportRepo.UpdateStatus(ctx, report.ID, report.Status); updateErr != nil {
			s.logger.Error().Err(updateErr).Msg("Failed to update failed report")
		}

		// Update work status
		if updateErr := s.workClient.UpdateWorkStatus(ctx, workID, "failed"); updateErr != nil {
			s.logger.Error().Err(updateErr).Msg("Failed to update work status to failed")
		}

		return nil, fmt.Errorf("plagiarism check failed: %w", err)
	}

	// Update report with results
	completedAt := time.Now()
	processingTime := int(completedAt.Sub(startTime).Milliseconds())

	report.Status = models.ReportStatusCompleted.String()
	report.PlagiarismFlag = result.PlagiarismFlag
	report.OriginalWorkID = result.OriginalWorkID
	report.MatchPercentage = result.MatchPercentage
	report.FileHash = result.FileHash
	report.ProcessingTimeMs = &processingTime
	report.ComparedFilesCount = result.ComparedWithCount
	report.CompletedAt = &completedAt
	report.UpdatedAt = completedAt

	// Save compared hashes
	if result.SimilarWorks != nil {
		comparedHashes := make([]string, 0, len(result.SimilarWorks))
		for _, work := range result.SimilarWorks {
			comparedHashes = append(comparedHashes, work.FileHash)
		}
		report.ComparedHashes = comparedHashes
	}

	// Save details
	if result.Details != nil {
		report.Details = result.Details
	}

	// Update report in database
	if err := s.reportRepo.Update(ctx, report); err != nil {
		return nil, fmt.Errorf("failed to update report with results: %w", err)
	}

	// Update work status
	workStatus := "analyzed"
	if result.PlagiarismFlag {
		workStatus = "plagiarized"
	}

	if err := s.workClient.UpdateWorkStatus(ctx, workID, workStatus); err != nil {
		s.logger.Error().Err(err).Msg("Failed to update work status")
	}

	// Publish analysis completed event
	event := models.AnalysisCompletedEvent{
		WorkID:          workID,
		ReportID:        report.ID,
		Status:          report.Status,
		PlagiarismFlag:  report.PlagiarismFlag,
		OriginalWorkID:  report.OriginalWorkID,
		MatchPercentage: report.MatchPercentage,
		ProcessingTime:  processingTime,
		CompletedAt:     completedAt,
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to marshal analysis completed event")
	} else {
		if err := s.rabbitMQPublisher.Publish(ctx, "plagiarism_exchange", "analysis.completed", eventJSON); err != nil {
			s.logger.Error().Err(err).Msg("Failed to publish analysis completed event")
		}
	}

	s.logger.Info().
		Str("work_id", workID).
		Bool("plagiarism", result.PlagiarismFlag).
		Int("match_percentage", result.MatchPercentage).
		Int("processing_time_ms", processingTime).
		Msg("Analysis completed successfully")

	return result, nil
}

func (s *analysisService) AnalyzeWorkAsync(ctx context.Context, workID, fileID, assignmentID, studentID string) (string, error) {
	// Create initial report
	reportID := uuid.New().String()
	report := &models.Report{
		ID:           reportID,
		WorkID:       workID,
		FileID:       fileID,
		AssignmentID: assignmentID,
		StudentID:    studentID,
		Status:       models.ReportStatusPending.String(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.reportRepo.Create(ctx, report); err != nil {
		return "", fmt.Errorf("failed to create report: %w", err)
	}

	// Publish async analysis request
	request := models.PlagiarismCheckRequest{
		WorkID:       workID,
		FileID:       fileID,
		AssignmentID: assignmentID,
		StudentID:    studentID,
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal analysis request: %w", err)
	}

	if err := s.rabbitMQPublisher.Publish(ctx, "plagiarism_exchange", "analysis.request", requestJSON); err != nil {
		return "", fmt.Errorf("failed to publish analysis request: %w", err)
	}

	s.logger.Info().
		Str("work_id", workID).
		Str("report_id", reportID).
		Msg("Async analysis requested")

	return reportID, nil
}

func (s *analysisService) GetAnalysisResult(ctx context.Context, workID string) (*models.AnalysisResult, error) {
	report, err := s.reportRepo.GetByWorkID(ctx, workID)
	if err != nil {
		return nil, fmt.Errorf("failed to get report: %w", err)
	}

	if report == nil {
		return nil, errors.New("analysis not found for this work")
	}

	return s.convertReportToResult(report), nil
}

func (s *analysisService) BatchAnalyze(ctx context.Context, workIDs []string) (*models.BatchAnalysisResponse, error) {
	startTime := time.Now()

	if len(workIDs) > s.config.BatchSize {
		return nil, fmt.Errorf("batch size exceeds limit of %d", s.config.BatchSize)
	}

	s.logger.Info().
		Int("work_count", len(workIDs)).
		Msg("Starting batch analysis")

	response := &models.BatchAnalysisResponse{
		Total:       len(workIDs),
		Processed:   0,
		Failed:      0,
		Results:     make([]models.PlagiarismCheckResponse, 0, len(workIDs)),
		CompletedAt: time.Now(),
	}

	// Process in batches
	batchSize := 5 // Process 5 works at a time
	for i := 0; i < len(workIDs); i += batchSize {
		end := i + batchSize
		if end > len(workIDs) {
			end = len(workIDs)
		}

		batch := workIDs[i:end]

		// Process batch concurrently
		var wg sync.WaitGroup
		results := make([]models.PlagiarismCheckResponse, len(batch))
		errors := make([]error, len(batch))

		for j, workID := range batch {
			wg.Add(1)
			go func(idx int, wID string) {
				defer wg.Done()

				// Get work info (in real implementation, you'd get fileID, assignmentID, studentID)
				// For now, use placeholder
				result, err := s.AnalyzeWork(ctx, wID, "file_"+wID, "assignment_"+wID, "student_"+wID)
				if err != nil {
					errors[idx] = err
					return
				}

				results[idx] = models.PlagiarismCheckResponse{
					ReportID:        "report_" + wID,
					WorkID:          wID,
					Status:          result.Status,
					PlagiarismFlag:  result.PlagiarismFlag,
					MatchPercentage: result.MatchPercentage,
					OriginalWorkID:  result.OriginalWorkID,
					AnalyzedAt:      result.AnalyzedAt,
				}
			}(j, workID)
		}

		wg.Wait()

		// Collect results
		for j, result := range results {
			if result.WorkID != "" {
				response.Results = append(response.Results, result)
				response.Processed++
			} else if errors[j] != nil {
				s.logger.Error().
					Err(errors[j]).
					Str("work_id", batch[j]).
					Msg("Failed to analyze work in batch")
				response.Failed++
			}
		}
	}

	response.CompletedAt = time.Now()

	s.logger.Info().
		Int("total", response.Total).
		Int("processed", response.Processed).
		Int("failed", response.Failed).
		Dur("duration", response.CompletedAt.Sub(startTime)).
		Msg("Batch analysis completed")

	return response, nil
}

func (s *analysisService) GetServiceStatus(ctx context.Context) (*models.HealthCheckResponse, error) {
	// Check database connection
	dbOK := true
	if err := s.reportRepo.Ping(ctx); err != nil {
		dbOK = false
		s.logger.Error().Err(err).Msg("Database health check failed")
	}

	// Check external services (simplified)
	workServiceOK := true
	fileServiceOK := true

	// Get statistics
	stats, err := s.reportRepo.GetStats(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get service statistics")
	}

	response := &models.HealthCheckResponse{
		Status:        "healthy",
		Database:      dbOK,
		RabbitMQ:      true, // Would check RabbitMQ connection
		WorkService:   workServiceOK,
		FileService:   fileServiceOK,
		ActiveWorkers: 0,     // Would get from worker pool
		QueueLength:   0,     // Would get from queue
		Uptime:        "24h", // Would calculate actual uptime
		Timestamp:     time.Now(),
	}

	if !dbOK || !workServiceOK || !fileServiceOK {
		response.Status = "degraded"
	}

	return response, nil
}

func (s *analysisService) RetryFailedAnalyses(ctx context.Context, limit int) (int, error) {
	// Get failed reports
	failedReports, err := s.reportRepo.GetReportsByStatus(ctx, models.ReportStatusFailed.String(), limit)
	if err != nil {
		return 0, fmt.Errorf("failed to get failed reports: %w", err)
	}

	retryCount := 0
	for _, report := range failedReports {
		s.logger.Info().
			Str("work_id", report.WorkID).
			Str("report_id", report.ID).
			Msg("Retrying failed analysis")

		// Retry analysis
		_, err := s.AnalyzeWork(ctx, report.WorkID, report.FileID, report.AssignmentID, report.StudentID)
		if err != nil {
			s.logger.Error().
				Err(err).
				Str("work_id", report.WorkID).
				Msg("Failed to retry analysis")
			continue
		}

		retryCount++
	}

	s.logger.Info().
		Int("total_failed", len(failedReports)).
		Int("retried", retryCount).
		Msg("Failed analyses retry completed")

	return retryCount, nil
}

func (s *analysisService) convertReportToResult(report *models.Report) *models.AnalysisResult {
	result := &models.AnalysisResult{
		WorkID:            report.WorkID,
		Status:            report.Status,
		PlagiarismFlag:    report.PlagiarismFlag,
		OriginalWorkID:    report.OriginalWorkID,
		MatchPercentage:   report.MatchPercentage,
		FileHash:          report.FileHash,
		ComparedWithCount: report.ComparedFilesCount,
		AnalyzedAt:        report.UpdatedAt,
	}

	if report.ProcessingTimeMs != nil {
		result.ProcessingTimeMs = *report.ProcessingTimeMs
	}

	// Parse details if available
	if report.Details != nil && len(report.Details) > 0 {
		var details models.ReportDetails
		if err := json.Unmarshal(report.Details, &details); err == nil {
			// Convert comparison results to similar works
			for _, compResult := range details.ComparisonResults {
				similarWork := models.SimilarWork{
					WorkID:          compResult.ComparedWorkID,
					StudentID:       compResult.StudentID,
					MatchPercentage: compResult.MatchPercentage,
					FileHash:        compResult.FileHash,
				}
				result.SimilarWorks = append(result.SimilarWorks, similarWork)
			}
			result.Details = report.Details
		}
	}

	return result
}
