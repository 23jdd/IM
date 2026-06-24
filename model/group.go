package model

import "time"

// 群成员角色与群状态常量
const (
	GroupRoleMember byte = 0 // 普通成员
	GroupRoleAdmin  byte = 1 // 管理员
	GroupRoleOwner  byte = 2 // 群主

	GroupStatusNormal    byte = 0 // 正常
	GroupStatusDisbanded byte = 1 // 已解散
)

// GroupInfo 表示群的基本信息。
type GroupInfo struct {
	GroupId      string    `db:"group_id" json:"group_id"`
	Name         string    `db:"name" json:"name"`
	Avatar       string    `db:"avatar" json:"avatar,omitempty"`
	OwnerUid     string    `db:"owner_uid" json:"owner_uid"`
	Description  string    `db:"description" json:"description,omitempty"`
	Announcement string    `db:"announcement" json:"announcement,omitempty"`
	MemberCount  uint      `db:"member_count" json:"member_count"`
	Status       byte      `db:"status" json:"status"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

// GroupMember 表示一条群成员记录。
type GroupMember struct {
	Id        uint64     `db:"id" json:"id"`
	GroupId   string     `db:"group_id" json:"group_id"`
	Uid       string     `db:"uid" json:"uid"`
	Role      byte       `db:"role" json:"role"`
	Nickname  string     `db:"nickname" json:"nickname,omitempty"`
	MuteUntil *time.Time `db:"mute_until" json:"mute_until,omitempty"`
	JoinedAt  time.Time  `db:"joined_at" json:"joined_at"`
}

// GroupBrief 是"我的群聊"列表项。
type GroupBrief struct {
	GroupId string `db:"group_id" json:"group_id"`
	Name    string `db:"name" json:"name"`
}

// GroupJoinRequest 是入群申请（待群主审批）。
type GroupJoinRequest struct {
	Id        uint64    `db:"id" json:"id"`
	GroupId   string    `db:"group_id" json:"group_id"`
	Uid       string    `db:"uid" json:"uid"`
	Status    byte      `db:"status" json:"status"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
