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

	if err := mysql.InsertGroup(ctx, g); err != nil {
		return nil, fmt.Errorf("insert group: %w", err)
	}

	m := &model.GroupMember{
		GroupId:  groupId,
		Uid:      ownerUid,
		Role:     model.GroupRoleOwner,
		JoinedAt: now,
	}
	if err := mysql.InsertGroupMember(ctx, m); err != nil {
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
