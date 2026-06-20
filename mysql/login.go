package mysql

import (
	"IM/mysql/model"
	"context"
)

// FindUserByLogin 按多种登录标识（uid / 手机 / 邮箱 / 昵称）查找用户。
func FindUserByLogin(ctx context.Context, identifier string) (*model.User, error) {
	query := `SELECT uid, password_hash, name, avatar, gender, birthday, signature,
	                 phone, email, last_ip, status, last_seen_at, created_at, updated_at
	          FROM user
	          WHERE uid = ? OR phone = ? OR email = ? OR name = ?
	          LIMIT 1`
	var u model.User
	err := msgConn.QueryRowCtx(ctx, &u, query, identifier, identifier, identifier, identifier)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
