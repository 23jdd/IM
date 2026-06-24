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
	insertGroup             = mysql.InsertGroup
	insertGroupMember       = mysql.InsertGroupMember
	findUserGroupsWithInfo  = mysql.FindUserGroupsWithInfo
	findGroupMembers        = mysql.FindGroupMembers
	findGroup               = mysql.FindGroup
	insertJoinRequest       = mysql.InsertJoinRequest
	findPendingJoinRequests = mysql.FindPendingJoinRequests
	deleteJoinRequest       = mysql.DeleteJoinRequest
	deleteGroupMember       = mysql.DeleteGroupMember
	updateGroupStatus       = mysql.UpdateGroupStatus
	updateGroupOwner        = mysql.UpdateGroupOwner
	updateGroupMemberRole   = mysql.UpdateGroupMemberRole
	updateGroupMemberMute   = mysql.UpdateGroupMemberMute
	updateGroupAnnouncement = mysql.UpdateGroupAnnouncement
)

// CreateGroup 创建群：生成群信息并把创建者写入为群主成员。
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

// JoinGroup 将 uid 以普通成员身份直接加入群。
func JoinGroup(ctx context.Context, groupId, uid string) error {
	member := &model.GroupMember{
		GroupId:  groupId,
		Uid:      uid,
		Role:     model.GroupRoleMember,
		JoinedAt: time.Now(),
	}
	return mysql.InsertGroupMember(ctx, member)
}

// LeaveGroup 退群：群主不可直接退群，退群后通知群主。
func LeaveGroup(ctx context.Context, groupId, uid string) error {
	g, err := findGroup(ctx, groupId)
	if err != nil {
		return errors.New("群不存在")
	}
	if g.OwnerUid == uid {
		return errors.New("群主不能退群，请先转让群主或解散群")
	}
	if err := deleteGroupMember(ctx, groupId, uid); err != nil {
		return err
	}
	notify(g.OwnerUid, "group_member_left", map[string]any{"group_id": groupId, "from_uid": uid})
	return nil
}

// GetGroup 按群号查询群信息。
func GetGroup(ctx context.Context, groupId string) (*model.GroupInfo, error) {
	return mysql.FindGroup(ctx, groupId)
}

// GetGroupMembers 返回某群的全部成员。
func GetGroupMembers(ctx context.Context, groupId string) ([]*model.GroupMember, error) {
	return mysql.FindGroupMembers(ctx, groupId)
}

// GetUserGroups 返回 uid 的群成员记录（其加入的所有群）。
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

// RequestJoinGroup 申请加群（待群主审批）。校验未在群，写申请并通知群主。
func RequestJoinGroup(ctx context.Context, groupId, uid string) error {
	members, err := findGroupMembers(ctx, groupId)
	if err != nil {
		return fmt.Errorf("find members: %w", err)
	}
	for _, m := range members {
		if m.Uid == uid {
			return errors.New("你已在群中")
		}
	}
	g, err := findGroup(ctx, groupId)
	if err != nil {
		return errors.New("群不存在")
	}
	if err := insertJoinRequest(ctx, groupId, uid); err != nil {
		return fmt.Errorf("insert join request: %w", err)
	}
	notify(g.OwnerUid, "group_join_request", map[string]any{"group_id": groupId, "from_uid": uid})
	return nil
}

// GetGroupJoinRequests 群主查看某群的待审批入群申请。
func GetGroupJoinRequests(ctx context.Context, groupId, requesterUid string) ([]*model.GroupJoinRequest, error) {
	g, err := findGroup(ctx, groupId)
	if err != nil {
		return nil, errors.New("群不存在")
	}
	if g.OwnerUid != requesterUid {
		return nil, errors.New("只有群主可查看入群申请")
	}
	return findPendingJoinRequests(ctx, groupId)
}

// ApproveJoinRequest 群主通过入群申请：加入群成员，删除申请，通知申请人。
func ApproveJoinRequest(ctx context.Context, groupId, approverUid, applicantUid string) error {
	g, err := findGroup(ctx, groupId)
	if err != nil {
		return errors.New("群不存在")
	}
	if g.OwnerUid != approverUid {
		return errors.New("只有群主可审批")
	}
	if err := insertGroupMember(ctx, &model.GroupMember{
		GroupId:  groupId,
		Uid:      applicantUid,
		Role:     model.GroupRoleMember,
		JoinedAt: time.Now(),
	}); err != nil {
		return fmt.Errorf("add member: %w", err)
	}
	_ = deleteJoinRequest(ctx, groupId, applicantUid)
	notify(applicantUid, "group_join_approved", map[string]any{"group_id": groupId, "from_uid": approverUid})
	return nil
}

// RejectJoinRequest 群主拒绝入群申请：删除申请，通知申请人。
func RejectJoinRequest(ctx context.Context, groupId, approverUid, applicantUid string) error {
	g, err := findGroup(ctx, groupId)
	if err != nil {
		return errors.New("群不存在")
	}
	if g.OwnerUid != approverUid {
		return errors.New("只有群主可审批")
	}
	_ = deleteJoinRequest(ctx, groupId, applicantUid)
	notify(applicantUid, "group_join_rejected", map[string]any{"group_id": groupId, "from_uid": approverUid})
	return nil
}

// findMember 在成员列表中按 uid 查找成员（不存在返回 nil）。
func findMember(members []*model.GroupMember, uid string) *model.GroupMember {
	for _, m := range members {
		if m.Uid == uid {
			return m
		}
	}
	return nil
}

