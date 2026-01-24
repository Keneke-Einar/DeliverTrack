package messaging

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/streadway/amqp"
)

type MessageBroker struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

type PackageEvent struct {
	PackageID       string  `json:"package_id"`
	TrackingNumber  string  `json:"tracking_number"`
	Status          string  `json:"status"`
	Location        string  `json:"location"`
	Latitude        float64 `json:"latitude"`
	Longitude       float64 `json:"longitude"`
	Timestamp       string  `json:"timestamp"`
}

const (
	ExchangeName = "delivertrack"
	QueueName    = "package_updates"
	RoutingKey   = "package.update"
)

func NewMessageBroker(rabbitURL string) (*MessageBroker, error) {
	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare exchange
	err = channel.ExchangeDeclare(
		ExchangeName,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Declare queue
	_, err = channel.QueueDeclare(
		QueueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue to exchange
	err = channel.QueueBind(
		QueueName,
		RoutingKey,
		ExchangeName,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to bind queue: %w", err)
	}

	log.Println("Connected to RabbitMQ")
	return &MessageBroker{
		conn:    conn,
		channel: channel,
	}, nil
}

func (mb *MessageBroker) PublishPackageUpdate(event PackageEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	err = mb.channel.Publish(
		ExchangeName,
		RoutingKey,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Printf("Published package update: %s", event.TrackingNumber)
	return nil
}

func (mb *MessageBroker) ConsumePackageUpdates(handler func(PackageEvent) error) error {
	msgs, err := mb.channel.Consume(
		QueueName,
		"",
		false, // auto-ack
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	go func() {
		for msg := range msgs {
			var event PackageEvent
			if err := json.Unmarshal(msg.Body, &event); err != nil {
				log.Printf("Error unmarshaling message: %v", err)
				msg.Nack(false, false)
				continue
			}

			if err := handler(event); err != nil {
				log.Printf("Error handling message: %v", err)
				msg.Nack(false, true)
				continue
			}

			msg.Ack(false)
		}
	}()

	log.Println("Started consuming package updates")
	return nil
}

func (mb *MessageBroker) Close() {
	if mb.channel != nil {
		mb.channel.Close()
	}
	if mb.conn != nil {
		mb.conn.Close()
	}
}
