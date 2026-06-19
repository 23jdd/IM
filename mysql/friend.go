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
