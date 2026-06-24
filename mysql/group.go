package mysql

import (
	"IM/model"
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// groupConn 群组相关表使用的数据库连接（复用消息库连接）。
var groupConn sqlx.SqlConn

// InitGroupConn 初始化群组模块的数据库连接，直接复用 msgConn。
func InitGroupConn(dataSource string) {
	groupConn = msgConn
}

// InsertGroup 插入一条群信息记录（创建群时调用）。
func InsertGroup(ctx context.Context, g *model.GroupInfo) error {
	query := `INSERT INTO group_info (group_id, name, avatar, owner_uid, description, announcement, member_count, status, created_at, updated_at)
	           VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := groupConn.ExecCtx(ctx, query,
		g.GroupId, g.Name, g.Avatar, g.OwnerUid, g.Description, g.Announcement,
		g.MemberCount, g.Status, g.CreatedAt, g.UpdatedAt,
	)
	return err
}

// FindGroup 按 groupId 查询群信息。
func FindGroup(ctx context.Context, groupId string) (*model.GroupInfo, error) {
	query := `SELECT group_id, name, avatar, owner_uid, description, announcement, member_count, status, created_at, updated_at
	           FROM group_info WHERE group_id = ?`
	var g model.GroupInfo
	err := groupConn.QueryRowCtx(ctx, &g, query, groupId)
	if err != nil {
		return nil, err
	}
	return &g, nil
}

// InsertGroupMember 插入一条群成员记录（成员入群时调用）。
func InsertGroupMember(ctx context.Context, m *model.GroupMember) error {
	query := `INSERT INTO group_member (group_id, uid, role, nickname, joined_at)
	           VALUES (?, ?, ?, ?, ?)`
	_, err := groupConn.ExecCtx(ctx, query, m.GroupId, m.Uid, m.Role, m.Nickname, m.JoinedAt)
	return err
}

// FindGroupMembers 查询某群的全部成员。
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

// FindUserGroups 查询某用户加入的全部群成员记录。
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

// DeleteGroupMember 删除群成员（退群或被踢时调用）。
func DeleteGroupMember(ctx context.Context, groupId, uid string) error {
	query := `DELETE FROM group_member WHERE group_id = ? AND uid = ?`
	_, err := groupConn.ExecCtx(ctx, query, groupId, uid)
	return err
}

// UpdateGroupMemberRole 更新群成员的角色（如设置/取消管理员）。
func UpdateGroupMemberRole(ctx context.Context, groupId, uid string, role byte) error {
	query := `UPDATE group_member SET role = ? WHERE group_id = ? AND uid = ?`
	_, err := groupConn.ExecCtx(ctx, query, role, groupId, uid)
	return err
}

// UpdateGroupStatus 更新群状态（如解散 = 1）。
func UpdateGroupStatus(ctx context.Context, groupId string, status byte) error {
	query := `UPDATE group_info SET status = ? WHERE group_id = ?`
	_, err := groupConn.ExecCtx(ctx, query, status, groupId)
	return err
}

// UpdateGroupOwner 更新群主（群主转让时调用）。
func UpdateGroupOwner(ctx context.Context, groupId, newOwnerUid string) error {
	query := `UPDATE group_info SET owner_uid = ? WHERE group_id = ?`
	_, err := groupConn.ExecCtx(ctx, query, newOwnerUid, groupId)
	return err
}

// UpdateGroupMemberMute 设置成员禁言截止时间（until 为 nil 表示解除禁言）。
func UpdateGroupMemberMute(ctx context.Context, groupId, uid string, until *time.Time) error {
	query := `UPDATE group_member SET mute_until = ? WHERE group_id = ? AND uid = ?`
	_, err := groupConn.ExecCtx(ctx, query, until, groupId, uid)
	return err
}

// UpdateGroupAnnouncement 更新群公告。
func UpdateGroupAnnouncement(ctx context.Context, groupId, announcement string) error {
	query := `UPDATE group_info SET announcement = ? WHERE group_id = ?`
	_, err := groupConn.ExecCtx(ctx, query, announcement, groupId)
	return err
}

// FindUserGroupsWithInfo 返回用户加入的群（含群名）。
func FindUserGroupsWithInfo(ctx context.Context, uid string) ([]*model.GroupBrief, error) {
	query := `SELECT g.group_id, g.name
	           FROM group_member m JOIN group_info g ON m.group_id = g.group_id
	           WHERE m.uid = ? AND g.status = ?`
	var items []*model.GroupBrief
	err := groupConn.QueryRowsCtx(ctx, &items, query, uid, model.GroupStatusNormal)
	if err != nil {
		return nil, err
	}
	return items, nil
}

// InsertJoinRequest 写入一条入群申请（重复申请重置为待审批）。
func InsertJoinRequest(ctx context.Context, groupId, uid string) error {
	query := `INSERT INTO group_join_request (group_id, uid, status) VALUES (?, ?, 0)
	          ON DUPLICATE KEY UPDATE status = 0`
	_, err := groupConn.ExecCtx(ctx, query, groupId, uid)
	return err
}

// FindPendingJoinRequests 返回某群待审批的入群申请。
func FindPendingJoinRequests(ctx context.Context, groupId string) ([]*model.GroupJoinRequest, error) {
	query := `SELECT id, group_id, uid, status, created_at
	          FROM group_join_request WHERE group_id = ? AND status = 0`
	var items []*model.GroupJoinRequest
	err := groupConn.QueryRowsCtx(ctx, &items, query, groupId)
	if err != nil {
		return nil, err
	}
	return items, nil
}

// DeleteJoinRequest 删除某条入群申请（审批后调用）。
func DeleteJoinRequest(ctx context.Context, groupId, uid string) error {
	query := `DELETE FROM group_join_request WHERE group_id = ? AND uid = ?`
	_, err := groupConn.ExecCtx(ctx, query, groupId, uid)
	return err
}
