package Tests

import (
	"IM/model"
	"IM/mongdb"
	"encoding/json"
	"testing"
	"time"
)

func TestChatMessageModel(t *testing.T) {
	msg := &model.ChatMessage{
		MsgId:     "msg_001",
		FromUid:   "user_a",
		ToUid:     "user_b",
		MsgType:   5,
		Content:   "hello",
		Status:    model.MsgStatusUnread,
		CreatedAt: time.Now(),
	}

	if msg.MsgId != "msg_001" {
		t.Errorf("MsgId = %s, want msg_001", msg.MsgId)
	}
	if msg.Status != model.MsgStatusUnread {
		t.Errorf("Status = %d, want %d", msg.Status, model.MsgStatusUnread)
	}
}

func TestChatMessageJSON(t *testing.T) {
	msg := &model.ChatMessage{
		MsgId:     "msg_001",
		FromUid:   "user_a",
		ToUid:     "user_b",
		MsgType:   5,
		Content:   "你好",
		Status:    model.MsgStatusUnread,
		CreatedAt: time.Now(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded model.ChatMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.Content != "你好" {
		t.Errorf("Content = %s, want 你好", decoded.Content)
	}
	if decoded.Status != model.MsgStatusUnread {
		t.Errorf("Status = %d, want %d", decoded.Status, model.MsgStatusUnread)
	}
}

func TestFriendRelationModel(t *testing.T) {
	f := &model.FriendRelation{
		Uid:       "user_a",
		FriendUid: "user_b",
		Status:    model.FriendStatusPending,
		Remark:    "好友备注",
		CreatedAt: time.Now(),
	}

	if f.Status != model.FriendStatusPending {
		t.Errorf("Status = %d, want %d", f.Status, model.FriendStatusPending)
	}
	if f.Uid != "user_a" {
		t.Errorf("Uid = %s, want user_a", f.Uid)
	}
}

func TestFriendStatusConstants(t *testing.T) {
	if model.FriendStatusPending != 0 {
		t.Errorf("Pending = %d, want 0", model.FriendStatusPending)
	}
	if model.FriendStatusAccepted != 1 {
		t.Errorf("Accepted = %d, want 1", model.FriendStatusAccepted)
	}
	if model.FriendStatusBlocked != 2 {
		t.Errorf("Blocked = %d, want 2", model.FriendStatusBlocked)
	}
}

func TestGroupInfoModel(t *testing.T) {
	g := &model.GroupInfo{
		GroupId:     "group_001",
		Name:        "测试群组",
		OwnerUid:    "user_a",
		MemberCount: 3,
		Status:      model.GroupStatusNormal,
		CreatedAt:   time.Now(),
	}

	if g.Status != model.GroupStatusNormal {
		t.Errorf("Status = %d, want %d", g.Status, model.GroupStatusNormal)
	}
	if g.OwnerUid != "user_a" {
		t.Errorf("OwnerUid = %s, want user_a", g.OwnerUid)
	}
}

func TestGroupMemberModel(t *testing.T) {
	muteTime := time.Now().Add(time.Hour)
	m := &model.GroupMember{
		GroupId:   "group_001",
		Uid:       "user_b",
		Role:      model.GroupRoleAdmin,
		Nickname:  "管理员",
		MuteUntil: &muteTime,
		JoinedAt:  time.Now(),
	}

	if m.Role != model.GroupRoleAdmin {
		t.Errorf("Role = %d, want %d", m.Role, model.GroupRoleAdmin)
	}
	if m.MuteUntil == nil {
		t.Fatal("MuteUntil should not be nil")
	}
	if m.Nickname != "管理员" {
		t.Errorf("Nickname = %s, want 管理员", m.Nickname)
	}
}

func TestGroupRoleConstants(t *testing.T) {
	if model.GroupRoleMember != 0 {
		t.Errorf("Member = %d, want 0", model.GroupRoleMember)
	}
	if model.GroupRoleAdmin != 1 {
		t.Errorf("Admin = %d, want 1", model.GroupRoleAdmin)
	}
	if model.GroupRoleOwner != 2 {
		t.Errorf("Owner = %d, want 2", model.GroupRoleOwner)
	}
}

func TestMessageDocConversion(t *testing.T) {
	now := time.Now()
	msg := &model.ChatMessage{
		MsgId:     "msg_001",
		FromUid:   "user_a",
		ToUid:     "user_b",
		MsgType:   5,
		Content:   "hello",
		Status:    model.MsgStatusUnread,
		CreatedAt: now,
	}

	doc := &mongdb.MessageDoc{
		MsgId:     msg.MsgId,
		FromUid:   msg.FromUid,
		ToUid:     msg.ToUid,
		MsgType:   msg.MsgType,
		Content:   msg.Content,
		Status:    msg.Status,
		CreatedAt: msg.CreatedAt,
	}

	if doc.MsgId != msg.MsgId {
		t.Errorf("doc.MsgId = %s, want %s", doc.MsgId, msg.MsgId)
	}
	if doc.ToUid != msg.ToUid {
		t.Errorf("doc.ToUid = %s, want %s", doc.ToUid, msg.ToUid)
	}
	if doc.CreatedAt != now {
		t.Error("CreatedAt mismatch")
	}
}

func TestMessageStatusConstants(t *testing.T) {
	if model.MsgStatusUnread != 0 {
		t.Errorf("Unread = %d, want 0", model.MsgStatusUnread)
	}
	if model.MsgStatusRead != 1 {
		t.Errorf("Read = %d, want 1", model.MsgStatusRead)
	}
	if model.MsgStatusRevoked != 2 {
		t.Errorf("Revoked = %d, want 2", model.MsgStatusRevoked)
	}
}
