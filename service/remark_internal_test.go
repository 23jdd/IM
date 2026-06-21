package service

import (
	"context"
	"testing"
)

func TestUpdateFriendRemark(t *testing.T) {
	orig := updateFriendRemark
	defer func() { updateFriendRemark = orig }()

	var gotUid, gotFriend, gotRemark string
	updateFriendRemark = func(ctx context.Context, uid, friendUid, remark string) error {
		gotUid, gotFriend, gotRemark = uid, friendUid, remark
		return nil
	}
	if err := UpdateFriendRemark(context.Background(), "me", "bob", "Bobby"); err != nil {
		t.Fatal(err)
	}
	if gotUid != "me" || gotFriend != "bob" || gotRemark != "Bobby" {
		t.Errorf("args = (%s,%s,%s), want (me,bob,Bobby)", gotUid, gotFriend, gotRemark)
	}
}

func TestUpdateFriendRemarkSelfRejected(t *testing.T) {
	orig := updateFriendRemark
	defer func() { updateFriendRemark = orig }()
	called := false
	updateFriendRemark = func(ctx context.Context, uid, friendUid, remark string) error {
		called = true
		return nil
	}
	if err := UpdateFriendRemark(context.Background(), "me", "me", "x"); err == nil {
		t.Fatal("expected error setting remark on self")
	}
	if called {
		t.Error("update must not be called for self")
	}
}
