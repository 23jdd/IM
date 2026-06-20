package service

import (
	"IM/model"
	"IM/mysql"
	"IM/utils"
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"
)

// 通过函数变量注入，便于群组逻辑的单元测试（不依赖真实 DB）。
var (
	insertGroup            = mysql.InsertGroup
	insertGroupMember      = mysql.InsertGroupMember
	findUserGroupsWithInfo = mysql.FindUserGroupsWithInfo
	findGroupMembers       = mysql.FindGroupMembers
)

func CreateGroup(ctx context.Context, ownerUid, name, description string) (*model.GroupInfo, error) {
	groupId := strconv.FormatUint(utils.GenerateId(), 10)
	now := time.Now()

	g := &model.GroupInfo{
		GroupId:     groupId,
		Name:        name,
		OwnerUid:    ownerUid,
		Description: description,
		MemberCount: 1,
		Status:      model.GroupStatusNormal,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := insertGroup(ctx, g); err != nil {
		return nil, fmt.Errorf("insert group: %w", err)
	}

	m := &model.GroupMember{
		GroupId:  groupId,
		Uid:      ownerUid,
		Role:     model.GroupRoleOwner,
		JoinedAt: now,
	}
	if err := insertGroupMember(ctx, m); err != nil {
		return nil, fmt.Errorf("insert member: %w", err)
	}

	return g, nil
}

func JoinGroup(ctx context.Context, groupId, uid string) error {
	member := &model.GroupMember{
		GroupId:  groupId,
		Uid:      uid,
		Role:     model.GroupRoleMember,
		JoinedAt: time.Now(),
	}
	return mysql.InsertGroupMember(ctx, member)
}

func LeaveGroup(ctx context.Context, groupId, uid string) error {
	g, err := mysql.FindGroup(ctx, groupId)
	if err != nil {
		return err
	}
	if g.OwnerUid == uid {
		return errors.New("owner cannot leave group, disband it instead")
	}
	return mysql.DeleteGroupMember(ctx, groupId, uid)
}

func GetGroup(ctx context.Context, groupId string) (*model.GroupInfo, error) {
	return mysql.FindGroup(ctx, groupId)
}

func GetGroupMembers(ctx context.Context, groupId string) ([]*model.GroupMember, error) {
	return mysql.FindGroupMembers(ctx, groupId)
}

func GetUserGroups(ctx context.Context, uid string) ([]*model.GroupMember, error) {
	return mysql.FindUserGroups(ctx, uid)
}

// GetUserGroupList 返回用户加入的群（含群名），供"我的群聊"列表展示。
func GetUserGroupList(ctx context.Context, uid string) ([]*model.GroupBrief, error) {
	return findUserGroupsWithInfo(ctx, uid)
}

// InviteToGroup 由群成员 inviterUid 邀请 targetUid 入群，入群后实时通知对方。
func InviteToGroup(ctx context.Context, groupId, inviterUid, targetUid string) error {
	members, err := findGroupMembers(ctx, groupId)
	if err != nil {
		return fmt.Errorf("find members: %w", err)
	}
	isMember := false
	for _, m := range members {
		if m.Uid == targetUid {
			return errors.New("对方已在群中")
		}
		if m.Uid == inviterUid {
			isMember = true
		}
	}
	if !isMember {
		return errors.New("只有群成员才能邀请")
	}

	if err := insertGroupMember(ctx, &model.GroupMember{
		GroupId:  groupId,
		Uid:      targetUid,
		Role:     model.GroupRoleMember,
		JoinedAt: time.Now(),
	}); err != nil {
		return fmt.Errorf("add member: %w", err)
	}

	notify(targetUid, "group_invite", map[string]any{"group_id": groupId, "from_uid": inviterUid})
	return nil
}
