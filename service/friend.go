package service

import (
	"IM/model"
	"IM/mysql"
	"context"
	"errors"
	"fmt"
	"time"
)

// 通过函数变量注入，便于好友逻辑的单元测试（不依赖真实 DB）。
var (
	insertFriend       = mysql.InsertFriend
	updateFriendStatus = mysql.UpdateFriendStatus
	deleteFriendRel    = mysql.DeleteFriend
	findFriendList     = mysql.FindFriendList
)

// SendFriendRequest requester 向 target 发起好友申请（单向 pending，避免申请人自己也看到申请）。
func SendFriendRequest(ctx context.Context, requester, target, remark string) error {
	if requester == target {
		return errors.New("cannot add yourself")
	}
	if err := insertFriend(ctx, &model.FriendRelation{
		Uid:       requester,
		FriendUid: target,
		Status:    model.FriendStatusPending,
		Remark:    remark,
		CreatedAt: time.Now(),
	}); err != nil {
		return fmt.Errorf("insert friend request: %w", err)
	}
	notify(target, "friend_request", map[string]any{"from_uid": requester, "remark": remark})
	return nil
}

// AcceptFriendRequest accepter 接受 requester 的申请：把 requester→accepter 置为 accepted，
// 并建立 accepter→requester 的 accepted 关系（双向成为好友）。
func AcceptFriendRequest(ctx context.Context, accepter, requester string) error {
	if err := updateFriendStatus(ctx, requester, accepter, model.FriendStatusAccepted); err != nil {
		return fmt.Errorf("accept friend: %w", err)
	}
	_ = insertFriend(ctx, &model.FriendRelation{
		Uid:       accepter,
		FriendUid: requester,
		Status:    model.FriendStatusAccepted,
		CreatedAt: time.Now(),
	})
	notify(requester, "friend_accepted", map[string]any{"from_uid": accepter})
	return nil
}

func BlockFriend(ctx context.Context, uid, friendUid string) error {
	if err := updateFriendStatus(ctx, uid, friendUid, model.FriendStatusBlocked); err != nil {
		return fmt.Errorf("block friend: %w", err)
	}
	return nil
}

func RemoveFriend(ctx context.Context, uid, friendUid string) error {
	if err := deleteFriendRel(ctx, uid, friendUid); err != nil {
		return fmt.Errorf("remove friend: %w", err)
	}
	_ = deleteFriendRel(ctx, friendUid, uid)
	return nil
}

func GetFriends(ctx context.Context, uid string) ([]*model.FriendRelation, error) {
	return mysql.FindFriends(ctx, uid)
}

func GetFriendRequests(ctx context.Context, uid string) ([]*model.FriendRelation, error) {
	return mysql.FindFriendRequests(ctx, uid)
}

// GetFriendList 返回好友列表展示信息（含昵称/头像/备注）。
func GetFriendList(ctx context.Context, uid string) ([]*model.FriendInfo, error) {
	return findFriendList(ctx, uid)
}
