package service

import (
	"IM/model"
	"IM/mongdb"
	"IM/mysql"
	"IM/utils"
	"context"
	"errors"
	"strconv"
	"time"
)

// 通过函数变量注入，便于朋友圈逻辑的单元测试（不依赖真实 Mongo/MySQL）。
var (
	insertMoment     = mongdb.InsertMoment
	findMoments      = mongdb.FindMoments
	findMoment       = mongdb.FindMoment
	updateMomentLike = mongdb.UpdateMomentLike
	addMomentComment = mongdb.AddComment
	deleteMoment     = mongdb.DeleteMoment
	getFriendUids    = defaultFriendUids
)

func defaultFriendUids(ctx context.Context, uid string) ([]string, error) {
	friends, err := mysql.FindFriends(ctx, uid)
	if err != nil {
		return nil, err
	}
	uids := make([]string, 0, len(friends))
	for _, f := range friends {
		uids = append(uids, f.FriendUid)
	}
	return uids, nil
}

// PublishMoment 发布一条朋友圈动态（images 为已存储图片的 _id 列表）。
func PublishMoment(ctx context.Context, uid, content string, images []string) (*model.Moment, error) {
	if images == nil {
		images = []string{}
	}
	m := &model.Moment{
		MomentId:  strconv.FormatUint(utils.GenerateId(), 10),
		Uid:       uid,
		Content:   content,
		Images:    images,
		Likes:     []string{},
		Comments:  []model.MomentComment{},
		CreatedAt: time.Now(),
	}
	if err := insertMoment(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// GetTimeline 返回自己 + 好友的动态（时间倒序）。好友查询失败时仍返回自己的。
func GetTimeline(ctx context.Context, uid string) ([]*model.Moment, error) {
	uids := []string{uid}
	if friends, err := getFriendUids(ctx, uid); err == nil {
		uids = append(uids, friends...)
	}
	return findMoments(ctx, uids, 100)
}

// ToggleLike 切换点赞状态，返回切换后的状态（true=已点赞）。
func ToggleLike(ctx context.Context, momentId, uid string) (bool, error) {
	m, err := findMoment(ctx, momentId)
	if err != nil {
		return false, err
	}
	liked := false
	for _, u := range m.Likes {
		if u == uid {
			liked = true
			break
		}
	}
	if err := updateMomentLike(ctx, momentId, uid, !liked); err != nil {
		return false, err
	}
	return !liked, nil
}

// CommentMoment 给动态添加一条评论。
func CommentMoment(ctx context.Context, momentId, uid, content string) (*model.MomentComment, error) {
	c := model.MomentComment{Uid: uid, Content: content, CreatedAt: time.Now()}
	if err := addMomentComment(ctx, momentId, c); err != nil {
		return nil, err
	}
	return &c, nil
}

// DeleteMoment 删除动态，仅允许发布者本人删除。
func DeleteMoment(ctx context.Context, momentId, uid string) error {
	m, err := findMoment(ctx, momentId)
	if err != nil {
		return err
	}
	if m.Uid != uid {
		return errors.New("无权删除他人的动态")
	}
	return deleteMoment(ctx, momentId)
}
