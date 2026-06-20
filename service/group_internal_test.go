package service

import (
	"IM/model"
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestCreateGroupInsertsGroupAndOwnerMember(t *testing.T) {
	origG := insertGroup
	origM := insertGroupMember
	defer func() { insertGroup = origG; insertGroupMember = origM }()

	var savedGroup *model.GroupInfo
	var savedMember *model.GroupMember
	insertGroup = func(ctx context.Context, g *model.GroupInfo) error { savedGroup = g; return nil }
	insertGroupMember = func(ctx context.Context, m *model.GroupMember) error { savedMember = m; return nil }

	g, err := CreateGroup(context.Background(), "owner1", "My Group", "desc")
	if err != nil {
		t.Fatal(err)
	}
	if g == nil || g.GroupId == "" || g.Name != "My Group" || g.OwnerUid != "owner1" {
		t.Errorf("unexpected group: %+v", g)
	}
	if savedGroup == nil || savedGroup.GroupId != g.GroupId {
		t.Fatal("insertGroup not called correctly")
	}
	if savedMember == nil || savedMember.Uid != "owner1" ||
		savedMember.Role != model.GroupRoleOwner || savedMember.GroupId != g.GroupId {
		t.Errorf("owner member not inserted correctly: %+v", savedMember)
	}
}

func TestCreateGroupInsertGroupErrorStops(t *testing.T) {
	origG := insertGroup
	origM := insertGroupMember
	defer func() { insertGroup = origG; insertGroupMember = origM }()

	insertGroup = func(ctx context.Context, g *model.GroupInfo) error { return errors.New("db down") }
	memberCalled := false
	insertGroupMember = func(ctx context.Context, m *model.GroupMember) error {
		memberCalled = true
		return nil
	}

	if _, err := CreateGroup(context.Background(), "o", "n", ""); err == nil {
		t.Fatal("expected error when group insert fails")
	}
	if memberCalled {
		t.Error("member insert must not run after group insert fails")
	}
}

func TestGetUserGroupList(t *testing.T) {
	orig := findUserGroupsWithInfo
	defer func() { findUserGroupsWithInfo = orig }()

	findUserGroupsWithInfo = func(ctx context.Context, uid string) ([]*model.GroupBrief, error) {
		return []*model.GroupBrief{
			{GroupId: "g1", Name: "Group One"},
			{GroupId: "g2", Name: "Group Two"},
		}, nil
	}

	list, err := GetUserGroupList(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].GroupId != "g1" || list[0].Name != "Group One" {
		t.Errorf("unexpected group list: %+v", list)
	}
}

func TestInviteToGroupByMember(t *testing.T) {
	origFind := findGroupMembers
	origIns := insertGroupMember
	defer func() { findGroupMembers = origFind; insertGroupMember = origIns; SetNotifier(nil) }()

	findGroupMembers = func(ctx context.Context, groupId string) ([]*model.GroupMember, error) {
		return []*model.GroupMember{{Uid: "inviter"}}, nil
	}
	var inserted *model.GroupMember
	insertGroupMember = func(ctx context.Context, m *model.GroupMember) error {
		inserted = m
		return nil
	}
	var notifyUid string
	var payload map[string]any
	SetNotifier(func(toUid string, p []byte) {
		notifyUid = toUid
		_ = json.Unmarshal(p, &payload)
	})

	if err := InviteToGroup(context.Background(), "g1", "inviter", "t"); err != nil {
		t.Fatal(err)
	}
	if inserted == nil || inserted.GroupId != "g1" || inserted.Uid != "t" ||
		inserted.Role != model.GroupRoleMember {
		t.Errorf("inserted member wrong: %+v", inserted)
	}
	if notifyUid != "t" {
		t.Errorf("notify target = %s, want t", notifyUid)
	}
	if payload["event"] != "group_invite" || payload["group_id"] != "g1" || payload["from_uid"] != "inviter" {
		t.Errorf("unexpected notify payload: %+v", payload)
	}
}

func TestInviteToGroupNonMemberRejected(t *testing.T) {
	origFind := findGroupMembers
	origIns := insertGroupMember
	defer func() { findGroupMembers = origFind; insertGroupMember = origIns }()

	findGroupMembers = func(ctx context.Context, groupId string) ([]*model.GroupMember, error) {
		return []*model.GroupMember{{Uid: "other"}}, nil
	}
	insCalled := false
	insertGroupMember = func(ctx context.Context, m *model.GroupMember) error {
		insCalled = true
		return nil
	}

	if err := InviteToGroup(context.Background(), "g1", "inviter", "t"); err == nil {
		t.Fatal("expected error when inviter is not a member")
	}
	if insCalled {
		t.Error("insertGroupMember must not be called when inviter is not a member")
	}
}

func TestInviteToGroupAlreadyMember(t *testing.T) {
	origFind := findGroupMembers
	origIns := insertGroupMember
	defer func() { findGroupMembers = origFind; insertGroupMember = origIns }()

	findGroupMembers = func(ctx context.Context, groupId string) ([]*model.GroupMember, error) {
		return []*model.GroupMember{{Uid: "inviter"}, {Uid: "t"}}, nil
	}
	insCalled := false
	insertGroupMember = func(ctx context.Context, m *model.GroupMember) error {
		insCalled = true
		return nil
	}

	if err := InviteToGroup(context.Background(), "g1", "inviter", "t"); err == nil {
		t.Fatal("expected error when target already in group")
	}
	if insCalled {
		t.Error("insertGroupMember must not be called when target already a member")
	}
}
