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

// SendGroupMessage 持久化一条群消息（带 group_id），并 best-effort 归档。
func SendGroupMessage(ctx context.Context, fromUid, groupId string, msgType byte, content string) (*model.ChatMessage, error) {
	msg := &model.ChatMessage{
		MsgId:     strconv.FormatUint(utils.GenerateId(), 10),
		FromUid:   fromUid,
		GroupId:   groupId,
		MsgType:   msgType,
		Content:   content,
		Status:    model.MsgStatusUnread,
		CreatedAt: time.Now(),
	}

	if err := insertChatMessage(ctx, msg); err != nil {
		return nil, err
	}

	if err := publishChatEvent(ctx, &rabbitmq.MessageEvent{
		MsgId:     msg.MsgId,
		FromUid:   msg.FromUid,
		ToUid:     msg.ToUid,
		GroupId:   msg.GroupId,
		MsgType:   msg.MsgType,
		Content:   msg.Content,
		CreatedAt: msg.CreatedAt,
	}); err != nil {
		log.Printf("archive publish failed for group msg %s: %v", msg.MsgId, err)
	}

	return msg, nil
}

func GetOfflineMessages(ctx context.Context, uid string) ([]*model.ChatMessage, error) {
	return mysql.FindOfflineMessages(ctx, uid)
}

func MarkMessagesRead(ctx context.Context, msgIds []string) error {
	return mysql.MarkMessagesRead(ctx, msgIds)
}

// ConversationItem 是会话列表项（最近联系人 + 最后一条消息）。
type ConversationItem struct {
	Peer    string    `json:"peer"`
	Content string    `json:"content"`
	Time    time.Time `json:"time"`
}

// findRecentMessages 便于测试注入。
var findRecentMessages = mysql.FindRecentMessages

// GetConversations 基于最近消息聚合出会话列表（按对端去重，取最近一条）。
func GetConversations(ctx context.Context, uid string) ([]*ConversationItem, error) {
	msgs, err := findRecentMessages(ctx, uid, 500)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	out := make([]*ConversationItem, 0)
	for _, m := range msgs { // 已按时间倒序
		peer := m.ToUid
		if m.FromUid != uid {
			peer = m.FromUid
		}
		if peer == "" || peer == uid || seen[peer] {
			continue
		}
		seen[peer] = true
		out = append(out, &ConversationItem{Peer: peer, Content: m.Content, Time: m.CreatedAt})
	}
	return out, nil
}
