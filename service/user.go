package service

import (
	"IM/mysql"
	"IM/mysql/model"
	"IM/utils"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type RegisterReq struct {
	Name     string `json:"name" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
}

type RegisterResp struct {
	Uid string `json:"uid"`
}

type LoginReq struct {
	Uid      string `json:"uid" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// findUserByLogin 支持 uid/手机/邮箱/昵称 登录；函数变量便于测试注入。
var findUserByLogin = mysql.FindUserByLogin

type LoginResp struct {
	Token string `json:"token"`
	Uid   string `json:"uid"`
	Name  string `json:"name"`
}

type ProfileResp struct {
	Uid       string `json:"uid"`
	Name      string `json:"name"`
	Avatar    string `json:"avatar"`
	Gender    uint64 `json:"gender"`
	Birthday  string `json:"birthday,omitempty"`
	Signature string `json:"signature,omitempty"`
	Email     string `json:"email,omitempty"`
	Phone     string `json:"phone,omitempty"`
	Status    uint64 `json:"status"`
}

type UpdateProfileReq struct {
	Name      string `json:"name"`
	Avatar    string `json:"avatar"`
	Gender    uint64 `json:"gender"`
	Birthday  string `json:"birthday"`
	Signature string `json:"signature"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
}

type ChangePasswordReq struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

func Register(ctx context.Context, req *RegisterReq) (*RegisterResp, error) {
	uid := strconv.FormatUint(utils.GenerateId(), 10)

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now()
	user := &model.User{
		Uid:          uid,
		PasswordHash: string(hash),
		Name:         req.Name,
		Email:        req.Email,
		Phone:        req.Phone,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	_, err = mysql.UserModel.Insert(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}

	return &RegisterResp{Uid: uid}, nil
}

func Login(ctx context.Context, req *LoginReq) (*LoginResp, error) {
	user, err := findUserByLogin(ctx, req.Uid)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, errors.New("invalid uid or password")
		}
		return nil, fmt.Errorf("find user: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		return nil, errors.New("invalid uid or password")
	}

	token, err := utils.GenerateToken(user.Uid, time.Now().Add(24*time.Hour))
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &LoginResp{
		Token: token,
		Uid:   user.Uid,
		Name:  user.Name,
	}, nil
}

func GetProfile(ctx context.Context, uid string) (*ProfileResp, error) {
	user, err := mysql.UserModel.FindOne(ctx, uid)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("find user: %w", err)
	}

	resp := &ProfileResp{
		Uid:       user.Uid,
		Name:      user.Name,
		Avatar:    user.Avatar,
		Gender:    user.Gender,
		Signature: user.Signature,
		Email:     user.Email,
		Phone:     user.Phone,
		Status:    user.Status,
	}
	if user.Birthday.Valid {
		resp.Birthday = user.Birthday.Time.Format("2006-01-02")
	}
	return resp, nil
}

func UpdateProfile(ctx context.Context, uid string, req *UpdateProfileReq) error {
	user, err := mysql.UserModel.FindOne(ctx, uid)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return errors.New("user not found")
		}
		return fmt.Errorf("find user: %w", err)
	}

	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Avatar != "" {
		user.Avatar = req.Avatar
	}
	if req.Gender != 0 {
		user.Gender = req.Gender
	}
	if req.Signature != "" {
		user.Signature = req.Signature
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}
	if req.Birthday != "" {
		t, err := time.Parse("2006-01-02", req.Birthday)
		if err == nil {
			user.Birthday = sql.NullTime{Time: t, Valid: true}
		}
	}
	user.UpdatedAt = time.Now()

	return mysql.UserModel.Update(ctx, user)
}

func ChangePassword(ctx context.Context, uid string, req *ChangePasswordReq) error {
	user, err := mysql.UserModel.FindOne(ctx, uid)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return errors.New("user not found")
		}
		return fmt.Errorf("find user: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword))
	if err != nil {
		return errors.New("old password is incorrect")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	user.PasswordHash = string(hash)
	user.UpdatedAt = time.Now()

	return mysql.UserModel.Update(ctx, user)
}
