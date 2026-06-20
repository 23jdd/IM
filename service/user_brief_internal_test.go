package service

import (
	"IM/model"
	"context"
	"errors"
	"testing"
)

func TestGetUserBriefReturnsUser(t *testing.T) {
	orig := findUserBrief
	defer func() { findUserBrief = orig }()

	findUserBrief = func(ctx context.Context, uid string) (*model.UserBrief, error) {
		return &model.UserBrief{Uid: uid, Name: "Alice", Avatar: "img1"}, nil
	}

	u, err := GetUserBrief(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if u.Uid != "u1" || u.Name != "Alice" || u.Avatar != "img1" {
		t.Errorf("unexpected user brief: %+v", u)
	}
}

func TestGetUserBriefNotFound(t *testing.T) {
	orig := findUserBrief
	defer func() { findUserBrief = orig }()

	findUserBrief = func(ctx context.Context, uid string) (*model.UserBrief, error) {
		return nil, errors.New("not found")
	}

	if _, err := GetUserBrief(context.Background(), "ghost"); err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}
