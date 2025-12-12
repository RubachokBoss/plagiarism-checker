package queue

import (
	"context"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
)

type RabbitMQMessage struct {
	Body      []byte
	Timestamp time.Time
	Ack       func(multiple bool) error
	Nack      func(multiple bool, requeue bool) error
}

type RabbitMQConsumer interface {
	Consume(ctx context.Context) (<-chan RabbitMQMessage, error)
	GetQueueLength() (int, error)
	Close() error
}

type rabbitMQConsumer struct {
	channel     *amqp.Channel // amqp091-go использует amqp.Channel
	queue       string
	consumerTag string
	logger      zerolog.Logger
}

func NewRabbitMQConsumer(channel *amqp.Channel, queue, consumerTag string, logger zerolog.Logger) RabbitMQConsumer {
	return &rabbitMQConsumer{
		channel:     channel,
		queue:       queue,
		consumerTag: consumerTag,
		logger:      logger,
	}
}

func (c *rabbitMQConsumer) Consume(ctx context.Context) (<-chan RabbitMQMessage, error) {
	// Set prefetch count
	err := c.channel.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		return nil, err
	}

	// Start consuming
	msgs, err := c.channel.Consume(
		c.queue,       // queue
		c.consumerTag, // consumer
		false,         // auto-ack
		false,         // exclusive
		false,         // no-local
		false,         // no-wait
		nil,           // args
	)
	if err != nil {
		return nil, err
	}

	// Convert to our message type
	output := make(chan RabbitMQMessage)

	go func() {
		defer close(output)

		for {
			select {
			case <-ctx.Done():
				c.logger.Info().Msg("Stopping RabbitMQ consumer")
				return
			case msg, ok := <-msgs:
				if !ok {
					c.logger.Warn().Msg("RabbitMQ message channel closed")
					return
				}

				// Create our message wrapper
				rabbitMsg := RabbitMQMessage{
					Body:      msg.Body,
					Timestamp: msg.Timestamp,
					Ack:       msg.Ack,
					Nack:      msg.Nack,
				}

				// Send to output channel
				select {
				case output <- rabbitMsg:
					// Message sent successfully
				case <-ctx.Done():
					// Context cancelled, nack the message
					msg.Nack(false, true)
					return
				}
			}
		}
	}()

	c.logger.Info().
		Str("queue", c.queue).
		Str("consumer_tag", c.consumerTag).
		Msg("RabbitMQ consumer started")

	return output, nil
}

func (c *rabbitMQConsumer) GetQueueLength() (int, error) {
	queue, err := c.channel.QueueDeclarePassive(
		c.queue, // name
		true,    // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	if err != nil {
		return 0, err
	}

	return queue.Messages, nil
}

func (c *rabbitMQConsumer) Close() error {
	if c.channel != nil {
		// Cancel consumer
		if err := c.channel.Cancel(c.consumerTag, false); err != nil {
			c.logger.Error().Err(err).Msg("Failed to cancel RabbitMQ consumer")
		}

		// Channel will be closed by parent
	}

	c.logger.Info().Msg("RabbitMQ consumer closed")
	return nil
}
