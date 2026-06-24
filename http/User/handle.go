package User

import (
	"IM/service"
	"IM/utils"
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Response 统一的 HTTP 响应结构。
type Response struct {
	Code int    `json:"code"` // 业务状态码，0 表示成功
	Msg  string `json:"msg"`  // 提示信息
	Data any    `json:"data,omitempty"` // 业务数据，可选
}

// ok 返回成功响应。
func ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, &Response{Code: 0, Msg: "ok", Data: data})
}

// fail 返回失败响应（HTTP 状态仍为 200，通过 code 区分业务错误）。
func fail(c *gin.Context, code int, msg string) {
	c.JSON(http.StatusOK, &Response{Code: code, Msg: msg})
}

// Register 用户注册。
func Register(c *gin.Context) {
	var req service.RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, -1, err.Error())
		return
	}

	resp, err := service.Register(c.Request.Context(), &req)
	if err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, resp)
}

// Login 用户登录。
func Login(c *gin.Context) {
	var req service.LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, -1, err.Error())
		return
	}

	resp, err := service.Login(c.Request.Context(), &req)
	if err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, resp)
}

// GetProfile 获取当前登录用户的资料。
func GetProfile(c *gin.Context) {
	uid := c.GetString("uid")

	resp, err := service.GetProfile(c.Request.Context(), uid)
	if err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, resp)
}

// UpdateProfile 更新当前用户的资料。
func UpdateProfile(c *gin.Context) {
	uid := c.GetString("uid")

	var req service.UpdateProfileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, -1, err.Error())
		return
	}

	err := service.UpdateProfile(c.Request.Context(), uid, &req)
	if err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, nil)
}

// ChangePassword 修改当前用户密码。
func ChangePassword(c *gin.Context) {
	uid := c.GetString("uid")

	var req service.ChangePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, -1, err.Error())
		return
	}

	err := service.ChangePassword(c.Request.Context(), uid, &req)
	if err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, nil)
}

// GetFriends 获取当前用户的好友列表。
func GetFriends(c *gin.Context) {
	uid := c.GetString("uid")

	resp, err := service.GetFriendList(c.Request.Context(), uid)
	if err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, resp)
}

// GetConversations 获取当前用户的会话列表。
func GetConversations(c *gin.Context) {
	uid := c.GetString("uid")

	resp, err := service.GetConversations(c.Request.Context(), uid)
	if err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, resp)
}

// CreateGroup 创建群组。
func CreateGroup(c *gin.Context) {
	uid := c.GetString("uid")
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, -1, err.Error())
		return
	}
	g, err := service.CreateGroup(c.Request.Context(), uid, req.Name, req.Description)
	if err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, gin.H{"group_id": g.GroupId, "name": g.Name})
}

// GetMyGroups 获取当前用户加入的群组列表。
func GetMyGroups(c *gin.Context) {
	uid := c.GetString("uid")
	resp, err := service.GetUserGroupList(c.Request.Context(), uid)
	if err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, resp)
}

// GetGroupMembers 获取指定群组的成员列表。
func GetGroupMembers(c *gin.Context) {
	groupId := c.Query("group_id")
	if groupId == "" {
		fail(c, -1, "group_id required")
		return
	}
	resp, err := service.GetGroupMembers(c.Request.Context(), groupId)
	if err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, resp)
}

// JoinGroup 申请加入群组。
func JoinGroup(c *gin.Context) {
	uid := c.GetString("uid")
	var req struct {
		GroupId string `json:"group_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, -1, err.Error())
		return
	}
	if err := service.RequestJoinGroup(c.Request.Context(), req.GroupId, uid); err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, nil)
}

// GroupJoinRequests 获取群组的入群申请列表（需管理员权限）。
func GroupJoinRequests(c *gin.Context) {
	uid := c.GetString("uid")
	groupId := c.Query("group_id")
	if groupId == "" {
		fail(c, -1, "group_id required")
		return
	}
	resp, err := service.GetGroupJoinRequests(c.Request.Context(), groupId, uid)
	if err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, resp)
}

// ApproveJoin 通过入群申请。
func ApproveJoin(c *gin.Context) {
	uid := c.GetString("uid")
	var req struct {
		GroupId string `json:"group_id" binding:"required"`
		Uid     string `json:"uid" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, -1, err.Error())
		return
	}
	if err := service.ApproveJoinRequest(c.Request.Context(), req.GroupId, uid, req.Uid); err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, nil)
}

