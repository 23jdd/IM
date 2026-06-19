package tcp

import (
	"IM/service"
	"IM/tcp/Message"
	"context"
	"encoding/json"
	"log"
)

type TextChatPayload struct {
	ToUid   string `json:"to_uid"`
	Content string `json:"content"`
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
	msg, err := service.SendChatMessage(ctx, c.uid, payload.ToUid, Message.Text, payload.Content)
	if err != nil {
		log.Println("chat: save message failed:", err)
		c.SendNack(m.GetKey())
		return
	}

	c.SendAck(m.GetKey())

	err = c.server.RouteTo(payload.ToUid, Message.NewMessage(
		Message.Text, 0,
		[]byte(msg.Content),
	))
	if err != nil {
		log.Printf("chat: route to %s failed (offline): %v", payload.ToUid, err)
	}
}

func OfflineSyncHandler(m *Message.Message, c *Client) {
	ctx := context.Background()
	msgs, err := service.GetOfflineMessages(ctx, c.uid)
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
