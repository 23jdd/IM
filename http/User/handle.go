package User

import (
	"IM/service"
	"IM/utils"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data,omitempty"`
}

func ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, &Response{Code: 0, Msg: "ok", Data: data})
}

func fail(c *gin.Context, code int, msg string) {
	c.JSON(http.StatusOK, &Response{Code: code, Msg: msg})
}

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

func GetProfile(c *gin.Context) {
	uid := c.GetString("uid")

	resp, err := service.GetProfile(c.Request.Context(), uid)
	if err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, resp)
}

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

func GetFriends(c *gin.Context) {
	uid := c.GetString("uid")

	resp, err := service.GetFriendList(c.Request.Context(), uid)
	if err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, resp)
}

func GetConversations(c *gin.Context) {
	uid := c.GetString("uid")

	resp, err := service.GetConversations(c.Request.Context(), uid)
	if err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, resp)
}

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

func GetMyGroups(c *gin.Context) {
	uid := c.GetString("uid")
	resp, err := service.GetUserGroupList(c.Request.Context(), uid)
	if err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, resp)
}

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

func JoinGroup(c *gin.Context) {
	uid := c.GetString("uid")
	var req struct {
		GroupId string `json:"group_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, -1, err.Error())
		return
	}
	if err := service.JoinGroup(c.Request.Context(), req.GroupId, uid); err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, nil)
}

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

func parseDataURL(s string) (contentType, data string) {
	contentType = "image/jpeg"
	idx := strings.Index(s, ",")
	if idx < 0 {
		return contentType, s
	}
	meta := s[:idx]
	data = s[idx+1:]
	if strings.HasPrefix(meta, "data:") {
		meta = meta[len("data:"):]
		if semi := strings.Index(meta, ";"); semi >= 0 {
			meta = meta[:semi]
		}
		if meta != "" {
			contentType = meta
		}
	}
	return contentType, data
}

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
			continue
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

func GetTimeline(c *gin.Context) {
	uid := c.GetString("uid")
	resp, err := service.GetTimeline(c.Request.Context(), uid)
	if err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, resp)
}

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

func GetFriendRequests(c *gin.Context) {
	uid := c.GetString("uid")
	resp, err := service.GetFriendRequests(c.Request.Context(), uid)
	if err != nil {
		fail(c, -1, err.Error())
		return
	}
	ok(c, resp)
}

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

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.JSON(http.StatusUnauthorized, &Response{Code: -1, Msg: "missing authorization header"})
			c.Abort()
			return
		}

		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
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

		c.Set("uid", claims.Uid)
		c.Next()
	}
}
