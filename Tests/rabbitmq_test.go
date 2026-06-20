package Tests

import (
	"IM/model"
	"IM/rabbitmq"
	"encoding/json"
	"testing"
	"time"
)

func TestMessageEventJSON(t *testing.T) {
	now := time.Now()
	event := &rabbitmq.MessageEvent{
		MsgId:     "msg_001",
		FromUid:   "user_a",
		ToUid:     "user_b",
		GroupId:   "",
		MsgType:   5,
		Content:   "hello world",
		CreatedAt: now,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded rabbitmq.MessageEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.MsgId != "msg_001" {
		t.Errorf("MsgId = %s, want msg_001", decoded.MsgId)
	}
	if decoded.Content != "hello world" {
		t.Errorf("Content = %s, want hello world", decoded.Content)
	}
}

func TestNotificationEventJSON(t *testing.T) {
	event := &rabbitmq.NotificationEvent{
		Uid:     "user_a",
		Type:    "friend_request",
		Content: "user_b wants to be your friend",
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded rabbitmq.NotificationEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.Type != "friend_request" {
		t.Errorf("Type = %s, want friend_request", decoded.Type)
	}
}

func TestMessageEventToModelConversion(t *testing.T) {
	now := time.Now()
	event := &rabbitmq.MessageEvent{
		MsgId:     "msg_002",
		FromUid:   "user_a",
		ToUid:     "user_b",
		MsgType:   5,
		Content:   "test message",
		CreatedAt: now,
	}

	msg := &model.ChatMessage{
		MsgId:     event.MsgId,
		FromUid:   event.FromUid,
		ToUid:     event.ToUid,
		GroupId:   event.GroupId,
		MsgType:   event.MsgType,
		Content:   event.Content,
		Status:    model.MsgStatusUnread,
		CreatedAt: event.CreatedAt,
	}

	if msg.MsgId != event.MsgId {
		t.Errorf("MsgId mismatch")
	}
	if msg.Status != model.MsgStatusUnread {
		t.Errorf("Status = %d, want unread", msg.Status)
	}
}
