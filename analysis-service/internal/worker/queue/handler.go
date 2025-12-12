package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/models"
	"github.com/rs/zerolog"
)

type MessageHandler interface {
	HandleWorkCreated(ctx context.Context, event models.WorkCreatedEvent) error
	HandleAnalysisRequest(ctx context.Context, request models.PlagiarismCheckRequest) error
	HandleBatchRequest(ctx context.Context, request models.BatchAnalysisRequest) error
}

type messageHandler struct {
	logger zerolog.Logger
	// Add dependencies here (e.g., services)
}

func NewMessageHandler(logger zerolog.Logger) MessageHandler {
	return &messageHandler{
		logger: logger,
	}
}

func (h *messageHandler) HandleWorkCreated(ctx context.Context, event models.WorkCreatedEvent) error {
	h.logger.Info().
		Str("work_id", event.WorkID).
		Str("file_id", event.FileID).
		Str("assignment_id", event.AssignmentID).
		Msg("Handling work created event")

	// In real implementation, this would trigger analysis
	// For now, just log the event
	return nil
}

func (h *messageHandler) HandleAnalysisRequest(ctx context.Context, request models.PlagiarismCheckRequest) error {
	h.logger.Info().
		Str("work_id", request.WorkID).
		Str("file_id", request.FileID).
		Str("assignment_id", request.AssignmentID).
		Msg("Handling analysis request")

	// In real implementation, this would start analysis
	// For now, just log the request
	return nil
}

func (h *messageHandler) HandleBatchRequest(ctx context.Context, request models.BatchAnalysisRequest) error {
	h.logger.Info().
		Int("work_count", len(request.WorkIDs)).
		Msg("Handling batch analysis request")

	// Process each work in batch
	for _, workID := range request.WorkIDs {
		h.logger.Debug().Str("work_id", workID).Msg("Processing work in batch")
		// In real implementation, process each work
	}

	return nil
}

// ProcessMessage processes incoming RabbitMQ messages
func (h *messageHandler) ProcessMessage(ctx context.Context, msg RabbitMQMessage) error {
	// Parse message based on routing key or message type
	// This is a simplified implementation
	var messageData map[string]interface{}
	if err := json.Unmarshal(msg.Body, &messageData); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// Determine message type
	msgType, ok := messageData["type"].(string)
	if !ok {
		return fmt.Errorf("message type not specified")
	}

	switch msgType {
	case "work.created":
		var event models.WorkCreatedEvent
		if err := json.Unmarshal(msg.Body, &event); err != nil {
			return fmt.Errorf("failed to unmarshal work created event: %w", err)
		}
		return h.HandleWorkCreated(ctx, event)

	case "analysis.request":
		var request models.PlagiarismCheckRequest
		if err := json.Unmarshal(msg.Body, &request); err != nil {
			return fmt.Errorf("failed to unmarshal analysis request: %w", err)
		}
		return h.HandleAnalysisRequest(ctx, request)

	case "batch.request":
		var request models.BatchAnalysisRequest
		if err := json.Unmarshal(msg.Body, &request); err != nil {
			return fmt.Errorf("failed to unmarshal batch request: %w", err)
		}
		return h.HandleBatchRequest(ctx, request)

	default:
		h.logger.Warn().Str("type", msgType).Msg("Unknown message type")
		return nil
	}
}
