package service

import (
	"IM/model"
	"context"
	"errors"
	"testing"
)

func TestBlockFriendUpserts(t *testing.T) {
	orig := upsertBlocked
	defer func() { upsertBlocked = orig }()

	var gotUid, gotFriend string
	upsertBlocked = func(ctx context.Context, uid, friendUid string) error {
		gotUid, gotFriend = uid, friendUid
		return nil
	}
	if err := BlockFriend(context.Background(), "me", "bad"); err != nil {
		t.Fatal(err)
	}
	if gotUid != "me" || gotFriend != "bad" {
		t.Errorf("upsert args = (%s,%s), want (me,bad)", gotUid, gotFriend)
	}
}

func TestBlockFriendSelfRejected(t *testing.T) {
	orig := upsertBlocked
	defer func() { upsertBlocked = orig }()
	called := false
	upsertBlocked = func(ctx context.Context, uid, friendUid string) error { called = true; return nil }

	if err := BlockFriend(context.Background(), "me", "me"); err == nil {
		t.Fatal("expected error blocking self")
	}
	if called {
		t.Error("upsert must not be called for self-block")
	}
}

func TestUnblockFriendDeletesRelation(t *testing.T) {
	orig := deleteFriendRel
	defer func() { deleteFriendRel = orig }()
	var gotUid, gotFriend string
	deleteFriendRel = func(ctx context.Context, uid, friendUid string) error {
		gotUid, gotFriend = uid, friendUid
		return nil
	}
	if err := UnblockFriend(context.Background(), "me", "bad"); err != nil {
		t.Fatal(err)
	}
	if gotUid != "me" || gotFriend != "bad" {
		t.Errorf("delete args = (%s,%s), want (me,bad)", gotUid, gotFriend)
	}
}

func TestIsBlockedDelegates(t *testing.T) {
	orig := isBlockedBetween
	defer func() { isBlockedBetween = orig }()
	isBlockedBetween = func(ctx context.Context, a, b string) (bool, error) {
		return a == "x" && b == "y", nil
	}
	if ok, _ := IsBlocked(context.Background(), "x", "y"); !ok {
		t.Error("expected blocked true for (x,y)")
	}
	if ok, _ := IsBlocked(context.Background(), "a", "b"); ok {
		t.Error("expected blocked false for (a,b)")
	}
}

func TestGetBlockedListDelegates(t *testing.T) {
	orig := findBlockedList
	defer func() { findBlockedList = orig }()
	findBlockedList = func(ctx context.Context, uid string) ([]*model.FriendInfo, error) {
		if uid != "me" {
			return nil, errors.New("unexpected uid")
		}
		return []*model.FriendInfo{{Uid: "bad", Name: "Bad"}}, nil
	}
	list, err := GetBlockedList(context.Background(), "me")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Uid != "bad" {
		t.Errorf("unexpected blocked list: %+v", list)
	}
}
