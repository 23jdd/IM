package service

import (
	"IM/model"
	"context"
	"encoding/json"
	"testing"
)

func TestSendFriendRequestSingleDirection(t *testing.T) {
	orig := insertFriend
	defer func() { insertFriend = orig }()

	var inserts []*model.FriendRelation
	insertFriend = func(ctx context.Context, f *model.FriendRelation) error {
		inserts = append(inserts, f)
		return nil
	}

	if err := SendFriendRequest(context.Background(), "a", "b", "hi bob"); err != nil {
		t.Fatal(err)
	}
	if len(inserts) != 1 {
		t.Fatalf("expected exactly 1 insert (single direction), got %d", len(inserts))
	}
	got := inserts[0]
	if got.Uid != "a" || got.FriendUid != "b" || got.Status != model.FriendStatusPending || got.Remark != "hi bob" {
		t.Errorf("unexpected request record: %+v", got)
	}
}

func TestSendFriendRequestSelfRejected(t *testing.T) {
	orig := insertFriend
	defer func() { insertFriend = orig }()

	called := false
	insertFriend = func(ctx context.Context, f *model.FriendRelation) error {
		called = true
		return nil
	}

	if err := SendFriendRequest(context.Background(), "a", "a", ""); err == nil {
		t.Fatal("expected error adding yourself")
	}
	if called {
		t.Error("insertFriend must not be called for self-add")
	}
}

func TestAcceptFriendRequestEstablishesBidirectional(t *testing.T) {
	origUpd := updateFriendStatus
	origIns := insertFriend
	defer func() { updateFriendStatus = origUpd; insertFriend = origIns }()

	var updUid, updFriend string
	var updStatus byte
	updateFriendStatus = func(ctx context.Context, uid, friendUid string, status byte) error {
		updUid, updFriend, updStatus = uid, friendUid, status
		return nil
	}
	var inserted *model.FriendRelation
	insertFriend = func(ctx context.Context, f *model.FriendRelation) error {
		inserted = f
		return nil
	}

	// accepter=b 接受 requester=a 的申请
	if err := AcceptFriendRequest(context.Background(), "b", "a"); err != nil {
		t.Fatal(err)
	}
	// 把 a->b 置为 accepted
	if updUid != "a" || updFriend != "b" || updStatus != model.FriendStatusAccepted {
		t.Errorf("updateFriendStatus = (%s,%s,%d), want (a,b,accepted)", updUid, updFriend, updStatus)
	}
	// 建立反向 b->a accepted
	if inserted == nil || inserted.Uid != "b" || inserted.FriendUid != "a" ||
		inserted.Status != model.FriendStatusAccepted {
		t.Errorf("reverse accepted relation wrong: %+v", inserted)
	}
}

func TestSendFriendRequestNotifiesTarget(t *testing.T) {
	origIns := insertFriend
	defer func() { insertFriend = origIns; SetNotifier(nil) }()
	insertFriend = func(ctx context.Context, f *model.FriendRelation) error { return nil }

	var gotUid string
	var gotPayload map[string]any
	SetNotifier(func(toUid string, payload []byte) {
		gotUid = toUid
		_ = json.Unmarshal(payload, &gotPayload)
	})

	if err := SendFriendRequest(context.Background(), "a", "b", "hi bob"); err != nil {
		t.Fatal(err)
	}
	if gotUid != "b" {
		t.Errorf("notify target = %s, want b", gotUid)
	}
	if gotPayload["event"] != "friend_request" || gotPayload["from_uid"] != "a" || gotPayload["remark"] != "hi bob" {
		t.Errorf("unexpected notify payload: %+v", gotPayload)
	}
}

func TestAcceptFriendRequestNotifiesRequester(t *testing.T) {
	origUpd := updateFriendStatus
	origIns := insertFriend
	defer func() { updateFriendStatus = origUpd; insertFriend = origIns; SetNotifier(nil) }()
	updateFriendStatus = func(ctx context.Context, uid, friendUid string, status byte) error { return nil }
	insertFriend = func(ctx context.Context, f *model.FriendRelation) error { return nil }

	var gotUid string
	var gotPayload map[string]any
	SetNotifier(func(toUid string, payload []byte) {
		gotUid = toUid
		_ = json.Unmarshal(payload, &gotPayload)
	})

	if err := AcceptFriendRequest(context.Background(), "b", "a"); err != nil {
		t.Fatal(err)
	}
	if gotUid != "a" {
		t.Errorf("notify target = %s, want a (requester)", gotUid)
	}
	if gotPayload["event"] != "friend_accepted" || gotPayload["from_uid"] != "b" {
		t.Errorf("unexpected notify payload: %+v", gotPayload)
	}
}
