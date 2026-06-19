package mysql

import (
	"IM/model"
	"context"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var groupConn sqlx.SqlConn

func InitGroupConn(dataSource string) {
	groupConn = msgConn
}

func InsertGroup(ctx context.Context, g *model.GroupInfo) error {
	query := `INSERT INTO group_info (group_id, name, avatar, owner_uid, description, member_count, status, created_at, updated_at)
	           VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := groupConn.ExecCtx(ctx, query,
		g.GroupId, g.Name, g.Avatar, g.OwnerUid, g.Description,
		g.MemberCount, g.Status, g.CreatedAt, g.UpdatedAt,
	)
	return err
}

func FindGroup(ctx context.Context, groupId string) (*model.GroupInfo, error) {
	query := `SELECT group_id, name, avatar, owner_uid, description, member_count, status, created_at, updated_at
	           FROM group_info WHERE group_id = ?`
	var g model.GroupInfo
	err := groupConn.QueryRowCtx(ctx, &g, query, groupId)
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func InsertGroupMember(ctx context.Context, m *model.GroupMember) error {
	query := `INSERT INTO group_member (group_id, uid, role, nickname, joined_at)
	           VALUES (?, ?, ?, ?, ?)`
	_, err := groupConn.ExecCtx(ctx, query, m.GroupId, m.Uid, m.Role, m.Nickname, m.JoinedAt)
	return err
}

func FindGroupMembers(ctx context.Context, groupId string) ([]*model.GroupMember, error) {
	query := `SELECT id, group_id, uid, role, nickname, mute_until, joined_at
	           FROM group_member WHERE group_id = ?`
	var members []*model.GroupMember
	err := groupConn.QueryRowsCtx(ctx, &members, query, groupId)
	if err != nil {
		return nil, err
	}
	return members, nil
}

func FindUserGroups(ctx context.Context, uid string) ([]*model.GroupMember, error) {
	query := `SELECT id, group_id, uid, role, nickname, mute_until, joined_at
	           FROM group_member WHERE uid = ?`
	var members []*model.GroupMember
	err := groupConn.QueryRowsCtx(ctx, &members, query, uid)
	if err != nil {
		return nil, err
	}
	return members, nil
}

func DeleteGroupMember(ctx context.Context, groupId, uid string) error {
	query := `DELETE FROM group_member WHERE group_id = ? AND uid = ?`
	_, err := groupConn.ExecCtx(ctx, query, groupId, uid)
	return err
}

func UpdateGroupMemberRole(ctx context.Context, groupId, uid string, role byte) error {
	query := `UPDATE group_member SET role = ? WHERE group_id = ? AND uid = ?`
	_, err := groupConn.ExecCtx(ctx, query, role, groupId, uid)
	return err
}
