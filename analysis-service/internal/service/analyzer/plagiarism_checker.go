package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/models"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/service/integration"
	"github.com/rs/zerolog"
)

type PlagiarismChecker interface {
	CheckPlagiarism(ctx context.Context, workID, fileID, assignmentID, studentID string) (*models.AnalysisResult, error)
	BatchCheck(ctx context.Context, requests []models.PlagiarismCheckRequest) ([]models.AnalysisResult, error)
	GetCheckerInfo() CheckerInfo
}

type CheckerInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Algorithm   string `json:"algorithm"`
	Description string `json:"description"`
}

type plagiarismChecker struct {
	workClient     integration.WorkClient
	fileClient     integration.FileClient
	hashComparator HashComparator
	logger         zerolog.Logger
	config         PlagiarismCheckerConfig
}

type PlagiarismCheckerConfig struct {
	HashAlgorithm       string
	SimilarityThreshold int
	EnableDeepAnalysis  bool
	Timeout             time.Duration
	MaxRetries          int
}

func NewPlagiarismChecker(
	workClient integration.WorkClient,
	fileClient integration.FileClient,
	hashComparator HashComparator,
	logger zerolog.Logger,
	config PlagiarismCheckerConfig,
) PlagiarismChecker {
	return &plagiarismChecker{
		workClient:     workClient,
		fileClient:     fileClient,
		hashComparator: hashComparator,
		logger:         logger,
		config:         config,
	}
}

func (c *plagiarismChecker) CheckPlagiarism(ctx context.Context, workID, fileID, assignmentID, studentID string) (*models.AnalysisResult, error) {
	startTime := time.Now()

	c.logger.Info().
		Str("work_id", workID).
		Str("file_id", fileID).
		Str("assignment_id", assignmentID).
		Msg("Starting plagiarism check")

	currentFileHash, currentFileSize, err := c.fileClient.GetFileHash(ctx, fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current file hash: %w", err)
	}

	c.logger.Debug().
		Str("work_id", workID).
		Str("file_hash", currentFileHash).
		Int64("file_size", currentFileSize).
		Msg("Got current file hash")

	previousWorks, err := c.workClient.GetPreviousWorks(ctx, assignmentID, workID)
	if err != nil {
		return nil, fmt.Errorf("failed to get previous works: %w", err)
	}

	c.logger.Debug().
		Str("work_id", workID).
		Int("previous_works_count", len(previousWorks)).
		Msg("Got previous works")

	result := &models.AnalysisResult{
		WorkID:            workID,
		Status:            "processing",
		FileHash:          currentFileHash,
		ComparedWithCount: len(previousWorks),
		AnalyzedAt:        time.Now(),
	}

	if len(previousWorks) == 0 {
		result.Status = "completed"
		result.PlagiarismFlag = false
		result.MatchPercentage = 0
		result.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())

		c.logger.Info().
			Str("work_id", workID).
			Msg("No previous works to compare with")

		return result, nil
	}

	var similarWorks []models.SimilarWork
	var highestMatch int = 0
	var originalWorkID *string

	for _, prevWork := range previousWorks {
		prevFileHash := prevWork.FileHash
		if prevFileHash == "" {
			c.logger.Warn().
				Str("prev_work_id", prevWork.WorkID).
				Msg("Previous work missing file hash, skipping")
			continue
		}

		matchPercentage, err := c.hashComparator.CompareHashes(currentFileHash, prevFileHash)
		if err != nil {
			c.logger.Error().
				Err(err).
				Str("prev_work_id", prevWork.WorkID).
				Msg("Failed to compare hashes")
			continue
		}

		similarWork := models.SimilarWork{
			WorkID:          prevWork.WorkID,
			StudentID:       prevWork.StudentID,
			MatchPercentage: matchPercentage,
			FileHash:        prevFileHash,
			SubmittedAt:     prevWork.SubmittedAt,
		}
		similarWorks = append(similarWorks, similarWork)

		if matchPercentage > highestMatch {
			highestMatch = matchPercentage

			if matchPercentage == 100 && prevWork.StudentID != studentID {
				originalWorkID = &prevWork.WorkID
			}
		}

		c.logger.Debug().
			Str("work_id", workID).
			Str("prev_work_id", prevWork.WorkID).
			Int("match_percentage", matchPercentage).
			Msg("Compared with previous work")
	}

	plagiarismDetected := false
	if highestMatch >= c.config.SimilarityThreshold {
		if originalWorkID != nil {
			plagiarismDetected = true
		}
	}

	details := models.ReportDetails{
		ComparisonResults: make([]models.ComparisonResult, 0, len(similarWorks)),
		FileInfo: models.FileInfo{
			FileSize: currentFileSize,
		},
		AnalysisMetadata: models.AnalysisMetadata{
			AlgorithmUsed:    c.config.HashAlgorithm,
			SimilarityMethod: "hash_comparison",
			AnalysisVersion:  "1.0",
			Threshold:        c.config.SimilarityThreshold,
			StartedAt:        startTime,
			CompletedAt:      time.Now(),
		},
	}

	for _, work := range similarWorks {
		details.ComparisonResults = append(details.ComparisonResults, models.ComparisonResult{
			ComparedWorkID:  work.WorkID,
			StudentID:       work.StudentID,
			MatchPercentage: work.MatchPercentage,
			FileHash:        work.FileHash,
			ComparedAt:      time.Now().Format(time.RFC3339),
		})
	}

	detailsJSON, _ := json.Marshal(details)

	result.Status = "completed"
	result.PlagiarismFlag = plagiarismDetected
	result.OriginalWorkID = originalWorkID
	result.MatchPercentage = highestMatch
	result.SimilarWorks = similarWorks
	result.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())
	result.Details = detailsJSON

	c.logger.Info().
		Str("work_id", workID).
		Bool("plagiarism_detected", plagiarismDetected).
		Int("match_percentage", highestMatch).
		Int("compared_with", len(previousWorks)).
		Int("processing_time_ms", result.ProcessingTimeMs).
		Msg("Plagiarism check completed")

	return result, nil
}

func (c *plagiarismChecker) BatchCheck(ctx context.Context, requests []models.PlagiarismCheckRequest) ([]models.AnalysisResult, error) {
	results := make([]models.AnalysisResult, 0, len(requests))

	for _, req := range requests {
		result, err := c.CheckPlagiarism(ctx, req.WorkID, req.FileID, req.AssignmentID, req.StudentID)
		if err != nil {
			c.logger.Error().
				Err(err).
				Str("work_id", req.WorkID).
				Msg("Failed to check plagiarism in batch")

			failedResult := &models.AnalysisResult{
				WorkID:     req.WorkID,
				Status:     "failed",
				AnalyzedAt: time.Now(),
			}
			results = append(results, *failedResult)
			continue
		}

		results = append(results, *result)
	}

	return results, nil
}

func (c *plagiarismChecker) GetCheckerInfo() CheckerInfo {
	return CheckerInfo{
		Name:        "Plagiarism Checker",
		Version:     "1.0.0",
		Algorithm:   c.config.HashAlgorithm,
		Description: "Checks for plagiarism by comparing file hashes",
	}
}
