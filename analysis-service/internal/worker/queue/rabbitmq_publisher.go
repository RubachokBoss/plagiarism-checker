package queue

import (
	"context"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
)

type RabbitMQPublisher interface {
	Publish(ctx context.Context, exchange, routingKey string, body []byte) error
	PublishWithDelay(ctx context.Context, exchange, routingKey string, body []byte, delay time.Duration) error
	Close() error
}

type rabbitMQPublisher struct {
	channel *amqp.Channel // amqp091-go использует amqp.Channel
	logger  zerolog.Logger
}

func NewRabbitMQPublisher(channel *amqp.Channel, logger zerolog.Logger) RabbitMQPublisher {
	return &rabbitMQPublisher{
		channel: channel,
		logger:  logger,
	}
}

func (p *rabbitMQPublisher) Publish(ctx context.Context, exchange, routingKey string, body []byte) error {
	// Set up context with timeout
	publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return p.channel.PublishWithContext(
		publishCtx,
		exchange,   // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
		},
	)
}

func (p *rabbitMQPublisher) PublishWithDelay(ctx context.Context, exchange, routingKey string, body []byte, delay time.Duration) error {
	// For delayed messages, we need to use delayed message exchange plugin
	// This is a simplified implementation

	// Set up context with timeout
	publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Add delay header if supported
	headers := amqp.Table{}
	if delay > 0 {
		headers["x-delay"] = int32(delay.Milliseconds())
	}

	return p.channel.PublishWithContext(
		publishCtx,
		exchange,   // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			Headers:      headers,
		},
	)
}

func (p *rabbitMQPublisher) Close() error {
	// Channel will be closed by parent
	p.logger.Info().Msg("RabbitMQ publisher closed")
	return nil
}
