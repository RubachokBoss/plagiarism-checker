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

	return nil
}

func (h *messageHandler) HandleAnalysisRequest(ctx context.Context, request models.PlagiarismCheckRequest) error {
	h.logger.Info().
		Str("work_id", request.WorkID).
		Str("file_id", request.FileID).
		Str("assignment_id", request.AssignmentID).
		Msg("Handling analysis request")

	return nil
}

func (h *messageHandler) HandleBatchRequest(ctx context.Context, request models.BatchAnalysisRequest) error {
	h.logger.Info().
		Int("work_count", len(request.WorkIDs)).
		Msg("Handling batch analysis request")

	for _, workID := range request.WorkIDs {
		h.logger.Debug().Str("work_id", workID).Msg("Processing work in batch")
	}

	return nil
}

func (h *messageHandler) ProcessMessage(ctx context.Context, msg RabbitMQMessage) error {
	var messageData map[string]interface{}
	if err := json.Unmarshal(msg.Body, &messageData); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

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
