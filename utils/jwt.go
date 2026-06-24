package utils

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// jwtSecret 默认从环境变量 IM_JWT_SECRET 读取，缺省回退到内置值；
// 可在启动时通过 SetJWTSecret 从配置覆盖。
var jwtSecret = func() []byte {
	if s := os.Getenv("IM_JWT_SECRET"); s != "" {
		return []byte(s)
	}
	return []byte("imSystem-secret")
}()

// SetJWTSecret 设置签名密钥（空字符串忽略，保留当前值）。
func SetJWTSecret(secret string) {
	if secret != "" {
		jwtSecret = []byte(secret)
	}
}

// Claims 自定义 JWT 载荷，包含用户 ID 和标准注册声明
type Claims struct {
	Uid string `json:"uid"`
	jwt.RegisteredClaims
}

// GenerateToken 为指定用户生成 HS256 签名的 JWT，expiresAt 为过期时间
func GenerateToken(uid string, expiresAt time.Time) (string, error) {
	claims := &Claims{
		Uid: uid,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ParseToken 解析并校验 JWT，返回其中的 Claims，校验失败时返回错误
func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		// 显式校验签名算法，防止 alg 混淆攻击（如 alg=none 或非 HMAC）。
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, jwt.ErrSignatureInvalid
}
