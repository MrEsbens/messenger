package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MrEsbens/messenger-message-service/internal/domain"
	amqp "github.com/rabbitmq/amqp091-go"
)

type ModerationProducerInterface interface {
	Publish(ctx context.Context, req *domain.ModerationRequest) error
	Close() error
}

type moderationProducer struct {
	conn         *amqp.Connection
	channel      *amqp.Channel
	exchangeName string
}

func NewModerationProducer(amqpURL, exchangeName string) (ModerationProducerInterface, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Объявляем exchange (тип: topic, durable)
	if err := ch.ExchangeDeclare(
		exchangeName, // name
		"topic",      // type
		true,         // durable
		false,        // auto-delete
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	return &moderationProducer{
		conn:         conn,
		channel:      ch,
		exchangeName: exchangeName,
	}, nil
}

func (p *moderationProducer) Publish(ctx context.Context, req *domain.ModerationRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal moderation request: %w", err)
	}

	// Routing key: moderation.request.<server_id> или moderation.request.global
	routingKey := "moderation.request."
	if req.ServerID != "" {
		routingKey += req.ServerID
	} else {
		routingKey += "global"
	}

	return p.channel.PublishWithContext(ctx,
		p.exchangeName, // exchange
		routingKey,     // routing key
		false,          // mandatory
		false,          // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent, // durable message
			Timestamp:    time.Now(),
			MessageId:    req.EventID,
		},
	)
}

func (p *moderationProducer) Close() error {
	if p.channel != nil {
		_ = p.channel.Close()
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}
