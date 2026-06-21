package mysql

import (
	"IM/model"
	"context"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var friendConn sqlx.SqlConn

func InitFriendConn(dataSource string) {
	friendConn = msgConn
}

func InsertFriend(ctx context.Context, f *model.FriendRelation) error {
	query := `INSERT INTO friend_relation (uid, friend_uid, status, remark, created_at)
	           VALUES (?, ?, ?, ?, ?)`
	_, err := friendConn.ExecCtx(ctx, query, f.Uid, f.FriendUid, f.Status, f.Remark, f.CreatedAt)
	return err
}

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

func UpdateFriendStatus(ctx context.Context, uid, friendUid string, status byte) error {
	query := `UPDATE friend_relation SET status = ? WHERE uid = ? AND friend_uid = ?`
	_, err := friendConn.ExecCtx(ctx, query, status, uid, friendUid)
	return err
}

func DeleteFriend(ctx context.Context, uid, friendUid string) error {
	query := `DELETE FROM friend_relation WHERE uid = ? AND friend_uid = ?`
	_, err := friendConn.ExecCtx(ctx, query, uid, friendUid)
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
