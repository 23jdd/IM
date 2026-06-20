package service

import (
	"IM/model"
	"IM/mysql"
	"IM/rabbitmq"
	"IM/utils"
	"context"
	"log"
	"strconv"
	"time"
)

// 通过函数变量注入持久化与发布依赖，便于单元测试替换。
var (
	insertChatMessage = mysql.InsertChatMessage
	publishChatEvent  = rabbitmq.PublishMessage
)

func SendChatMessage(ctx context.Context, fromUid, toUid string, msgType byte, content string) (*model.ChatMessage, error) {
	msg := &model.ChatMessage{
		MsgId:     strconv.FormatUint(utils.GenerateId(), 10),
		FromUid:   fromUid,
		ToUid:     toUid,
		MsgType:   msgType,
		Content:   content,
		Status:    model.MsgStatusUnread,
		CreatedAt: time.Now(),
	}

	if err := insertChatMessage(ctx, msg); err != nil {
		return nil, err
	}

	// best-effort：发布到 RabbitMQ，由消费者异步归档到 MongoDB。
	// 归档失败不影响主流程（消息已落 MySQL 离线表）。
	if err := publishChatEvent(ctx, &rabbitmq.MessageEvent{
		MsgId:     msg.MsgId,
		FromUid:   msg.FromUid,
		ToUid:     msg.ToUid,
		GroupId:   msg.GroupId,
		MsgType:   msg.MsgType,
		Content:   msg.Content,
		CreatedAt: msg.CreatedAt,
	}); err != nil {
		log.Printf("archive publish failed for msg %s: %v", msg.MsgId, err)
	}

	return msg, nil
}

func GetOfflineMessages(ctx context.Context, uid string) ([]*model.ChatMessage, error) {
	return mysql.FindOfflineMessages(ctx, uid)
}

func MarkMessagesRead(ctx context.Context, msgIds []string) error {
	return mysql.MarkMessagesRead(ctx, msgIds)
}
