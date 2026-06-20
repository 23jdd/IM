package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// AuthService 通过 Go 侧调用后端 HTTP REST API（:8080），
// 避免 WebView 跨域问题。前端通过绑定调用这些方法。
type AuthService struct {
	baseURL string
	client  *http.Client
}

func NewAuthService() *AuthService {
	return &AuthService{
		baseURL: "http://127.0.0.1:8080",
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

// SetBaseURL 允许前端覆盖后端地址。
func (a *AuthService) SetBaseURL(url string) {
	if url != "" {
		a.baseURL = url
	}
}

type apiResponse struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func (a *AuthService) do(method, path, token string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, a.baseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var r apiResponse
	if err := json.Unmarshal(raw, &r); err != nil {
		return fmt.Errorf("响应解析失败: %s", string(raw))
	}
	if r.Code != 0 {
		return errors.New(r.Msg)
	}
	if out != nil && len(r.Data) > 0 {
		return json.Unmarshal(r.Data, out)
	}
	return nil
}

type RegisterResult struct {
	Uid string `json:"uid"`
}

// Register 注册新用户，返回分配的 uid。
func (a *AuthService) Register(name, password, email, phone string) (*RegisterResult, error) {
	var out RegisterResult
	err := a.do(http.MethodPost, "/api/user/register", "", map[string]string{
		"name":     name,
		"password": password,
		"email":    email,
		"phone":    phone,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

type LoginResult struct {
	Token string `json:"token"`
	Uid   string `json:"uid"`
	Name  string `json:"name"`
}

// Login 使用 uid + 密码登录，返回 JWT token。
func (a *AuthService) Login(uid, password string) (*LoginResult, error) {
	var out LoginResult
	err := a.do(http.MethodPost, "/api/user/login", "", map[string]string{
		"uid":      uid,
		"password": password,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

type Profile struct {
	Uid       string `json:"uid"`
	Name      string `json:"name"`
	Avatar    string `json:"avatar"`
	Gender    uint64 `json:"gender"`
	Birthday  string `json:"birthday"`
	Signature string `json:"signature"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	Status    uint64 `json:"status"`
}

// GetProfile 获取当前登录用户资料。
func (a *AuthService) GetProfile(token string) (*Profile, error) {
	var out Profile
	err := a.do(http.MethodGet, "/api/user/profile", token, nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateProfile 更新用户资料。
func (a *AuthService) UpdateProfile(token string, p Profile) error {
	return a.do(http.MethodPut, "/api/user/profile", token, map[string]any{
		"name":      p.Name,
		"avatar":    p.Avatar,
		"gender":    p.Gender,
		"birthday":  p.Birthday,
		"signature": p.Signature,
		"email":     p.Email,
		"phone":     p.Phone,
	}, nil)
}

// ChangePassword 修改密码。
func (a *AuthService) ChangePassword(token, oldPassword, newPassword string) error {
	return a.do(http.MethodPut, "/api/user/password", token, map[string]string{
		"old_password": oldPassword,
		"new_password": newPassword,
	}, nil)
}

type FriendInfo struct {
	Uid    string `json:"uid"`
	Remark string `json:"remark"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

// GetFriends 获取好友列表。
func (a *AuthService) GetFriends(token string) ([]FriendInfo, error) {
	var out []FriendInfo
	if err := a.do(http.MethodGet, "/api/friend/list", token, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type ConversationInfo struct {
	Peer    string `json:"peer"`
	Content string `json:"content"`
	Time    string `json:"time"`
}

// GetConversations 获取会话列表（最近联系人）。
func (a *AuthService) GetConversations(token string) ([]ConversationInfo, error) {
	var out []ConversationInfo
	if err := a.do(http.MethodGet, "/api/conversation/list", token, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}
