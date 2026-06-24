package model

import "time"

// ChatMessage 表示一条聊天消息（单聊或群聊）。
type ChatMessage struct {
	MsgId     string    `db:"msg_id" json:"msg_id"`
	FromUid   string    `db:"from_uid" json:"from_uid"`
	ToUid     string    `db:"to_uid" json:"to_uid"`
	GroupId   string    `db:"group_id" json:"group_id,omitempty"`
	MsgType   byte      `db:"msg_type" json:"msg_type"`
	Content   string    `db:"content" json:"content"`
	Status    byte      `db:"status" json:"status"` // 0=未读 1=已读 2=已撤回
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// 消息状态常量
const (
	MsgStatusUnread  byte = 0 // 未读
	MsgStatusRead    byte = 1 // 已读
	MsgStatusRevoked byte = 2 // 已撤回
)
