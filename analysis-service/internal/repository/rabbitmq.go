package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
)

type RabbitMQRepository interface {
	Publish(ctx context.Context, exchange, routingKey string, message []byte) error
	Consume(ctx context.Context, queue, consumer string) (<-chan amqp091.Delivery, error)
	SetupQueue(exchange, queue, routingKey string) error
	Close() error
}

type rabbitMQRepository struct {
	conn    *amqp091.Connection
	channel *amqp091.Channel
	logger  zerolog.Logger
}

func NewRabbitMQRepository(url string, logger zerolog.Logger) (RabbitMQRepository, error) {
	conn, err := amqp091.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	logger.Info().Msg("Connected to RabbitMQ")

	return &rabbitMQRepository{
		conn:    conn,
		channel: channel,
		logger:  logger,
	}, nil
}

// Замените методы Publish и Consume
func (r *rabbitMQRepository) Publish(ctx context.Context, exchange, routingKey string, message []byte) error {
	return r.channel.PublishWithContext(
		ctx,
		exchange,
		routingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         message,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
		},
	)
}

func (r *rabbitMQRepository) Consume(ctx context.Context, queue, consumer string) (<-chan amqp.Delivery, error) {
	err := r.channel.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	return r.channel.ConsumeWithContext(
		ctx,
		queue,
		consumer,
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
}

func (r *rabbitMQRepository) SetupQueue(exchange, queue, routingKey string) error {
	// Declare exchange
	err := r.channel.ExchangeDeclare(
		exchange, // name
		"direct", // type
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Declare queue
	q, err := r.channel.QueueDeclare(
		queue, // name
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue to exchange
	err = r.channel.QueueBind(
		q.Name,     // queue name
		routingKey, // routing key
		exchange,   // exchange
		false,      // no-wait
		nil,        // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	r.logger.Info().
		Str("exchange", exchange).
		Str("queue", q.Name).
		Str("routing_key", routingKey).
		Msg("RabbitMQ queue setup complete")

	return nil
}

func (r *rabbitMQRepository) Close() error {
	if r.channel != nil {
		if err := r.channel.Close(); err != nil {
			r.logger.Error().Err(err).Msg("Failed to close RabbitMQ channel")
		}
	}

	if r.conn != nil {
		if err := r.conn.Close(); err != nil {
			r.logger.Error().Err(err).Msg("Failed to close RabbitMQ connection")
		}
	}

	return nil
}

// PublishWorkCreated publishes a work created event
func (r *rabbitMQRepository) PublishWorkCreated(ctx context.Context, event interface{}) error {
	message, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	return r.Publish(ctx, "plagiarism_exchange", "work.created", message)
}

// PublishAnalysisCompleted publishes an analysis completed event
func (r *rabbitMQRepository) PublishAnalysisCompleted(ctx context.Context, event interface{}) error {
	message, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	return r.Publish(ctx, "plagiarism_exchange", "analysis.completed", message)
}