// IsMuted 判断成员当前是否处于禁言状态。
func IsMuted(m *model.GroupMember) bool {
	return m != nil && m.MuteUntil != nil && m.MuteUntil.After(time.Now())
}

// DisbandGroup 解散群（仅群主）：标记群状态为已解散并通知全体成员。
func DisbandGroup(ctx context.Context, groupId, operatorUid string) error {
	g, err := findGroup(ctx, groupId)
	if err != nil {
		return errors.New("群不存在")
	}
	if g.OwnerUid != operatorUid {
		return errors.New("只有群主可解散群")
	}
	members, _ := findGroupMembers(ctx, groupId)
	if err := updateGroupStatus(ctx, groupId, model.GroupStatusDisbanded); err != nil {
		return err
	}
	for _, m := range members {
		notify(m.Uid, "group_disbanded", map[string]any{"group_id": groupId})
	}
	return nil
}

// KickMember 踢出群成员（群主/管理员；管理员只能踢普通成员，且不能踢群主/自己）。
func KickMember(ctx context.Context, groupId, operatorUid, targetUid string) error {
	if operatorUid == targetUid {
		return errors.New("不能踢出自己")
	}
	members, err := findGroupMembers(ctx, groupId)
	if err != nil {
		return fmt.Errorf("find members: %w", err)
	}
	op := findMember(members, operatorUid)
	target := findMember(members, targetUid)
	if op == nil {
		return errors.New("你不在群中")
	}
	if target == nil {
		return errors.New("对方不在群中")
	}
	if op.Role != model.GroupRoleOwner && op.Role != model.GroupRoleAdmin {
		return errors.New("只有群主或管理员可踢人")
	}
	if target.Role == model.GroupRoleOwner {
		return errors.New("不能踢出群主")
	}
	if target.Role == model.GroupRoleAdmin && op.Role != model.GroupRoleOwner {
		return errors.New("只有群主可踢出管理员")
	}
	if err := deleteGroupMember(ctx, groupId, targetUid); err != nil {
		return err
	}
	notify(targetUid, "group_kicked", map[string]any{"group_id": groupId, "from_uid": operatorUid})
	for _, m := range members {
		if m.Uid != targetUid {
			notify(m.Uid, "group_member_changed", map[string]any{"group_id": groupId})
		}
	}
	return nil
}

// TransferGroupOwner 转让群主（仅群主）：将群主转给目标成员，原群主降为普通成员。
func TransferGroupOwner(ctx context.Context, groupId, operatorUid, targetUid string) error {
	if operatorUid == targetUid {
		return errors.New("不能转让给自己")
	}
	g, err := findGroup(ctx, groupId)
	if err != nil {
		return errors.New("群不存在")
	}
	if g.OwnerUid != operatorUid {
		return errors.New("只有群主可转让群主")
	}
	members, err := findGroupMembers(ctx, groupId)
	if err != nil {
		return fmt.Errorf("find members: %w", err)
	}
	if findMember(members, targetUid) == nil {
		return errors.New("对方不在群中")
	}
	if err := updateGroupOwner(ctx, groupId, targetUid); err != nil {
		return err
	}
	_ = updateGroupMemberRole(ctx, groupId, targetUid, model.GroupRoleOwner)
	_ = updateGroupMemberRole(ctx, groupId, operatorUid, model.GroupRoleMember)
	for _, m := range members {
		notify(m.Uid, "group_owner_changed", map[string]any{"group_id": groupId, "owner_uid": targetUid})
	}
	return nil
}

// MuteMember 禁言/解除禁言群成员（群主/管理员）。minutes<=0 表示解除禁言。
func MuteMember(ctx context.Context, groupId, operatorUid, targetUid string, minutes int) error {
	if operatorUid == targetUid {
		return errors.New("不能禁言自己")
	}
	members, err := findGroupMembers(ctx, groupId)
	if err != nil {
		return fmt.Errorf("find members: %w", err)
	}
	op := findMember(members, operatorUid)
	target := findMember(members, targetUid)
	if op == nil {
		return errors.New("你不在群中")
	}
	if target == nil {
		return errors.New("对方不在群中")
	}
	if op.Role != model.GroupRoleOwner && op.Role != model.GroupRoleAdmin {
		return errors.New("只有群主或管理员可禁言")
	}
	if target.Role == model.GroupRoleOwner {
		return errors.New("不能禁言群主")
	}
	if target.Role == model.GroupRoleAdmin && op.Role != model.GroupRoleOwner {
		return errors.New("只有群主可禁言管理员")
	}
	var until *time.Time
	if minutes > 0 {
		t := time.Now().Add(time.Duration(minutes) * time.Minute)
		until = &t
	}
	if err := updateGroupMemberMute(ctx, groupId, targetUid, until); err != nil {
		return err
	}
	notify(targetUid, "group_muted", map[string]any{"group_id": groupId, "minutes": minutes})
	return nil
}

// SetGroupAnnouncement 设置群公告（群主/管理员），并通知全体成员。
func SetGroupAnnouncement(ctx context.Context, groupId, operatorUid, text string) error {
	members, err := findGroupMembers(ctx, groupId)
	if err != nil {
		return fmt.Errorf("find members: %w", err)
	}
	op := findMember(members, operatorUid)
	if op == nil {
		return errors.New("你不在群中")
	}
	if op.Role != model.GroupRoleOwner && op.Role != model.GroupRoleAdmin {
		return errors.New("只有群主或管理员可发布公告")
	}
	if err := updateGroupAnnouncement(ctx, groupId, text); err != nil {
		return err
	}
	for _, m := range members {
		notify(m.Uid, "group_announcement", map[string]any{"group_id": groupId})
	}
	return nil
}