// RejectJoin 拒绝入群申请。
func RejectJoin(c *gin.Context) {
	uid := c.GetString("uid")
	var req struct {
		GroupId string `json:"group_id" binding:"required"`
		Uid     string `json:"uid" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, -1, err.Error())
		return
	}
	if err := service.RejectJoinRequest(c.Request.Context(), req.GroupId, uid, req.Uid); err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, nil)
}

// GetGroupInfo 获取群组详细信息。
func GetGroupInfo(c *gin.Context) {
	groupId := c.Query("group_id")
	if groupId == "" {
		fail(c, -1, "group_id required")
		return
	}
	g, err := service.GetGroup(c.Request.Context(), groupId)
	if err != nil {
		fail(c, -1, "群不存在")
		return
	}
	ok(c, gin.H{
		"group_id":     g.GroupId,
		"name":         g.Name,
		"owner_uid":    g.OwnerUid,
		"description":  g.Description,
		"announcement": g.Announcement,
		"status":       g.Status,
	})
}

// LeaveGroup 退出群组。
func LeaveGroup(c *gin.Context) {
	uid := c.GetString("uid")
	var req struct {
		GroupId string `json:"group_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, -1, err.Error())
		return
	}
	if err := service.LeaveGroup(c.Request.Context(), req.GroupId, uid); err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, nil)
}

// DisbandGroup 解散群组（仅群主可操作）。
func DisbandGroup(c *gin.Context) {
	uid := c.GetString("uid")
	var req struct {
		GroupId string `json:"group_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, -1, err.Error())
		return
	}
	if err := service.DisbandGroup(c.Request.Context(), req.GroupId, uid); err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, nil)
}

// KickGroupMember 将成员移出群组。
func KickGroupMember(c *gin.Context) {
	uid := c.GetString("uid")
	var req struct {
		GroupId string `json:"group_id" binding:"required"`
		Uid     string `json:"uid" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, -1, err.Error())
		return
	}
	if err := service.KickMember(c.Request.Context(), req.GroupId, uid, req.Uid); err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, nil)
}

// TransferGroup 转让群主身份。
func TransferGroup(c *gin.Context) {
	uid := c.GetString("uid")
	var req struct {
		GroupId string `json:"group_id" binding:"required"`
		Uid     string `json:"uid" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, -1, err.Error())
		return
	}
	if err := service.TransferGroupOwner(c.Request.Context(), req.GroupId, uid, req.Uid); err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, nil)
}

// MuteGroupMember 禁言群成员，minutes 为禁言时长（分钟）。
func MuteGroupMember(c *gin.Context) {
	uid := c.GetString("uid")
	var req struct {
		GroupId string `json:"group_id" binding:"required"`
		Uid     string `json:"uid" binding:"required"`
		Minutes int    `json:"minutes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, -1, err.Error())
		return
	}
	if err := service.MuteMember(c.Request.Context(), req.GroupId, uid, req.Uid, req.Minutes); err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, nil)
}

// SetGroupAnnouncement 设置群公告。
func SetGroupAnnouncement(c *gin.Context) {
	uid := c.GetString("uid")
	var req struct {
		GroupId      string `json:"group_id" binding:"required"`
		Announcement string `json:"announcement"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, -1, err.Error())
		return
	}
	if err := service.SetGroupAnnouncement(c.Request.Context(), req.GroupId, uid, req.Announcement); err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, nil)
}

// UploadAvatar 上传头像（base64 编码），返回头像 ID。
func UploadAvatar(c *gin.Context) {
	uid := c.GetString("uid")
	var req struct {
		DataBase64  string `json:"data_base64" binding:"required"`
		ContentType string `json:"content_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, -1, err.Error())
		return
	}
	data, err := base64.StdEncoding.DecodeString(req.DataBase64)
	if err != nil {
		fail(c, -1, "invalid base64 data")
		return
	}
	ct := req.ContentType
	if ct == "" {
		ct = "image/png"
	}
	id, err := service.UploadAvatar(c.Request.Context(), uid, data, ct)
	if err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, gin.H{"avatar": id})
}

// GetAvatar 按头像 ID 获取头像数据（base64 返回）。
func GetAvatar(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		fail(c, -1, "id required")
		return
	}
	data, ct, err := service.GetAvatar(c.Request.Context(), id)
	if err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, gin.H{
		"content_type": ct,
		"data":         base64.StdEncoding.EncodeToString(data),
	})
}

// GetAvatarByUid 按用户 ID 获取其头像数据（base64 返回）。
func GetAvatarByUid(c *gin.Context) {
	uid := c.Query("uid")
	if uid == "" {
		fail(c, -1, "uid required")
		return
	}
	data, ct, err := service.GetAvatarByUid(c.Request.Context(), uid)
	if err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, gin.H{
		"content_type": ct,
		"data":         base64.StdEncoding.EncodeToString(data),
	})
}

