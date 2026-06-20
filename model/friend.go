package model

import "time"

const (
	FriendStatusPending  byte = 0
	FriendStatusAccepted byte = 1
	FriendStatusBlocked  byte = 2
)

type FriendRelation struct {
	Id         uint64    `db:"id" json:"id"`
	Uid        string    `db:"uid" json:"uid"`
	FriendUid  string    `db:"friend_uid" json:"friend_uid"`
	Status     byte      `db:"status" json:"status"`
	Remark     string    `db:"remark" json:"remark,omitempty"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
}

// FriendInfo 是好友列表项（join user 表后的展示信息）。
type FriendInfo struct {
	Uid    string `db:"friend_uid" json:"uid"`
	Remark string `db:"remark" json:"remark"`
	Name   string `db:"name" json:"name"`
	Avatar string `db:"avatar" json:"avatar"`
}
