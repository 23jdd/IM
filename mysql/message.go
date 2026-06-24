package mysql

import (
	"IM/model"
	"context"
	"fmt"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// msgConn 消息库主连接，同时被好友、群组、用户等模块复用。
var msgConn sqlx.SqlConn

// InitMessageConn 初始化消息库数据库连接。
func InitMessageConn(dataSource string) {
	msgConn = sqlx.MustNewConn(sqlx.SqlConf{
		DataSource: dataSource,
		DriverName: "mysql",
	})
}

// InsertChatMessage 插入一条聊天消息记录。
func InsertChatMessage(ctx context.Context, msg *model.ChatMessage) error {
	query := `INSERT INTO chat_message (msg_id, from_uid, to_uid, group_id, msg_type, content, status, created_at)
	           VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := msgConn.ExecCtx(ctx, query,
		msg.MsgId, msg.FromUid, msg.ToUid, msg.GroupId,
		msg.MsgType, msg.Content, msg.Status, msg.CreatedAt,
	)
	return err
}

// FindOfflineMessages 查询发给 uid 的离线消息（status=0 未读），最多 200 条按时间正序。
func FindOfflineMessages(ctx context.Context, uid string) ([]*model.ChatMessage, error) {
	query := `SELECT msg_id, from_uid, to_uid, group_id, msg_type, content, status, created_at
	           FROM chat_message
	           WHERE to_uid = ? AND status = 0
	           ORDER BY created_at ASC LIMIT 200`
	var msgs []*model.ChatMessage
	err := msgConn.QueryRowsCtx(ctx, &msgs, query, uid)
	if err != nil {
		return nil, err
	}
	return msgs, nil
}

// MarkMessagesRead 将指定 msgIds 批量标记为已读（status=1）。
func MarkMessagesRead(ctx context.Context, msgIds []string) error {
	if len(msgIds) == 0 {
		return nil
	}
	// 根据 id 数量动态拼接 IN 占位符（去掉末尾多余的逗号）
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(msgIds)), ",")
	query := fmt.Sprintf("UPDATE chat_message SET status = 1 WHERE msg_id IN (%s)", placeholders)
	// 将 msgIds 转换为 any 切片以作为可变参数传入
	args := make([]any, len(msgIds))
	for i, id := range msgIds {
		args[i] = id
	}
	_, err := msgConn.ExecCtx(ctx, query, args...)
	return err
}

// FindRecentMessages 返回与 uid 相关的最近单聊消息（按时间倒序），供会话列表聚合。
func FindRecentMessages(ctx context.Context, uid string, limit int) ([]*model.ChatMessage, error) {
	query := `SELECT msg_id, from_uid, to_uid, group_id, msg_type, content, status, created_at
	           FROM chat_message
	           WHERE from_uid = ? OR to_uid = ?
	           ORDER BY created_at DESC
	           LIMIT ?`
	var msgs []*model.ChatMessage
	err := msgConn.QueryRowsCtx(ctx, &msgs, query, uid, uid, limit)
	if err != nil {
		return nil, err
	}
	return msgs, nil
}

// FindMessageById 按 msg_id 查询单条消息。
func FindMessageById(ctx context.Context, msgId string) (*model.ChatMessage, error) {
	query := `SELECT msg_id, from_uid, to_uid, group_id, msg_type, content, status, created_at
	           FROM chat_message WHERE msg_id = ?`
	var m model.ChatMessage
	if err := msgConn.QueryRowCtx(ctx, &m, query, msgId); err != nil {
		return nil, err
	}
	return &m, nil
}

// UpdateMessageStatus 更新单条消息状态（如撤回 = 2）。
func UpdateMessageStatus(ctx context.Context, msgId string, status byte) error {
	query := `UPDATE chat_message SET status = ? WHERE msg_id = ?`
	_, err := msgConn.ExecCtx(ctx, query, status, msgId)
	return err
}
