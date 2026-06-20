package service

import (
	"IM/mongdb"
	"IM/mysql"
	"context"
	"fmt"
)

// 通过函数变量注入，便于头像逻辑的单元测试（不依赖真实 Mongo/MySQL）。
var (
	saveImage        = mongdb.SaveImage
	getImage         = mongdb.GetImage
	updateUserAvatar = mysql.UpdateUserAvatar
	findUserAvatar   = mysql.FindUserAvatar
)

// UploadAvatar 将图片存入 MongoDB，并把返回的 _id 写入 MySQL.user.avatar。
func UploadAvatar(ctx context.Context, uid string, data []byte, contentType string) (string, error) {
	id, err := saveImage(ctx, data, contentType)
	if err != nil {
		return "", fmt.Errorf("save image: %w", err)
	}
	if err := updateUserAvatar(ctx, uid, id); err != nil {
		return "", fmt.Errorf("update avatar: %w", err)
	}
	return id, nil
}

// StoreImage 存储一张通用图片（如朋友圈配图），返回其 _id。
func StoreImage(ctx context.Context, data []byte, contentType string) (string, error) {
	return saveImage(ctx, data, contentType)
}

// GetAvatar 按 _id 读取头像图片（二进制 + content-type）。
func GetAvatar(ctx context.Context, id string) ([]byte, string, error) {
	return getImage(ctx, id)
}

// GetAvatarByUid 按用户 uid 解析其头像图片。任何失败（无头像/用户不存在/图片缺失）
// 都返回空数据而非错误，便于前端退化为首字母占位。
func GetAvatarByUid(ctx context.Context, uid string) ([]byte, string, error) {
	avatarId, err := findUserAvatar(ctx, uid)
	if err != nil || avatarId == "" {
		return nil, "", nil
	}
	data, ct, err := getImage(ctx, avatarId)
	if err != nil {
		return nil, "", nil
	}
	return data, ct, nil
}
