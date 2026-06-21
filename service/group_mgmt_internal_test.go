package service

import (
	"IM/model"
	"context"
	"testing"
	"time"
)

func membersFn(ms ...*model.GroupMember) func(context.Context, string) ([]*model.GroupMember, error) {
	return func(ctx context.Context, groupId string) ([]*model.GroupMember, error) {
		return ms, nil
	}
}

func TestDisbandGroupByOwner(t *testing.T) {
	origG, origM, origS := findGroup, findGroupMembers, updateGroupStatus
	defer func() {
		findGroup, findGroupMembers, updateGroupStatus = origG, origM, origS
		SetNotifier(nil)
	}()

	findGroup = func(ctx context.Context, id string) (*model.GroupInfo, error) {
		return &model.GroupInfo{GroupId: id, OwnerUid: "owner"}, nil
	}
	findGroupMembers = membersFn(&model.GroupMember{Uid: "owner", Role: model.GroupRoleOwner}, &model.GroupMember{Uid: "b"})
	var gotStatus byte = 99
	updateGroupStatus = func(ctx context.Context, id string, s byte) error { gotStatus = s; return nil }
	notified := map[string]bool{}
	SetNotifier(func(uid string, _ []byte) { notified[uid] = true })

	if err := DisbandGroup(context.Background(), "g1", "owner"); err != nil {
		t.Fatal(err)
	}
	if gotStatus != model.GroupStatusDisbanded {
		t.Errorf("status = %d, want disbanded", gotStatus)
	}
	if !notified["owner"] || !notified["b"] {
		t.Errorf("all members should be notified: %+v", notified)
	}
}

func TestDisbandGroupByNonOwnerRejected(t *testing.T) {
	origG, origS := findGroup, updateGroupStatus
	defer func() { findGroup, updateGroupStatus = origG, origS }()

	findGroup = func(ctx context.Context, id string) (*model.GroupInfo, error) {
		return &model.GroupInfo{GroupId: id, OwnerUid: "owner"}, nil
	}
	called := false
	updateGroupStatus = func(ctx context.Context, id string, s byte) error { called = true; return nil }

	if err := DisbandGroup(context.Background(), "g1", "intruder"); err == nil {
		t.Fatal("expected error for non-owner disband")
	}
	if called {
		t.Error("status must not change for non-owner")
	}
}

func TestKickMemberByAdmin(t *testing.T) {
	origM, origD := findGroupMembers, deleteGroupMember
	defer func() { findGroupMembers, deleteGroupMember = origM, origD; SetNotifier(nil) }()

	findGroupMembers = membersFn(
		&model.GroupMember{Uid: "admin", Role: model.GroupRoleAdmin},
		&model.GroupMember{Uid: "victim", Role: model.GroupRoleMember},
	)
	var deleted string
	deleteGroupMember = func(ctx context.Context, gid, uid string) error { deleted = uid; return nil }
	SetNotifier(func(uid string, _ []byte) {})

	if err := KickMember(context.Background(), "g1", "admin", "victim"); err != nil {
		t.Fatal(err)
	}
	if deleted != "victim" {
		t.Errorf("deleted = %q, want victim", deleted)
	}
}

func TestKickMemberByMemberRejected(t *testing.T) {
	origM, origD := findGroupMembers, deleteGroupMember
	defer func() { findGroupMembers, deleteGroupMember = origM, origD }()

	findGroupMembers = membersFn(
		&model.GroupMember{Uid: "m1", Role: model.GroupRoleMember},
		&model.GroupMember{Uid: "m2", Role: model.GroupRoleMember},
	)
	called := false
	deleteGroupMember = func(ctx context.Context, gid, uid string) error { called = true; return nil }

	if err := KickMember(context.Background(), "g1", "m1", "m2"); err == nil {
		t.Fatal("expected error: member cannot kick")
	}
	if called {
		t.Error("delete must not be called")
	}
}

func TestKickOwnerRejected(t *testing.T) {
	origM := findGroupMembers
	defer func() { findGroupMembers = origM }()
	findGroupMembers = membersFn(
		&model.GroupMember{Uid: "admin", Role: model.GroupRoleAdmin},
		&model.GroupMember{Uid: "owner", Role: model.GroupRoleOwner},
	)
	if err := KickMember(context.Background(), "g1", "admin", "owner"); err == nil {
		t.Fatal("expected error kicking owner")
	}
}

