package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

type GroupBrief struct {
	GroupId string `json:"group_id"`
	Name    string `json:"name"`
}

// GroupCreate 创建群组（创建者自动成为群主成员）。
func (a *AuthService) GroupCreate(token, name, description string) (*GroupBrief, error) {
	var out GroupBrief
	if err := a.do(http.MethodPost, "/api/group/create", token, map[string]string{
		"name":        name,
		"description": description,
	}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GroupList 获取我加入的群列表。
func (a *AuthService) GroupList(token string) ([]GroupBrief, error) {
	var out []GroupBrief
	if err := a.do(http.MethodGet, "/api/group/list", token, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GroupJoin 加入群组。
func (a *AuthService) GroupJoin(token, groupId string) error {
	return a.do(http.MethodPost, "/api/group/join", token, map[string]string{
		"group_id": groupId,
	}, nil)
}

type GroupMemberInfo struct {
	Uid      string `json:"uid"`
	Role     int    `json:"role"`
	Nickname string `json:"nickname"`
}

// GroupMembers 获取群成员列表。
func (a *AuthService) GroupMembers(token, groupId string) ([]GroupMemberInfo, error) {
	var out []GroupMemberInfo
	path := "/api/group/members?group_id=" + url.QueryEscape(groupId)
	if err := a.do(http.MethodGet, path, token, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// UploadAvatar 上传头像（base64 图片），存 MongoDB 并更新 MySQL，返回图片 _id。
func (a *AuthService) UploadAvatar(token, dataBase64, contentType string) (string, error) {
	var out struct {
		Avatar string `json:"avatar"`
	}
	if err := a.do(http.MethodPost, "/api/user/avatar", token, map[string]string{
		"data_base64":  dataBase64,
		"content_type": contentType,
	}, &out); err != nil {
		return "", err
	}
	return out.Avatar, nil
}

// GetAvatar 按 _id 读取头像，返回可直接用于 <img src> 的 data URL（无图返回空串）。
func (a *AuthService) GetAvatar(token, id string) (string, error) {
	if id == "" {
		return "", nil
	}
	var out struct {
		ContentType string `json:"content_type"`
		Data        string `json:"data"`
	}
	if err := a.do(http.MethodGet, "/api/avatar?id="+url.QueryEscape(id), token, nil, &out); err != nil {
		return "", err
	}
	if out.Data == "" {
		return "", nil
	}
	ct := out.ContentType
	if ct == "" {
		ct = "image/png"
	}
	return "data:" + ct + ";base64," + out.Data, nil
}

// GetAvatarByUid 按用户 uid 解析头像，返回可直接用于 <img src> 的 data URL（无头像返回空串）。
func (a *AuthService) GetAvatarByUid(token, uid string) (string, error) {
	if uid == "" {
		return "", nil
	}
	var out struct {
		ContentType string `json:"content_type"`
		Data        string `json:"data"`
	}
	if err := a.do(http.MethodGet, "/api/avatar/by-uid?uid="+url.QueryEscape(uid), token, nil, &out); err != nil {
		return "", err
	}
	if out.Data == "" {
		return "", nil
	}
	ct := out.ContentType
	if ct == "" {
		ct = "image/png"
	}
	return "data:" + ct + ";base64," + out.Data, nil
}
