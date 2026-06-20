package Tests

import (
	"IM/model"
	"IM/service"
	"testing"
	"time"
)

func TestSendFriendRequestSelfValidation(t *testing.T) {
	err := service.SendFriendRequest(nil, "user_a", "user_a", "")
	if err == nil {
		t.Error("expected error for self-friend request")
	}
}

func TestFriendRelationModelComplete(t *testing.T) {
	now := time.Now()
	f := &model.FriendRelation{
		Id:         1,
		Uid:        "user_a",
		FriendUid:  "user_b",
		Status:     model.FriendStatusAccepted,
		Remark:     "best friend",
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if f.Status != model.FriendStatusAccepted {
		t.Errorf("Status = %d, want accepted", f.Status)
	}
	if f.Remark != "best friend" {
		t.Errorf("Remark = %s, want best friend", f.Remark)
	}
}

func TestGroupModelComplete(t *testing.T) {
	now := time.Now()
	g := &model.GroupInfo{
		GroupId:     "g_001",
		Name:        "friends",
		OwnerUid:    "user_a",
		Description: "test group",
		MemberCount: 10,
		Status:      model.GroupStatusNormal,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if g.MemberCount != 10 {
		t.Errorf("MemberCount = %d, want 10", g.MemberCount)
	}
	if g.Status != model.GroupStatusNormal {
		t.Errorf("Status = %d, want normal", g.Status)
	}
}

func TestGroupMemberModelComplete(t *testing.T) {
	now := time.Now()
	muteUntil := now.Add(time.Hour)
	m := &model.GroupMember{
		Id:        1,
		GroupId:   "g_001",
		Uid:       "user_b",
		Role:      model.GroupRoleMember,
		Nickname:  "Bob",
		MuteUntil: &muteUntil,
		JoinedAt:  now,
	}

	if m.Role != model.GroupRoleMember {
		t.Errorf("Role = %d, want member", m.Role)
	}
	if m.Nickname != "Bob" {
		t.Errorf("Nickname = %s, want Bob", m.Nickname)
	}
	if m.MuteUntil == nil {
		t.Error("MuteUntil should not be nil")
	}
}

func TestMessageStatusTransitions(t *testing.T) {
	if model.MsgStatusUnread != 0 {
		t.Error("unread must be 0")
	}
	if model.MsgStatusRead != 1 {
		t.Error("read must be 1")
	}
	if model.MsgStatusRevoked != 2 {
		t.Error("revoked must be 2")
	}
}
