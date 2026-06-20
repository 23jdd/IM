package service

import (
	"IM/model"
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestPublishMomentInserts(t *testing.T) {
	orig := insertMoment
	defer func() { insertMoment = orig }()

	var saved *model.Moment
	insertMoment = func(ctx context.Context, m *model.Moment) error {
		saved = m
		return nil
	}

	m, err := PublishMoment(context.Background(), "u1", "hello world", []string{"img1"})
	if err != nil {
		t.Fatal(err)
	}
	if m.Uid != "u1" || m.Content != "hello world" || len(m.Images) != 1 || m.Images[0] != "img1" {
		t.Errorf("unexpected moment: %+v", m)
	}
	if m.MomentId == "" {
		t.Error("moment id should not be empty")
	}
	if saved == nil || saved.MomentId != m.MomentId {
		t.Fatal("insertMoment not called correctly")
	}
}

func TestGetTimelineIncludesSelfAndFriends(t *testing.T) {
	origFriends := getFriendUids
	origFind := findMoments
	defer func() { getFriendUids = origFriends; findMoments = origFind }()

	getFriendUids = func(ctx context.Context, uid string) ([]string, error) {
		return []string{"f1", "f2"}, nil
	}
	var gotUids []string
	findMoments = func(ctx context.Context, uids []string, limit int64) ([]*model.Moment, error) {
		gotUids = uids
		return nil, nil
	}

	if _, err := GetTimeline(context.Background(), "me"); err != nil {
		t.Fatal(err)
	}
	want := []string{"me", "f1", "f2"}
	if !reflect.DeepEqual(gotUids, want) {
		t.Errorf("timeline uids = %v, want %v", gotUids, want)
	}
}

func TestGetTimelineFriendErrorStillReturnsSelf(t *testing.T) {
	origFriends := getFriendUids
	origFind := findMoments
	defer func() { getFriendUids = origFriends; findMoments = origFind }()

	getFriendUids = func(ctx context.Context, uid string) ([]string, error) {
		return nil, errors.New("db down")
	}
	var gotUids []string
	findMoments = func(ctx context.Context, uids []string, limit int64) ([]*model.Moment, error) {
		gotUids = uids
		return nil, nil
	}

	if _, err := GetTimeline(context.Background(), "me"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotUids, []string{"me"}) {
		t.Errorf("uids = %v, want [me]", gotUids)
	}
}

func TestToggleLikeAddsWhenNotLiked(t *testing.T) {
	origFind := findMoment
	origUpd := updateMomentLike
	defer func() { findMoment = origFind; updateMomentLike = origUpd }()

	findMoment = func(ctx context.Context, id string) (*model.Moment, error) {
		return &model.Moment{MomentId: id, Likes: []string{"other"}}, nil
	}
	var gotAdd bool
	updateMomentLike = func(ctx context.Context, momentId, uid string, add bool) error {
		gotAdd = add
		return nil
	}

	liked, err := ToggleLike(context.Background(), "m1", "me")
	if err != nil {
		t.Fatal(err)
	}
	if !liked || !gotAdd {
		t.Errorf("expected liked=true add=true, got liked=%v add=%v", liked, gotAdd)
	}
}

func TestToggleLikeRemovesWhenLiked(t *testing.T) {
	origFind := findMoment
	origUpd := updateMomentLike
	defer func() { findMoment = origFind; updateMomentLike = origUpd }()

	findMoment = func(ctx context.Context, id string) (*model.Moment, error) {
		return &model.Moment{MomentId: id, Likes: []string{"me", "other"}}, nil
	}
	var gotAdd bool
	updateMomentLike = func(ctx context.Context, momentId, uid string, add bool) error {
		gotAdd = add
		return nil
	}

	liked, err := ToggleLike(context.Background(), "m1", "me")
	if err != nil {
		t.Fatal(err)
	}
	if liked || gotAdd {
		t.Errorf("expected liked=false add=false, got liked=%v add=%v", liked, gotAdd)
	}
}

func TestDeleteMomentByOwner(t *testing.T) {
	origFind := findMoment
	origDel := deleteMoment
	defer func() { findMoment = origFind; deleteMoment = origDel }()

	findMoment = func(ctx context.Context, id string) (*model.Moment, error) {
		return &model.Moment{MomentId: id, Uid: "me"}, nil
	}
	deleted := ""
	deleteMoment = func(ctx context.Context, id string) error {
		deleted = id
		return nil
	}

	if err := DeleteMoment(context.Background(), "m1", "me"); err != nil {
		t.Fatal(err)
	}
	if deleted != "m1" {
		t.Errorf("deleted = %s, want m1", deleted)
	}
}

func TestDeleteMomentByOtherRejected(t *testing.T) {
	origFind := findMoment
	origDel := deleteMoment
	defer func() { findMoment = origFind; deleteMoment = origDel }()

	findMoment = func(ctx context.Context, id string) (*model.Moment, error) {
		return &model.Moment{MomentId: id, Uid: "owner"}, nil
	}
	delCalled := false
	deleteMoment = func(ctx context.Context, id string) error {
		delCalled = true
		return nil
	}

	if err := DeleteMoment(context.Background(), "m1", "intruder"); err == nil {
		t.Fatal("expected error deleting other's moment")
	}
	if delCalled {
		t.Error("deleteMoment must not be called for non-owner")
	}
}

func TestDeleteMomentNotFound(t *testing.T) {
	origFind := findMoment
	defer func() { findMoment = origFind }()

	findMoment = func(ctx context.Context, id string) (*model.Moment, error) {
		return nil, errors.New("not found")
	}

	if err := DeleteMoment(context.Background(), "ghost", "me"); err == nil {
		t.Fatal("expected error for nonexistent moment")
	}
}
