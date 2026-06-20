package service

import (
	"IM/model"
	"context"
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
