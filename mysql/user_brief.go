package mysql

import (
	"IM/model"
	"context"
)

// FindUserBrief 按 uid 查询用户的公开基本信息（uid/name/avatar/gender/signature）。
func FindUserBrief(ctx context.Context, uid string) (*model.UserBrief, error) {
	var u model.UserBrief
	query := `SELECT uid, name, COALESCE(avatar, '') AS avatar, gender, COALESCE(signature, '') AS signature FROM user WHERE uid = ?`
	if err := msgConn.QueryRowCtx(ctx, &u, query, uid); err != nil {
		return nil, err
	}
	return &u, nil
}
