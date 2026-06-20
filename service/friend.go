package service

import (
	"IM/model"
	"IM/mysql"
	"context"
	"errors"
	"fmt"
	"time"
)

func SendFriendRequest(ctx context.Context, uid, friendUid, remark string) error {
	if uid == friendUid {
		return errors.New("cannot add yourself")
	}

	now := time.Now()
	err := mysql.InsertFriend(ctx, &model.FriendRelation{
		Uid:       uid,
		FriendUid: friendUid,
		Status:    model.FriendStatusPending,
		Remark:    remark,
		CreatedAt: now,
	})
	if err != nil {
		return fmt.Errorf("insert friend: %w", err)
	}

	_ = mysql.InsertFriend(ctx, &model.FriendRelation{
		Uid:       friendUid,
		FriendUid: uid,
		Status:    model.FriendStatusPending,
		CreatedAt: now,
	})

	return nil
}

func AcceptFriendRequest(ctx context.Context, uid, friendUid string) error {
	if err := mysql.UpdateFriendStatus(ctx, uid, friendUid, model.FriendStatusAccepted); err != nil {
		return fmt.Errorf("accept friend: %w", err)
	}
	_ = mysql.UpdateFriendStatus(ctx, friendUid, uid, model.FriendStatusAccepted)
	return nil
}

func BlockFriend(ctx context.Context, uid, friendUid string) error {
	if err := mysql.UpdateFriendStatus(ctx, uid, friendUid, model.FriendStatusBlocked); err != nil {
		return fmt.Errorf("block friend: %w", err)
	}
	return nil
}

func RemoveFriend(ctx context.Context, uid, friendUid string) error {
	if err := mysql.DeleteFriend(ctx, uid, friendUid); err != nil {
		return fmt.Errorf("remove friend: %w", err)
	}
	_ = mysql.DeleteFriend(ctx, friendUid, uid)
	return nil
}

func GetFriends(ctx context.Context, uid string) ([]*model.FriendRelation, error) {
	return mysql.FindFriends(ctx, uid)
}

func GetFriendRequests(ctx context.Context, uid string) ([]*model.FriendRelation, error) {
	return mysql.FindFriendRequests(ctx, uid)
}

// findFriendList 便于测试注入。
var findFriendList = mysql.FindFriendList

// GetFriendList 返回好友列表展示信息（含昵称/头像/备注）。
func GetFriendList(ctx context.Context, uid string) ([]*model.FriendInfo, error) {
	return findFriendList(ctx, uid)
}
