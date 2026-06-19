package model

import "time"

const (
	GroupRoleMember byte = 0
	GroupRoleAdmin  byte = 1
	GroupRoleOwner  byte = 2

	GroupStatusNormal    byte = 0
	GroupStatusDisbanded byte = 1
)

type GroupInfo struct {
	GroupId     string    `db:"group_id" json:"group_id"`
	Name        string    `db:"name" json:"name"`
	Avatar      string    `db:"avatar" json:"avatar,omitempty"`
	OwnerUid    string    `db:"owner_uid" json:"owner_uid"`
	Description string    `db:"description" json:"description,omitempty"`
	MemberCount uint      `db:"member_count" json:"member_count"`
	Status      byte      `db:"status" json:"status"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

type GroupMember struct {
	Id        uint64     `db:"id" json:"id"`
	GroupId   string     `db:"group_id" json:"group_id"`
	Uid       string     `db:"uid" json:"uid"`
	Role      byte       `db:"role" json:"role"`
	Nickname  string     `db:"nickname" json:"nickname,omitempty"`
	MuteUntil *time.Time `db:"mute_until" json:"mute_until,omitempty"`
	JoinedAt  time.Time  `db:"joined_at" json:"joined_at"`
}