// parseDataURL 解析 data URL，返回内容类型与去掉前缀后的 base64 数据；非 data URL 时按默认类型处理。
func parseDataURL(s string) (contentType, data string) {
	contentType = "image/jpeg"
	idx := strings.Index(s, ",")
	if idx < 0 {
		return contentType, s // 没有逗号分隔，视为纯数据
	}
	meta := s[:idx]
	data = s[idx+1:]
	if strings.HasPrefix(meta, "data:") {
		meta = meta[len("data:"):]
		if semi := strings.Index(meta, ";"); semi >= 0 {
			meta = meta[:semi] // 去掉 ;base64 等参数，仅保留 MIME 类型
		}
		if meta != "" {
			contentType = meta
		}
	}
	return contentType, data
}

// PublishMoment 发布朋友圈动态，支持文字与多张图片。
func PublishMoment(c *gin.Context) {
	uid := c.GetString("uid")
	var req struct {
		Content string   `json:"content"`
		Images  []string `json:"images"` // data URL 列表
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, -1, err.Error())
		return
	}
	if req.Content == "" && len(req.Images) == 0 {
		fail(c, -1, "content or images required")
		return
	}
	imageIds := make([]string, 0, len(req.Images))
	for _, dataURL := range req.Images {
		ct, raw := parseDataURL(dataURL)
		data, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			continue // 单张图片解码失败则跳过，不影响整体发布
		}
		id, err := service.StoreImage(c.Request.Context(), data, ct)
		if err == nil {
			imageIds = append(imageIds, id)
		}
	}
	m, err := service.PublishMoment(c.Request.Context(), uid, req.Content, imageIds)
	if err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, m)
}

// GetTimeline 获取当前用户的朋友圈时间线。
func GetTimeline(c *gin.Context) {
	uid := c.GetString("uid")
	resp, err := service.GetTimeline(c.Request.Context(), uid)
	if err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, resp)
}

// LikeMoment 点赞/取消点赞动态，返回当前是否已点赞。
func LikeMoment(c *gin.Context) {
	uid := c.GetString("uid")
	var req struct {
		MomentId string `json:"moment_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, -1, err.Error())
		return
	}
	liked, err := service.ToggleLike(c.Request.Context(), req.MomentId, uid)
	if err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, gin.H{"liked": liked})
}

// CommentMoment 评论动态。
func CommentMoment(c *gin.Context) {
	uid := c.GetString("uid")
	var req struct {
		MomentId string `json:"moment_id" binding:"required"`
		Content  string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, -1, err.Error())
		return
	}
	cm, err := service.CommentMoment(c.Request.Context(), req.MomentId, uid, req.Content)
	if err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, cm)
}

// DeleteMoment 删除自己的动态。
func DeleteMoment(c *gin.Context) {
	uid := c.GetString("uid")
	var req struct {
		MomentId string `json:"moment_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, -1, err.Error())
		return
	}
	if err := service.DeleteMoment(c.Request.Context(), req.MomentId, uid); err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, nil)
}

// AddFriend 发送好友请求。
func AddFriend(c *gin.Context) {
	uid := c.GetString("uid")
	var req struct {
		FriendUid string `json:"friend_uid" binding:"required"`
		Remark    string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, -1, err.Error())
		return
	}
	if err := service.SendFriendRequest(c.Request.Context(), uid, req.FriendUid, req.Remark); err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, nil)
}

// GetFriendRequests 获取收到的好友请求列表。
func GetFriendRequests(c *gin.Context) {
	uid := c.GetString("uid")
	resp, err := service.GetFriendRequests(c.Request.Context(), uid)
	if err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, resp)
}

// AcceptFriend 接受好友请求。
func AcceptFriend(c *gin.Context) {
	uid := c.GetString("uid")
	var req struct {
		FriendUid string `json:"friend_uid" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, -1, err.Error())
		return
	}
	if err := service.AcceptFriendRequest(c.Request.Context(), uid, req.FriendUid); err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, nil)
}

// RemoveFriend 删除好友。
func RemoveFriend(c *gin.Context) {
	uid := c.GetString("uid")
	var req struct {
		FriendUid string `json:"friend_uid" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, -1, err.Error())
		return
	}
	if err := service.RemoveFriend(c.Request.Context(), uid, req.FriendUid); err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, nil)
}

