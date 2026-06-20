package service

import (
	"IM/mysql/model"
	"context"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// P2 #8：支持以 uid/手机/邮箱/昵称 任意标识登录。

func TestLoginSuccessByIdentifier(t *testing.T) {
	orig := findUserByLogin
	defer func() { findUserByLogin = orig }()

	hash, _ := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
	findUserByLogin = func(ctx context.Context, identifier string) (*model.User, error) {
		// 模拟按昵称/手机/邮箱任意标识查到同一用户
		return &model.User{Uid: "1001", Name: "alice", PasswordHash: string(hash)}, nil
	}

	resp, err := Login(context.Background(), &LoginReq{Uid: "alice", Password: "secret123"})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if resp.Uid != "1001" || resp.Name != "alice" || resp.Token == "" {
		t.Errorf("unexpected resp: %+v", resp)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	orig := findUserByLogin
	defer func() { findUserByLogin = orig }()

	hash, _ := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
	findUserByLogin = func(ctx context.Context, identifier string) (*model.User, error) {
		return &model.User{Uid: "1001", Name: "alice", PasswordHash: string(hash)}, nil
	}

	if _, err := Login(context.Background(), &LoginReq{Uid: "alice", Password: "wrong"}); err == nil {
		t.Fatal("expected error for wrong password")
	}
}

func TestLoginUserNotFound(t *testing.T) {
	orig := findUserByLogin
	defer func() { findUserByLogin = orig }()

	findUserByLogin = func(ctx context.Context, identifier string) (*model.User, error) {
		return nil, model.ErrNotFound
	}

	if _, err := Login(context.Background(), &LoginReq{Uid: "ghost", Password: "x"}); err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}
