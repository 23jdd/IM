package tcp

import (
	"IM/service"
	"IM/tcp/Message"
	"context"
	"encoding/json"
	"log"
	"time"
)

type TextChatPayload struct {
	ToUid   string `json:"to_uid"`
	Content string `json:"content"`
}

// RealtimeTextPayload 是服务端路由给接收方的实时文本帧体，携带发送者信息，
// 使接收端能正确归属消息（修复"实时帧丢失 from_uid"问题）。
type RealtimeTextPayload struct {
	FromUid   string    `json:"from_uid"`
	MsgId     string    `json:"msg_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// BuildRealtimeText 构造实时文本帧体（JSON）。
func BuildRealtimeText(fromUid, msgId, content string, createdAt time.Time) []byte {
	data, _ := json.Marshal(RealtimeTextPayload{
		FromUid:   fromUid,
		MsgId:     msgId,
		Content:   content,
		CreatedAt: createdAt,
	})
	return data
}

func ChatMessageHandler(m *Message.Message, c *Client) {
	var payload TextChatPayload
	if err := json.Unmarshal(m.Data, &payload); err != nil {
		log.Println("chat: unmarshal failed:", err)
		return
	}

	if payload.ToUid == "" || payload.Content == "" {
		return
	}

	ctx := context.Background()
	msg, err := service.SendChatMessage(ctx, c.UID(), payload.ToUid, Message.Text, payload.Content)
	if err != nil {
		log.Println("chat: save message failed:", err)
		c.SendNack(m.GetKey())
		return
	}

	c.SendAck(m.GetKey())

	err = c.server.RouteTo(payload.ToUid, Message.NewMessage(
		Message.Text, 0,
		BuildRealtimeText(c.UID(), msg.MsgId, msg.Content, msg.CreatedAt),
	))
	if err != nil {
		log.Printf("chat: route to %s failed (offline): %v", payload.ToUid, err)
	}
}

func OfflineSyncHandler(m *Message.Message, c *Client) {
	ctx := context.Background()
	msgs, err := service.GetOfflineMessages(ctx, c.UID())
	if err != nil {
		log.Println("offline sync: query failed:", err)
		return
	}

	for _, msg := range msgs {
		data, _ := json.Marshal(msg)
		if err := c.SendBlob(0, data); err != nil {
			log.Println("offline sync: send failed:", err)
			return
		}
	}

	if len(msgs) > 0 {
		ids := make([]string, len(msgs))
		for i, msg := range msgs {
			ids[i] = msg.MsgId
		}
		service.MarkMessagesRead(ctx, ids)
	}
}

func init() {
	RegisterRoute(Message.Text, ChatMessageHandler)
	RegisterRoute(Message.Json, OfflineSyncHandler)
}
