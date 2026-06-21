package service

import (
	"IM/mongdb"
	"context"
	"testing"
	"time"
)

func TestGetChatHistorySingle(t *testing.T) {
	orig := getChatHistoryMongo
	defer func() { getChatHistoryMongo = orig }()

	var gotU1, gotU2 string
	var gotBefore time.Time
	var gotLimit int64
	getChatHistoryMongo = func(ctx context.Context, u1, u2 string, before time.Time, limit int64) ([]*mongdb.MessageDoc, error) {
		gotU1, gotU2, gotBefore, gotLimit = u1, u2, before, limit
		return []*mongdb.MessageDoc{
			{MsgId: "m1", FromUid: "me", ToUid: "peer", Content: "hi", Status: 2},
		}, nil
	}

	before := time.Now()
	out, err := GetChatHistory(context.Background(), "me", "peer", "", before, 0)
	if err != nil {
		t.Fatal(err)
	}
	if gotU1 != "me" || gotU2 != "peer" {
		t.Errorf("uids = %q,%q", gotU1, gotU2)
	}
	if !gotBefore.Equal(before) {
		t.Errorf("before not forwarded")
	}
	if gotLimit != defaultHistoryLimit {
		t.Errorf("limit = %d, want default %d", gotLimit, defaultHistoryLimit)
	}
	if len(out) != 1 || out[0].MsgId != "m1" || out[0].Content != "hi" || out[0].Status != 2 {
		t.Errorf("unexpected mapping: %+v", out)
	}
}

func TestGetChatHistoryGroup(t *testing.T) {
	orig := getGroupChatHistoryMongo
	defer func() { getGroupChatHistoryMongo = orig }()

	var gotGroup string
	getGroupChatHistoryMongo = func(ctx context.Context, gid string, before time.Time, limit int64) ([]*mongdb.MessageDoc, error) {
		gotGroup = gid
		return []*mongdb.MessageDoc{{MsgId: "g1", GroupId: gid, FromUid: "x"}}, nil
	}

	out, err := GetChatHistory(context.Background(), "me", "", "grp1", time.Time{}, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if gotGroup != "grp1" {
		t.Errorf("group = %q", gotGroup)
	}
	if len(out) != 1 || out[0].MsgId != "g1" {
		t.Errorf("unexpected group history: %+v", out)
	}
}

func TestGetChatHistoryLimitClamped(t *testing.T) {
	orig := getGroupChatHistoryMongo
	defer func() { getGroupChatHistoryMongo = orig }()
	var gotLimit int64
	getGroupChatHistoryMongo = func(ctx context.Context, gid string, before time.Time, limit int64) ([]*mongdb.MessageDoc, error) {
		gotLimit = limit
		return nil, nil
	}
	if _, err := GetChatHistory(context.Background(), "me", "", "g", time.Time{}, 1000); err != nil {
		t.Fatal(err)
	}
	if gotLimit != maxHistoryLimit {
		t.Errorf("limit = %d, want clamped to %d", gotLimit, maxHistoryLimit)
	}
}

func TestGetChatHistoryRequiresPeerOrGroup(t *testing.T) {
	if _, err := GetChatHistory(context.Background(), "me", "", "", time.Time{}, 10); err == nil {
		t.Fatal("expected error when neither peer nor group provided")
	}
}
