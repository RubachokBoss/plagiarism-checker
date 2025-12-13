package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/RubachokBoss/plagiarism-checker/work-service/internal/models"
	"github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
)

type RabbitMQClient interface {
	PublishWorkCreated(ctx context.Context, event *models.WorkCreatedEvent) error
	Close() error
}

type rabbitMQClient struct {
	conn       *amqp091.Connection
	channel    *amqp091.Channel
	exchange   string
	routingKey string
	queueName  string
	logger     zerolog.Logger
}

func NewRabbitMQClient(url, exchange, routingKey, queueName string, logger zerolog.Logger) (RabbitMQClient, error) {
	conn, err := amqp091.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	err = channel.ExchangeDeclare(
		exchange, // name
		"direct", // type
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	queue, err := channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	err = channel.QueueBind(
		queue.Name, // queue name
		routingKey, // routing key
		exchange,   // exchange
		false,      // no-wait
		nil,        // arguments
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to bind queue: %w", err)
	}

	logger.Info().
		Str("exchange", exchange).
		Str("queue", queue.Name).
		Str("routing_key", routingKey).
		Msg("Connected to RabbitMQ")

	return &rabbitMQClient{
		conn:       conn,
		channel:    channel,
		exchange:   exchange,
		routingKey: routingKey,
		queueName:  queue.Name,
		logger:     logger,
	}, nil
}

func (c *rabbitMQClient) PublishWorkCreated(ctx context.Context, event *models.WorkCreatedEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = c.channel.PublishWithContext(
		publishCtx,
		c.exchange,   // exchange
		c.routingKey, // routing key
		false,        // mandatory
		false,        // immediate
		amqp091.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp091.Persistent, // Сохраняем сообщение
			Timestamp:    time.Now(),
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	c.logger.Info().
		Str("work_id", event.WorkID).
		Str("file_id", event.FileID).
		Msg("Work created event published")

	return nil
}

func (c *rabbitMQClient) Close() error {
	if c.channel != nil {
		if err := c.channel.Close(); err != nil {
			c.logger.Error().Err(err).Msg("Failed to close RabbitMQ channel")
		}
	}

	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			c.logger.Error().Err(err).Msg("Failed to close RabbitMQ connection")
		}
	}

	return nil
}
