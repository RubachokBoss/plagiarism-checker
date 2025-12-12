package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/models"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/repository"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/service"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/worker/queue"
	"github.com/rs/zerolog"
)

type AnalysisWorker interface {
	Start(ctx context.Context) error
	Stop() error
	ProcessWork(ctx context.Context, workID, fileID, assignmentID, studentID string) error
	GetStats() WorkerStats
}

type WorkerStats struct {
	ActiveWorkers  int `json:"active_workers"`
	ProcessedToday int `json:"processed_today"`
	TotalProcessed int `json:"total_processed"`
	FailedJobs     int `json:"failed_jobs"`
	QueueLength    int `json:"queue_length"`
}

type analysisWorker struct {
	workerPool      *WorkerPool
	queueConsumer   queue.RabbitMQConsumer
	reportRepo      repository.ReportRepository
	analysisService service.AnalysisService
	logger          zerolog.Logger
	stats           WorkerStats
	statsMutex      sync.RWMutex
	startTime       time.Time
}

func NewAnalysisWorker(
	workerPool *WorkerPool,
	queueConsumer queue.RabbitMQConsumer,
	reportRepo repository.ReportRepository,
	analysisService service.AnalysisService,
	logger zerolog.Logger,
) AnalysisWorker {
	return &analysisWorker{
		workerPool:      workerPool,
		queueConsumer:   queueConsumer,
		reportRepo:      reportRepo,
		analysisService: analysisService,
		logger:          logger,
		stats:           WorkerStats{},
		startTime:       time.Now(),
	}
}

func (w *analysisWorker) Start(ctx context.Context) error {
	w.logger.Info().Msg("Starting analysis worker...")

	// Start worker pool
	if err := w.workerPool.Start(ctx); err != nil {
		return fmt.Errorf("failed to start worker pool: %w", err)
	}

	// Start consuming messages
	msgs, err := w.queueConsumer.Consume(ctx)
	if err != nil {
		return fmt.Errorf("failed to start consuming messages: %w", err)
	}

	// Start message processing loop
	go w.processMessages(ctx, msgs)

	w.logger.Info().Msg("Analysis worker started successfully")
	return nil
}

func (w *analysisWorker) Stop() error {
	w.logger.Info().Msg("Stopping analysis worker...")

	// Stop worker pool
	if err := w.workerPool.Stop(); err != nil {
		w.logger.Error().Err(err).Msg("Failed to stop worker pool")
	}

	// Close queue consumer
	if err := w.queueConsumer.Close(); err != nil {
		w.logger.Error().Err(err).Msg("Failed to close queue consumer")
	}

	w.logger.Info().
		Int("total_processed", w.stats.TotalProcessed).
		Int("failed_jobs", w.stats.FailedJobs).
		Dur("uptime", time.Since(w.startTime)).
		Msg("Analysis worker stopped")

	return nil
}

func (w *analysisWorker) processMessages(ctx context.Context, msgs <-chan queue.RabbitMQMessage) {
	for {
		select {
		case <-ctx.Done():
			w.logger.Info().Msg("Stopping message processing")
			return
		case msg, ok := <-msgs:
			if !ok {
				w.logger.Warn().Msg("Message channel closed")
				return
			}

			// Process message in worker pool
			w.workerPool.Submit(func() {
				if err := w.processMessage(ctx, msg); err != nil {
					w.logger.Error().Err(err).Msg("Failed to process message")

					// Update stats
					w.statsMutex.Lock()
					w.stats.FailedJobs++
					w.statsMutex.Unlock()

					// Nack the message (requeue)
					if err := msg.Nack(false, true); err != nil {
						w.logger.Error().Err(err).Msg("Failed to nack message")
					}
				} else {
					// Ack the message
					if err := msg.Ack(false); err != nil {
						w.logger.Error().Err(err).Msg("Failed to ack message")
					}

					// Update stats
					w.statsMutex.Lock()
					w.stats.TotalProcessed++
					if time.Since(msg.Timestamp).Hours() < 24 {
						w.stats.ProcessedToday++
					}
					w.statsMutex.Unlock()
				}
			})
		}
	}
}

func (w *analysisWorker) processMessage(ctx context.Context, msg queue.RabbitMQMessage) error {
	// Parse work created event
	var event models.WorkCreatedEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	w.logger.Info().
		Str("work_id", event.WorkID).
		Str("file_id", event.FileID).
		Str("assignment_id", event.AssignmentID).
		Msg("Processing work analysis")

	// Process the work
	return w.ProcessWork(ctx, event.WorkID, event.FileID, event.AssignmentID, event.StudentID)
}

func (w *analysisWorker) ProcessWork(ctx context.Context, workID, fileID, assignmentID, studentID string) error {
	startTime := time.Now()

	// Check if report already exists
	exists, err := w.reportRepo.Exists(ctx, workID)
	if err != nil {
		return fmt.Errorf("failed to check if report exists: %w", err)
	}

	if exists {
		w.logger.Warn().
			Str("work_id", workID).
			Msg("Report already exists, skipping")
		return nil
	}

	// Create initial report
	report := &models.Report{
		ID:           fmt.Sprintf("report_%s", workID),
		WorkID:       workID,
		FileID:       fileID,
		AssignmentID: assignmentID,
		StudentID:    studentID,
		Status:       models.ReportStatusProcessing.String(),
		CreatedAt:    time.Now(),
		StartedAt:    &startTime,
		UpdatedAt:    time.Now(),
	}

	if err := w.reportRepo.Create(ctx, report); err != nil {
		return fmt.Errorf("failed to create report: %w", err)
	}

	// Perform plagiarism check
	result, err := w.analysisService.AnalyzeWork(ctx, workID, fileID, assignmentID, studentID)
	if err != nil {
		// Update report with failure
		report.Status = models.ReportStatusFailed.String()
		report.UpdatedAt = time.Now()
		if updateErr := w.reportRepo.Update(ctx, report); updateErr != nil {
			w.logger.Error().Err(updateErr).Msg("Failed to update failed report")
		}

		return fmt.Errorf("failed to analyze work: %w", err)
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

	// Convert details to JSON
	if result.Details != nil {
		report.Details = result.Details
	}

	// Save updated report
	if err := w.reportRepo.Update(ctx, report); err != nil {
		return fmt.Errorf("failed to update report with results: %w", err)
	}

	w.logger.Info().
		Str("work_id", workID).
		Bool("plagiarism_detected", result.PlagiarismFlag).
		Int("match_percentage", result.MatchPercentage).
		Int("processing_time_ms", processingTime).
		Int("compared_files", result.ComparedWithCount).
		Msg("Work analysis completed")

	return nil
}

func (w *analysisWorker) GetStats() WorkerStats {
	w.statsMutex.RLock()
	defer w.statsMutex.RUnlock()

	// Get current queue length
	queueLength, err := w.queueConsumer.GetQueueLength()
	if err != nil {
		w.logger.Error().Err(err).Msg("Failed to get queue length")
	} else {
		w.stats.QueueLength = queueLength
	}

	// Get active workers
	w.stats.ActiveWorkers = w.workerPool.GetActiveWorkers()

	return w.stats
}
