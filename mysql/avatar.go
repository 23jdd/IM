package mysql

import (
	"context"
	"time"
)

// UpdateUserAvatar 将用户头像字段更新为 MongoDB 图片的 _id。
func UpdateUserAvatar(ctx context.Context, uid, avatarId string) error {
	query := `UPDATE user SET avatar = ?, updated_at = ? WHERE uid = ?`
	_, err := msgConn.ExecCtx(ctx, query, avatarId, time.Now(), uid)
	return err
}

// FindUserAvatar 返回用户的头像图片 _id（无头像时为空串）。
func FindUserAvatar(ctx context.Context, uid string) (string, error) {
	var avatar string
	query := `SELECT COALESCE(avatar, '') FROM user WHERE uid = ?`
	err := msgConn.QueryRowCtx(ctx, &avatar, query, uid)
	if err != nil {
		return "", err
	}
	return avatar, nil
}
