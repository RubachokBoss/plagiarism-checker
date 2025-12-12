package rabbitmq

import (
	"fmt"

	. "github.com/rabbitmq/amqp091-go"
)

func NewConnection(url string) (*Connection, error) {
	conn, err := Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	return conn, nil
}

func NewChannel(conn *Connection) (*Channel, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}
	return channel, nil
}
