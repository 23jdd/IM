package service

import (
	"IM/model"
	"context"
	"testing"
	"time"
)

// P2：好友列表返回展示信息。
func TestGetFriendListReturnsItems(t *testing.T) {
	orig := findFriendList
	defer func() { findFriendList = orig }()

	findFriendList = func(ctx context.Context, uid string) ([]*model.FriendInfo, error) {
		return []*model.FriendInfo{
			{Uid: "u2", Name: "Bob", Remark: "bro"},
			{Uid: "u3", Name: "Carol"},
		}, nil
	}

	list, err := GetFriendList(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].Uid != "u2" || list[0].Name != "Bob" {
		t.Errorf("unexpected friend list: %+v", list)
	}
}

// P2：会话列表按对端去重，取最近一条。
func TestGetConversationsAggregation(t *testing.T) {
	orig := findRecentMessages
	defer func() { findRecentMessages = orig }()

	now := time.Now()
	findRecentMessages = func(ctx context.Context, uid string, limit int) ([]*model.ChatMessage, error) {
		// 已按时间倒序：me->a 最新, a->me 次之, b->me 最早
		return []*model.ChatMessage{
			{FromUid: "me", ToUid: "a", Content: "to a 2", CreatedAt: now},
			{FromUid: "a", ToUid: "me", Content: "from a 1", CreatedAt: now.Add(-time.Minute)},
			{FromUid: "b", ToUid: "me", Content: "from b", CreatedAt: now.Add(-2 * time.Minute)},
		}, nil
	}

	convs, err := GetConversations(context.Background(), "me")
	if err != nil {
		t.Fatal(err)
	}
	if len(convs) != 2 {
		t.Fatalf("expected 2 conversations, got %d: %+v", len(convs), convs)
	}
	if convs[0].Peer != "a" || convs[0].Content != "to a 2" {
		t.Errorf("conv[0] = %+v, want peer=a content='to a 2'", convs[0])
	}
	if convs[1].Peer != "b" {
		t.Errorf("conv[1].Peer = %s, want b", convs[1].Peer)
	}
}