func TestTransferGroupOwner(t *testing.T) {
	origG, origM, origO, origR := findGroup, findGroupMembers, updateGroupOwner, updateGroupMemberRole
	defer func() {
		findGroup, findGroupMembers, updateGroupOwner, updateGroupMemberRole = origG, origM, origO, origR
		SetNotifier(nil)
	}()

	findGroup = func(ctx context.Context, id string) (*model.GroupInfo, error) {
		return &model.GroupInfo{GroupId: id, OwnerUid: "owner"}, nil
	}
	findGroupMembers = membersFn(
		&model.GroupMember{Uid: "owner", Role: model.GroupRoleOwner},
		&model.GroupMember{Uid: "newowner", Role: model.GroupRoleMember},
	)
	var newOwner string
	updateGroupOwner = func(ctx context.Context, id, uid string) error { newOwner = uid; return nil }
	roles := map[string]byte{}
	updateGroupMemberRole = func(ctx context.Context, gid, uid string, r byte) error { roles[uid] = r; return nil }
	SetNotifier(func(uid string, _ []byte) {})

	if err := TransferGroupOwner(context.Background(), "g1", "owner", "newowner"); err != nil {
		t.Fatal(err)
	}
	if newOwner != "newowner" {
		t.Errorf("owner = %q, want newowner", newOwner)
	}
	if roles["newowner"] != model.GroupRoleOwner || roles["owner"] != model.GroupRoleMember {
		t.Errorf("roles not updated: %+v", roles)
	}
}

func TestMuteMemberSetsUntil(t *testing.T) {
	origM, origU := findGroupMembers, updateGroupMemberMute
	defer func() { findGroupMembers, updateGroupMemberMute = origM, origU; SetNotifier(nil) }()

	findGroupMembers = membersFn(
		&model.GroupMember{Uid: "owner", Role: model.GroupRoleOwner},
		&model.GroupMember{Uid: "victim", Role: model.GroupRoleMember},
	)
	var gotUntil *time.Time
	updateGroupMemberMute = func(ctx context.Context, gid, uid string, until *time.Time) error {
		gotUntil = until
		return nil
	}
	SetNotifier(func(uid string, _ []byte) {})

	if err := MuteMember(context.Background(), "g1", "owner", "victim", 10); err != nil {
		t.Fatal(err)
	}
	if gotUntil == nil || !gotUntil.After(time.Now()) {
		t.Errorf("mute until should be in the future: %v", gotUntil)
	}

	gotUntil = &time.Time{}
	if err := MuteMember(context.Background(), "g1", "owner", "victim", 0); err != nil {
		t.Fatal(err)
	}
	if gotUntil != nil {
		t.Errorf("unmute should set nil until, got %v", gotUntil)
	}
}

func TestSetGroupAnnouncementPermission(t *testing.T) {
	origM, origA := findGroupMembers, updateGroupAnnouncement
	defer func() { findGroupMembers, updateGroupAnnouncement = origM, origA; SetNotifier(nil) }()

	findGroupMembers = membersFn(&model.GroupMember{Uid: "m1", Role: model.GroupRoleMember})
	called := false
	updateGroupAnnouncement = func(ctx context.Context, gid, text string) error { called = true; return nil }

	if err := SetGroupAnnouncement(context.Background(), "g1", "m1", "hello"); err == nil {
		t.Fatal("expected error: member cannot set announcement")
	}
	if called {
		t.Error("announcement must not be updated by member")
	}

	findGroupMembers = membersFn(&model.GroupMember{Uid: "owner", Role: model.GroupRoleOwner})
	SetNotifier(func(uid string, _ []byte) {})
	if err := SetGroupAnnouncement(context.Background(), "g1", "owner", "hello"); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("announcement should be updated by owner")
	}
}

func TestLeaveGroupOwnerRejected(t *testing.T) {
	origG, origD := findGroup, deleteGroupMember
	defer func() { findGroup, deleteGroupMember = origG, origD }()

	findGroup = func(ctx context.Context, id string) (*model.GroupInfo, error) {
		return &model.GroupInfo{GroupId: id, OwnerUid: "owner"}, nil
	}
	called := false
	deleteGroupMember = func(ctx context.Context, gid, uid string) error { called = true; return nil }

	if err := LeaveGroup(context.Background(), "g1", "owner"); err == nil {
		t.Fatal("expected error: owner cannot leave")
	}
	if called {
		t.Error("owner must not be deleted on leave")
	}
}

func TestIsMuted(t *testing.T) {
	future := time.Now().Add(time.Minute)
	past := time.Now().Add(-time.Minute)
	if !IsMuted(&model.GroupMember{MuteUntil: &future}) {
		t.Error("should be muted when until is in the future")
	}
	if IsMuted(&model.GroupMember{MuteUntil: &past}) {
		t.Error("should not be muted when until is in the past")
	}
	if IsMuted(&model.GroupMember{}) {
		t.Error("nil mute_until should not be muted")
	}
}
