package mysql

import (
	"IM/model"
	"context"
	"fmt"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var msgConn sqlx.SqlConn

func InitMessageConn(dataSource string) {
	msgConn = sqlx.MustNewConn(sqlx.SqlConf{
		DataSource: dataSource,
		DriverName: "mysql",
	})
}

func InsertChatMessage(ctx context.Context, msg *model.ChatMessage) error {
	query := `INSERT INTO chat_message (msg_id, from_uid, to_uid, group_id, msg_type, content, status, created_at)
	           VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := msgConn.ExecCtx(ctx, query,
		msg.MsgId, msg.FromUid, msg.ToUid, msg.GroupId,
		msg.MsgType, msg.Content, msg.Status, msg.CreatedAt,
	)
	return err
}

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

func MarkMessagesRead(ctx context.Context, msgIds []string) error {
	if len(msgIds) == 0 {
		return nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(msgIds)), ",")
	query := fmt.Sprintf("UPDATE chat_message SET status = 1 WHERE msg_id IN (%s)", placeholders)
	args := make([]any, len(msgIds))
	for i, id := range msgIds {
		args[i] = id
	}
	_, err := msgConn.ExecCtx(ctx, query, args...)
	return err
}
