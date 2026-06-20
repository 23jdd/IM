package rabbitmq

import (
	"context"
	"encoding/json"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	conn    *amqp.Connection
	channel *amqp.Channel
)

func InitRabbitMQ(url string) error {
	var err error
	conn, err = amqp.Dial(url)
	if err != nil {
		return err
	}

	channel, err = conn.Channel()
	if err != nil {
		return err
	}

	_, err = channel.QueueDeclare("im.message", true, false, false, false, nil)
	if err != nil {
		return err
	}

	_, err = channel.QueueDeclare("im.notification", true, false, false, false, nil)
	if err != nil {
		return err
	}

	_, err = channel.QueueDeclare("im.file", true, false, false, false, nil)
	if err != nil {
		return err
	}

	return nil
}

func CloseRabbitMQ() {
	if channel != nil {
		channel.Close()
	}
	if conn != nil {
		conn.Close()
	}
}

type MessageEvent struct {
	MsgId     string    `json:"msg_id"`
	FromUid   string    `json:"from_uid"`
	ToUid     string    `json:"to_uid"`
	GroupId   string    `json:"group_id,omitempty"`
	MsgType   byte      `json:"msg_type"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func PublishMessage(ctx context.Context, event *MessageEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return channel.PublishWithContext(ctx, "", "im.message", false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}

func ConsumeMessages(handler func(event *MessageEvent) error) error {
	msgs, err := channel.Consume("im.message", "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		for d := range msgs {
			var event MessageEvent
			if err := json.Unmarshal(d.Body, &event); err != nil {
				log.Println("rabbitmq: unmarshal failed:", err)
				d.Nack(false, false)
				continue
			}
			if err := handler(&event); err != nil {
				log.Println("rabbitmq: handler failed:", err)
				d.Nack(false, true)
				continue
			}
			d.Ack(false)
		}
	}()

	return nil
}

type NotificationEvent struct {
	Uid     string `json:"uid"`
	Type    string `json:"type"` // friend_request, system_notice
	Content string `json:"content"`
}

func PublishNotification(ctx context.Context, event *NotificationEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return channel.PublishWithContext(ctx, "", "im.notification", false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}
