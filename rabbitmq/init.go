package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// ErrNotInitialized 表示 RabbitMQ 尚未初始化（连接失败或未调用 InitRabbitMQ）。
var ErrNotInitialized = errors.New("rabbitmq not initialized")

// conn、channel 为包级全局的 RabbitMQ 连接与信道，由 InitRabbitMQ 初始化。
var (
	conn    *amqp.Connection
	channel *amqp.Channel
)

// InitRabbitMQ 建立 RabbitMQ 连接与信道，并声明所需的持久化队列。
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

	// 声明持久化的消息队列
	_, err = channel.QueueDeclare("im.message", true, false, false, false, nil)
	if err != nil {
		return err
	}

	// 声明持久化的通知队列
	_, err = channel.QueueDeclare("im.notification", true, false, false, false, nil)
	if err != nil {
		return err
	}

	// 声明持久化的文件队列
	_, err = channel.QueueDeclare("im.file", true, false, false, false, nil)
	if err != nil {
		return err
	}

	return nil
}

// CloseRabbitMQ 关闭 RabbitMQ 信道与连接。
func CloseRabbitMQ() {
	if channel != nil {
		channel.Close()
	}
	if conn != nil {
		conn.Close()
	}
}

// MessageEvent 聊天消息事件，承载单聊/群聊消息的投递信息。
type MessageEvent struct {
	MsgId     string    `json:"msg_id"`
	FromUid   string    `json:"from_uid"`
	ToUid     string    `json:"to_uid"`
	GroupId   string    `json:"group_id,omitempty"`
	MsgType   byte      `json:"msg_type"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// PublishMessage 将消息事件序列化后发布到 im.message 队列，采用持久化投递。
func PublishMessage(ctx context.Context, event *MessageEvent) error {
	if channel == nil {
		return ErrNotInitialized
	}
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

// ConsumeMessages 启动消费协程处理 im.message 队列，手动确认投递结果。
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
				d.Nack(false, false) // 反序列化失败，不重新入队，丢弃该消息
				continue
			}
			if err := handler(&event); err != nil {
				log.Println("rabbitmq: handler failed:", err)
				d.Nack(false, true) // 处理失败，重新入队以便重试
				continue
			}
			d.Ack(false) // 处理成功，确认消息
		}
	}()

	return nil
}

// NotificationEvent 通知事件，如好友请求、系统通知等。
type NotificationEvent struct {
	Uid     string `json:"uid"`
	Type    string `json:"type"` // friend_request, system_notice
	Content string `json:"content"`
}

// PublishNotification 将通知事件序列化后发布到 im.notification 队列。
func PublishNotification(ctx context.Context, event *NotificationEvent) error {
	if channel == nil {
		return ErrNotInitialized
	}
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return channel.PublishWithContext(ctx, "", "im.notification", false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}
