package mysql

import (
	"IM/model"
	"context"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// friendConn 好友关系表使用的数据库连接（复用消息库连接）。
var friendConn sqlx.SqlConn

// InitFriendConn 初始化好友模块的数据库连接，直接复用 msgConn。
func InitFriendConn(dataSource string) {
	friendConn = msgConn
}

// InsertFriend 插入一条好友关系记录（如好友申请）。
func InsertFriend(ctx context.Context, f *model.FriendRelation) error {
	query := `INSERT INTO friend_relation (uid, friend_uid, status, remark, created_at)
	           VALUES (?, ?, ?, ?, ?)`
	_, err := friendConn.ExecCtx(ctx, query, f.Uid, f.FriendUid, f.Status, f.Remark, f.CreatedAt)
	return err
}

// FindFriends 查询 uid 已接受（status=accepted）的好友关系列表。
func FindFriends(ctx context.Context, uid string) ([]*model.FriendRelation, error) {
	query := `SELECT id, uid, friend_uid, status, remark, created_at, updated_at
	           FROM friend_relation
	           WHERE uid = ? AND status = ?`
	var friends []*model.FriendRelation
	err := friendConn.QueryRowsCtx(ctx, &friends, query, uid, model.FriendStatusAccepted)
	if err != nil {
		return nil, err
	}
	return friends, nil
}

// FindFriendRequests 查询发给 uid 的、待处理（status=pending）的好友申请。
func FindFriendRequests(ctx context.Context, uid string) ([]*model.FriendRelation, error) {
	query := `SELECT id, uid, friend_uid, status, remark, created_at, updated_at
	           FROM friend_relation
	           WHERE friend_uid = ? AND status = ?`
	var requests []*model.FriendRelation
	err := friendConn.QueryRowsCtx(ctx, &requests, query, uid, model.FriendStatusPending)
	if err != nil {
		return nil, err
	}
	return requests, nil
}

// UpdateFriendStatus 更新 uid 与 friendUid 之间好友关系的状态。
func UpdateFriendStatus(ctx context.Context, uid, friendUid string, status byte) error {
	query := `UPDATE friend_relation SET status = ? WHERE uid = ? AND friend_uid = ?`
	_, err := friendConn.ExecCtx(ctx, query, status, uid, friendUid)
	return err
}

// DeleteFriend 删除 uid 到 friendUid 的好友关系记录。
func DeleteFriend(ctx context.Context, uid, friendUid string) error {
	query := `DELETE FROM friend_relation WHERE uid = ? AND friend_uid = ?`
	_, err := friendConn.ExecCtx(ctx, query, uid, friendUid)
	return err
}

// UpdateFriendRemark 修改 uid 对 friendUid 的好友备注。
func UpdateFriendRemark(ctx context.Context, uid, friendUid, remark string) error {
	query := `UPDATE friend_relation SET remark = ? WHERE uid = ? AND friend_uid = ?`
	_, err := friendConn.ExecCtx(ctx, query, remark, uid, friendUid)
	return err
}

// FindFriendList 返回已接受好友的展示信息（join user 表）。
func FindFriendList(ctx context.Context, uid string) ([]*model.FriendInfo, error) {
	query := `SELECT f.friend_uid, COALESCE(f.remark,'') AS remark,
	                 COALESCE(u.name,'') AS name, COALESCE(u.avatar,'') AS avatar
	          FROM friend_relation f
	          LEFT JOIN user u ON f.friend_uid = u.uid
	          WHERE f.uid = ? AND f.status = ?`
	var items []*model.FriendInfo
	err := friendConn.QueryRowsCtx(ctx, &items, query, uid, model.FriendStatusAccepted)
	if err != nil {
		return nil, err
	}
	return items, nil
}

// UpsertBlockedFriend 将 uid->friendUid 置为已拉黑；若关系不存在则插入（可拉黑非好友）。
func UpsertBlockedFriend(ctx context.Context, uid, friendUid string) error {
	query := `INSERT INTO friend_relation (uid, friend_uid, status) VALUES (?, ?, ?)
	          ON DUPLICATE KEY UPDATE status = VALUES(status)`
	_, err := friendConn.ExecCtx(ctx, query, uid, friendUid, model.FriendStatusBlocked)
	return err
}

// FindBlockedList 返回 uid 拉黑的用户展示信息（join user 表）。
func FindBlockedList(ctx context.Context, uid string) ([]*model.FriendInfo, error) {
	query := `SELECT f.friend_uid, COALESCE(f.remark,'') AS remark,
	                 COALESCE(u.name,'') AS name, COALESCE(u.avatar,'') AS avatar
	          FROM friend_relation f
	          LEFT JOIN user u ON f.friend_uid = u.uid
	          WHERE f.uid = ? AND f.status = ?`
	var items []*model.FriendInfo
	err := friendConn.QueryRowsCtx(ctx, &items, query, uid, model.FriendStatusBlocked)
	if err != nil {
		return nil, err
	}
	return items, nil
}

// IsBlockedBetween 判断 a、b 之间是否存在任一方向的拉黑关系。
func IsBlockedBetween(ctx context.Context, a, b string) (bool, error) {
	query := `SELECT COUNT(*) FROM friend_relation
	          WHERE status = ? AND ((uid = ? AND friend_uid = ?) OR (uid = ? AND friend_uid = ?))`
	var n int
	if err := friendConn.QueryRowCtx(ctx, &n, query, model.FriendStatusBlocked, a, b, b, a); err != nil {
		return false, err
	}
	return n > 0, nil
}
