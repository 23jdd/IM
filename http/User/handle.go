package User

import (
	"IM/service"
	"IM/utils"
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
