package service

import (
	"IM/model"
	"IM/rabbitmq"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

// #4 归档链路：SendChatMessage 写库后必须发布 MQ 事件（best-effort）。

func TestSendChatMessagePublishesEvent(t *testing.T) {
	origInsert := insertChatMessage
	origPublish := publishChatEvent
	defer func() { insertChatMessage = origInsert; publishChatEvent = origPublish }()

	var inserted *model.ChatMessage
	insertChatMessage = func(ctx context.Context, msg *model.ChatMessage) error {
		inserted = msg
		return nil
	}
	var published *rabbitmq.MessageEvent
	publishChatEvent = func(ctx context.Context, ev *rabbitmq.MessageEvent) error {
		published = ev
		return nil
	}

	msg, err := SendChatMessage(context.Background(), "u1", "u2", 5, "hi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == nil {
		t.Fatal("returned msg is nil")
	}
	if msg.FromUid != "u1" || msg.ToUid != "u2" || msg.Content != "hi" || msg.MsgType != 5 {
		t.Errorf("msg fields wrong: %+v", msg)
	}
	if msg.Status != model.MsgStatusUnread {
		t.Errorf("status = %d, want unread", msg.Status)
	}
	if msg.MsgId == "" {
		t.Error("msg id should not be empty")
	}
	if inserted == nil {
		t.Fatal("insertChatMessage was not called")
	}
	if published == nil {
		t.Fatal("publishChatEvent was not called (archive link broken)")
	}
	if published.MsgId != msg.MsgId || published.FromUid != "u1" ||
		published.ToUid != "u2" || published.Content != "hi" {
		t.Errorf("published event mismatch: %+v", published)
	}
}

func TestSendChatMessageInsertErrorSkipsPublish(t *testing.T) {
	origInsert := insertChatMessage
	origPublish := publishChatEvent
	defer func() { insertChatMessage = origInsert; publishChatEvent = origPublish }()

	insertChatMessage = func(ctx context.Context, msg *model.ChatMessage) error {
		return errors.New("db down")
	}
	publishCalled := false
	publishChatEvent = func(ctx context.Context, ev *rabbitmq.MessageEvent) error {
		publishCalled = true
		return nil
	}

	_, err := SendChatMessage(context.Background(), "u1", "u2", 5, "hi")
	if err == nil {
		t.Fatal("expected error when insert fails")
	}
	if publishCalled {
		t.Error("publish must not be called when persistence fails")
	}
}

func TestSendChatMessagePublishErrorIsNonFatal(t *testing.T) {
	origInsert := insertChatMessage
	origPublish := publishChatEvent
	defer func() { insertChatMessage = origInsert; publishChatEvent = origPublish }()

	insertChatMessage = func(ctx context.Context, msg *model.ChatMessage) error { return nil }
	publishChatEvent = func(ctx context.Context, ev *rabbitmq.MessageEvent) error {
		return errors.New("mq down")
	}

	msg, err := SendChatMessage(context.Background(), "u1", "u2", 5, "hi")
	if err != nil {
		t.Fatalf("archive publish failure must not fail SendChatMessage: %v", err)
	}
	if msg == nil {
		t.Fatal("expected msg even when archive publish fails")
	}
}

func TestRecallMessageByOwnerWithinWindow(t *testing.T) {
	origFind := findMessageById
	origUpd := updateMessageStatus
	origMongo := updateMongoMessageStatus
	defer func() {
		findMessageById = origFind
		updateMessageStatus = origUpd
		updateMongoMessageStatus = origMongo
		SetNotifier(nil)
	}()

	findMessageById = func(ctx context.Context, msgId string) (*model.ChatMessage, error) {
		return &model.ChatMessage{MsgId: msgId, FromUid: "me", ToUid: "b", CreatedAt: time.Now()}, nil
	}
	var newStatus byte = 99
	updateMessageStatus = func(ctx context.Context, msgId string, status byte) error {
		newStatus = status
		return nil
	}
	updateMongoMessageStatus = func(ctx context.Context, msgId string, status byte) error {
		return nil
	}
	notified := map[string]map[string]any{}
	SetNotifier(func(toUid string, p []byte) {
		var m map[string]any
		_ = json.Unmarshal(p, &m)
		notified[toUid] = m
	})

	if err := RecallMessage(context.Background(), "me", "m1"); err != nil {
		t.Fatal(err)
	}
	if newStatus != model.MsgStatusRevoked {
		t.Errorf("status = %d, want revoked(%d)", newStatus, model.MsgStatusRevoked)
	}
	if notified["b"] == nil || notified["b"]["event"] != "recall" || notified["b"]["msg_id"] != "m1" {
		t.Errorf("peer not notified correctly: %+v", notified["b"])
	}
	if notified["me"] == nil {
		t.Error("self(other ends) should be notified")
	}
}

func TestRecallMessageByOtherRejected(t *testing.T) {
	origFind := findMessageById
	origUpd := updateMessageStatus
	defer func() { findMessageById = origFind; updateMessageStatus = origUpd }()

	findMessageById = func(ctx context.Context, msgId string) (*model.ChatMessage, error) {
		return &model.ChatMessage{MsgId: msgId, FromUid: "owner", CreatedAt: time.Now()}, nil
	}
	updCalled := false
	updateMessageStatus = func(ctx context.Context, msgId string, status byte) error {
		updCalled = true
		return nil
	}

	if err := RecallMessage(context.Background(), "intruder", "m1"); err == nil {
		t.Fatal("expected error recalling other's message")
	}
	if updCalled {
		t.Error("status must not be updated for non-owner")
	}
}

func TestRecallMessageExpired(t *testing.T) {
	origFind := findMessageById
	origUpd := updateMessageStatus
	defer func() { findMessageById = origFind; updateMessageStatus = origUpd }()

	findMessageById = func(ctx context.Context, msgId string) (*model.ChatMessage, error) {
		return &model.ChatMessage{MsgId: msgId, FromUid: "me", CreatedAt: time.Now().Add(-3 * time.Minute)}, nil
	}
	updCalled := false
	updateMessageStatus = func(ctx context.Context, msgId string, status byte) error {
		updCalled = true
		return nil
	}

	if err := RecallMessage(context.Background(), "me", "m1"); err == nil {
		t.Fatal("expected error recalling expired message")
	}
	if updCalled {
		t.Error("status must not be updated for expired message")
	}
}