// BlockFriend 拉黑好友。
func BlockFriend(c *gin.Context) {
	uid := c.GetString("uid")
	var req struct {
		FriendUid string `json:"friend_uid" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, -1, err.Error())
		return
	}
	if err := service.BlockFriend(c.Request.Context(), uid, req.FriendUid); err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, nil)
}

// UnblockFriend 取消拉黑好友。
func UnblockFriend(c *gin.Context) {
	uid := c.GetString("uid")
	var req struct {
		FriendUid string `json:"friend_uid" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, -1, err.Error())
		return
	}
	if err := service.UnblockFriend(c.Request.Context(), uid, req.FriendUid); err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, nil)
}

// GetBlockedList 获取黑名单列表。
func GetBlockedList(c *gin.Context) {
	uid := c.GetString("uid")
	resp, err := service.GetBlockedList(c.Request.Context(), uid)
	if err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, resp)
}

// UpdateFriendRemark 修改好友备注。
func UpdateFriendRemark(c *gin.Context) {
	uid := c.GetString("uid")
	var req struct {
		FriendUid string `json:"friend_uid" binding:"required"`
		Remark    string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, -1, err.Error())
		return
	}
	if err := service.UpdateFriendRemark(c.Request.Context(), uid, req.FriendUid, req.Remark); err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, nil)
}

// InviteToGroup 邀请好友加入群组。
func InviteToGroup(c *gin.Context) {
	uid := c.GetString("uid")
	var req struct {
		GroupId   string `json:"group_id" binding:"required"`
		FriendUid string `json:"friend_uid" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, -1, err.Error())
		return
	}
	if err := service.InviteToGroup(c.Request.Context(), req.GroupId, uid, req.FriendUid); err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, nil)
}

// GetUserInfo 按 uid 获取用户简要信息。
func GetUserInfo(c *gin.Context) {
	uid := c.Query("uid")
	if uid == "" {
		fail(c, -1, "uid required")
		return
	}
	u, err := service.GetUserBrief(c.Request.Context(), uid)
	if err != nil {
		fail(c, -1, "用户不存在")
		return
	}
	ok(c, u)
}

// UploadFile 上传文件（base64 编码），返回文件 ID。
func UploadFile(c *gin.Context) {
	var req struct {
		DataBase64  string `json:"data_base64" binding:"required"`
		ContentType string `json:"content_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, -1, err.Error())
		return
	}
	data, err := base64.StdEncoding.DecodeString(req.DataBase64)
	if err != nil {
		fail(c, -1, "invalid base64 data")
		return
	}
	ct := req.ContentType
	if ct == "" {
		ct = "application/octet-stream"
	}
	id, err := service.StoreImage(c.Request.Context(), data, ct)
	if err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, gin.H{"file_id": id})
}

// RecallMessage 撤回消息。
func RecallMessage(c *gin.Context) {
	uid := c.GetString("uid")
	var req struct {
		MsgId string `json:"msg_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, -1, err.Error())
		return
	}
	if err := service.RecallMessage(c.Request.Context(), uid, req.MsgId); err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, nil)
}

// GetChatHistory 分页拉取历史消息。
// 查询参数：peer（单聊对端，二选一）/ group（群聊）；before（unix 毫秒游标，可选）；limit（可选）。
func GetChatHistory(c *gin.Context) {
	uid := c.GetString("uid")
	peer := c.Query("peer")
	group := c.Query("group")
	if peer == "" && group == "" {
		fail(c, -1, "peer or group required")
		return
	}

	var before time.Time
	if ms, err := strconv.ParseInt(c.Query("before"), 10, 64); err == nil && ms > 0 {
		before = time.UnixMilli(ms).UTC()
	}
	var limit int64
	if n, err := strconv.ParseInt(c.Query("limit"), 10, 64); err == nil {
		limit = n
	}

	msgs, err := service.GetChatHistory(c.Request.Context(), uid, peer, group, before, limit)
	if err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, msgs)
}

// AuthMiddleware 鉴权中间件：校验 Authorization 头中的 Bearer Token，
// 解析成功后将用户 uid 写入上下文供后续处理使用。
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.JSON(http.StatusUnauthorized, &Response{Code: -1, Msg: "missing authorization header"})
			c.Abort()
			return
		}

		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" { // 必须为 "Bearer <token>" 格式
			c.JSON(http.StatusUnauthorized, &Response{Code: -1, Msg: "invalid authorization format"})
			c.Abort()
			return
		}

		claims, err := utils.ParseToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, &Response{Code: -1, Msg: "invalid token"})
			c.Abort()
			return
		}

		c.Set("uid", claims.Uid) // 注入 uid，后续 handler 通过 c.GetString("uid") 读取
		c.Next()
	}
}
