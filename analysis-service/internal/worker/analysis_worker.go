package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/models"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/repository"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/service"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/worker/queue"
	"github.com/google/uuid"
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

	if err := w.workerPool.Start(ctx); err != nil {
		return fmt.Errorf("failed to start worker pool: %w", err)
	}

	msgs, err := w.queueConsumer.Consume(ctx)
	if err != nil {
		return fmt.Errorf("failed to start consuming messages: %w", err)
	}

	go w.processMessages(ctx, msgs)

	w.logger.Info().Msg("Analysis worker started successfully")
	return nil
}

func (w *analysisWorker) Stop() error {
	w.logger.Info().Msg("Stopping analysis worker...")

	if err := w.workerPool.Stop(); err != nil {
		w.logger.Error().Err(err).Msg("Failed to stop worker pool")
	}

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

			w.workerPool.Submit(func() {
				if err := w.processMessage(ctx, msg); err != nil {
					w.logger.Error().Err(err).Msg("Failed to process message")

					w.statsMutex.Lock()
					w.stats.FailedJobs++
					w.statsMutex.Unlock()

					if isPermanentError(err) {
						if ackErr := msg.Ack(false); ackErr != nil {
							w.logger.Error().Err(ackErr).Msg("Failed to ack message")
						}
						return
					}

					if nackErr := msg.Nack(false, true); nackErr != nil {
						w.logger.Error().Err(nackErr).Msg("Failed to nack message")
					}
				} else {
					if err := msg.Ack(false); err != nil {
						w.logger.Error().Err(err).Msg("Failed to ack message")
					}

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
	var event models.WorkCreatedEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		return permanent(fmt.Errorf("failed to unmarshal event: %w", err))
	}

	if strings.TrimSpace(event.WorkID) == "" {
		return permanent(errors.New("empty work_id"))
	}
	if strings.TrimSpace(event.FileID) == "" {
		return permanent(errors.New("empty file_id"))
	}

	w.logger.Info().
		Str("work_id", event.WorkID).
		Str("file_id", event.FileID).
		Str("assignment_id", event.AssignmentID).
		Msg("Processing work analysis")

	return w.ProcessWork(ctx, event.WorkID, event.FileID, event.AssignmentID, event.StudentID)
}

func (w *analysisWorker) ProcessWork(ctx context.Context, workID, fileID, assignmentID, studentID string) error {
	startTime := time.Now()

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

	report := &models.Report{
		ID:           uuid.New().String(),
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

	result, err := w.analysisService.AnalyzeWork(ctx, workID, fileID, assignmentID, studentID)
	if err != nil {
		report.Status = models.ReportStatusFailed.String()
		report.UpdatedAt = time.Now()
		if updateErr := w.reportRepo.Update(ctx, report); updateErr != nil {
			w.logger.Error().Err(updateErr).Msg("Failed to update failed report")
		}

		return fmt.Errorf("failed to analyze work: %w", err)
	}

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

	if result.Details != nil {
		report.Details = result.Details
	}

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

	queueLength, err := w.queueConsumer.GetQueueLength()
	if err != nil {
		w.logger.Error().Err(err).Msg("Failed to get queue length")
	} else {
		w.stats.QueueLength = queueLength
	}

	w.stats.ActiveWorkers = w.workerPool.GetActiveWorkers()

	return w.stats
}

type permanentError struct {
	err error
}

func (e permanentError) Error() string { return e.err.Error() }
func (e permanentError) Unwrap() error { return e.err }

func permanent(err error) error {
	return permanentError{err: err}
}

func isPermanentError(err error) bool {
	var p permanentError
	return errors.As(err, &p)
}
