package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
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

type UserBriefInfo struct {
	Uid       string `json:"uid"`
	Name      string `json:"name"`
	Avatar    string `json:"avatar"`
	Gender    int    `json:"gender"`
	Signature string `json:"signature"`
}

// UserInfo 按 uid 查询用户公开信息（搜索添加好友前预览）。
func (a *AuthService) UserInfo(token, uid string) (*UserBriefInfo, error) {
	var out UserBriefInfo
	if err := a.do(http.MethodGet, "/api/user/info?uid="+url.QueryEscape(uid), token, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// MessageRecall 撤回自己的消息。
func (a *AuthService) MessageRecall(token, msgId string) error {
	return a.do(http.MethodPost, "/api/message/recall", token, map[string]string{
		"msg_id": msgId,
	}, nil)
}

type HistoryMessageInfo struct {
	MsgId     string `json:"msg_id"`
	FromUid   string `json:"from_uid"`
	ToUid     string `json:"to_uid"`
	GroupId   string `json:"group_id"`
	MsgType   int    `json:"msg_type"`
	Content   string `json:"content"`
	Status    int    `json:"status"`
	CreatedAt string `json:"created_at"`
}

// GetChatHistory 分页拉取历史消息。peer 与 group 二选一；before 为 unix 毫秒游标（0 表示最新一页）。
func (a *AuthService) GetChatHistory(token, peer, group string, before int64, limit int) ([]HistoryMessageInfo, error) {
	q := url.Values{}
	if peer != "" {
		q.Set("peer", peer)
	}
	if group != "" {
		q.Set("group", group)
	}
	if before > 0 {
		q.Set("before", strconv.FormatInt(before, 10))
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	var out []HistoryMessageInfo
	if err := a.do(http.MethodGet, "/api/message/history?"+q.Encode(), token, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetFriends 获取好友列表。
func (a *AuthService) GetFriends(token string) ([]FriendInfo, error) {
	var out []FriendInfo
	if err := a.do(http.MethodGet, "/api/friend/list", token, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type FriendRequestInfo struct {
	Uid       string `json:"uid"`        // 申请人
	FriendUid string `json:"friend_uid"` // 接收方（我）
	Status    int    `json:"status"`
	Remark    string `json:"remark"`
}

// FriendRequest 向对方发起好友申请。
func (a *AuthService) FriendRequest(token, friendUid, remark string) error {
	return a.do(http.MethodPost, "/api/friend/request", token, map[string]string{
		"friend_uid": friendUid,
		"remark":     remark,
	}, nil)
}

// FriendRequests 获取收到的好友申请列表。
func (a *AuthService) FriendRequests(token string) ([]FriendRequestInfo, error) {
	var out []FriendRequestInfo
	if err := a.do(http.MethodGet, "/api/friend/requests", token, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// FriendAccept 接受好友申请。
func (a *AuthService) FriendAccept(token, friendUid string) error {
	return a.do(http.MethodPost, "/api/friend/accept", token, map[string]string{
		"friend_uid": friendUid,
	}, nil)
}

// FriendRemove 删除好友。
func (a *AuthService) FriendRemove(token, friendUid string) error {
	return a.do(http.MethodPost, "/api/friend/remove", token, map[string]string{
		"friend_uid": friendUid,
	}, nil)
}

// FriendBlock 把对方加入黑名单。
func (a *AuthService) FriendBlock(token, friendUid string) error {
	return a.do(http.MethodPost, "/api/friend/block", token, map[string]string{
		"friend_uid": friendUid,
	}, nil)
}

// FriendUnblock 把对方移出黑名单。
func (a *AuthService) FriendUnblock(token, friendUid string) error {
	return a.do(http.MethodPost, "/api/friend/unblock", token, map[string]string{
		"friend_uid": friendUid,
	}, nil)
}

// FriendBlockList 获取黑名单列表。
func (a *AuthService) FriendBlockList(token string) ([]FriendInfo, error) {
	var out []FriendInfo
	if err := a.do(http.MethodGet, "/api/friend/blocklist", token, nil, &out); err != nil {
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

// GroupInvite 邀请好友入群（仅群成员可邀请）。
func (a *AuthService) GroupInvite(token, groupId, friendUid string) error {
	return a.do(http.MethodPost, "/api/group/invite", token, map[string]string{
		"group_id":   groupId,
		"friend_uid": friendUid,
	}, nil)
}

type GroupJoinRequestInfo struct {
	Uid       string `json:"uid"`
	GroupId   string `json:"group_id"`
	Status    int    `json:"status"`
	CreatedAt string `json:"created_at"`
}

// GroupJoinRequests 群主查看某群的待审批入群申请。
func (a *AuthService) GroupJoinRequests(token, groupId string) ([]GroupJoinRequestInfo, error) {
	var out []GroupJoinRequestInfo
	if err := a.do(http.MethodGet, "/api/group/requests?group_id="+url.QueryEscape(groupId), token, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GroupApprove 群主通过入群申请。
func (a *AuthService) GroupApprove(token, groupId, applicantUid string) error {
	return a.do(http.MethodPost, "/api/group/approve", token, map[string]string{
		"group_id": groupId,
		"uid":      applicantUid,
	}, nil)
}

// GroupReject 群主拒绝入群申请。
func (a *AuthService) GroupReject(token, groupId, applicantUid string) error {
	return a.do(http.MethodPost, "/api/group/reject", token, map[string]string{
		"group_id": groupId,
		"uid":      applicantUid,
	}, nil)
}

type GroupMemberInfo struct {
	Uid       string `json:"uid"`
	Role      int    `json:"role"`
	Nickname  string `json:"nickname"`
	MuteUntil string `json:"mute_until"`
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

type GroupInfoData struct {
	GroupId      string `json:"group_id"`
	Name         string `json:"name"`
	OwnerUid     string `json:"owner_uid"`
	Description  string `json:"description"`
	Announcement string `json:"announcement"`
	Status       int    `json:"status"`
}

// GroupInfo 获取群资料（含群公告）。
func (a *AuthService) GroupInfo(token, groupId string) (*GroupInfoData, error) {
	var out GroupInfoData
	if err := a.do(http.MethodGet, "/api/group/info?group_id="+url.QueryEscape(groupId), token, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GroupLeave 退出群聊（群主不可退，需先转让或解散）。
func (a *AuthService) GroupLeave(token, groupId string) error {
	return a.do(http.MethodPost, "/api/group/leave", token, map[string]string{
		"group_id": groupId,
	}, nil)
}

// GroupDisband 解散群（仅群主）。
func (a *AuthService) GroupDisband(token, groupId string) error {
	return a.do(http.MethodPost, "/api/group/disband", token, map[string]string{
		"group_id": groupId,
	}, nil)
}

// GroupKick 踢出群成员（群主/管理员）。
func (a *AuthService) GroupKick(token, groupId, targetUid string) error {
	return a.do(http.MethodPost, "/api/group/kick", token, map[string]string{
		"group_id": groupId,
		"uid":      targetUid,
	}, nil)
}

// GroupTransfer 转让群主（仅群主）。
func (a *AuthService) GroupTransfer(token, groupId, targetUid string) error {
	return a.do(http.MethodPost, "/api/group/transfer", token, map[string]string{
		"group_id": groupId,
		"uid":      targetUid,
	}, nil)
}

// GroupMute 禁言/解除禁言成员（minutes<=0 表示解除）。
func (a *AuthService) GroupMute(token, groupId, targetUid string, minutes int) error {
	return a.do(http.MethodPost, "/api/group/mute", token, map[string]any{
		"group_id": groupId,
		"uid":      targetUid,
		"minutes":  minutes,
	}, nil)
}

// GroupAnnounce 设置群公告（群主/管理员）。
func (a *AuthService) GroupAnnounce(token, groupId, announcement string) error {
	return a.do(http.MethodPost, "/api/group/announce", token, map[string]string{
		"group_id":     groupId,
		"announcement": announcement,
	}, nil)
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

// UploadFile 上传任意文件（base64），返回 file_id（复用通用二进制存储，下载经 /api/avatar?id=）。
func (a *AuthService) UploadFile(token, dataBase64, contentType string) (string, error) {
	var out struct {
		FileId string `json:"file_id"`
	}
	if err := a.do(http.MethodPost, "/api/file/upload", token, map[string]string{
		"data_base64":  dataBase64,
		"content_type": contentType,
	}, &out); err != nil {
		return "", err
	}
	return out.FileId, nil
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

type MomentCommentInfo struct {
	Uid       string `json:"uid"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

type MomentInfo struct {
	MomentId  string              `json:"moment_id"`
	Uid       string              `json:"uid"`
	Content   string              `json:"content"`
	Images    []string            `json:"images"`
	Likes     []string            `json:"likes"`
	Comments  []MomentCommentInfo `json:"comments"`
	CreatedAt string              `json:"created_at"`
}

// MomentPublish 发布朋友圈动态。images 为图片 data URL 列表。
func (a *AuthService) MomentPublish(token, content string, images []string) (*MomentInfo, error) {
	var out MomentInfo
	if err := a.do(http.MethodPost, "/api/moment/publish", token, map[string]any{
		"content": content,
		"images":  images,
	}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// MomentTimeline 获取朋友圈时间线（自己 + 好友）。
func (a *AuthService) MomentTimeline(token string) ([]MomentInfo, error) {
	var out []MomentInfo
	if err := a.do(http.MethodGet, "/api/moment/timeline", token, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// MomentLike 切换点赞，返回切换后的状态。
func (a *AuthService) MomentLike(token, momentId string) (bool, error) {
	var out struct {
		Liked bool `json:"liked"`
	}
	if err := a.do(http.MethodPost, "/api/moment/like", token, map[string]string{
		"moment_id": momentId,
	}, &out); err != nil {
		return false, err
	}
	return out.Liked, nil
}

// MomentComment 评论动态。
func (a *AuthService) MomentComment(token, momentId, content string) error {
	return a.do(http.MethodPost, "/api/moment/comment", token, map[string]string{
		"moment_id": momentId,
		"content":   content,
	}, nil)
}

// MomentDelete 删除自己的动态。
func (a *AuthService) MomentDelete(token, momentId string) error {
	return a.do(http.MethodPost, "/api/moment/delete", token, map[string]string{
		"moment_id": momentId,
	}, nil)
}
