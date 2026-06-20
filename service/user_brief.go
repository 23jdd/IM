package service

import (
	"IM/model"
	"IM/mysql"
	"context"
)

// findUserBrief 便于测试注入。
var findUserBrief = mysql.FindUserBrief

// GetUserBrief 按 uid 查询用户公开信息（搜索添加好友前预览）。
func GetUserBrief(ctx context.Context, uid string) (*model.UserBrief, error) {
	return findUserBrief(ctx, uid)
}
