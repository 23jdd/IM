package Tests

import (
	"IM/utils"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// #5 JWT 安全：拒绝 alg=none、拒绝非 HS256 的 HMAC、密钥可配置。

func TestParseTokenRejectsNoneAlg(t *testing.T) {
	claims := jwt.MapClaims{
		"uid": "evil",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	s, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign none token: %v", err)
	}
	if _, err := utils.ParseToken(s); err == nil {
		t.Fatal("ParseToken must reject alg=none tokens")
	}
}

func TestParseTokenRejectsWrongHMACAlg(t *testing.T) {
	defer utils.SetJWTSecret("imSystem-secret")
	utils.SetJWTSecret("known-secret")

	// 同密钥但使用 HS384 —— ParseToken 仅允许 HS256，应拒绝。
	claims := &utils.Claims{
		Uid: "u",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS384, claims)
	s, err := tok.SignedString([]byte("known-secret"))
	if err != nil {
		t.Fatalf("sign hs384 token: %v", err)
	}
	if _, err := utils.ParseToken(s); err == nil {
		t.Fatal("ParseToken must reject HS384 tokens")
	}
}

func TestSetJWTSecretAffectsValidation(t *testing.T) {
	defer utils.SetJWTSecret("imSystem-secret")

	utils.SetJWTSecret("secretA")
	token, err := utils.GenerateToken("user-x", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	claims, err := utils.ParseToken(token)
	if err != nil || claims.Uid != "user-x" {
		t.Fatalf("expected valid parse, got claims=%v err=%v", claims, err)
	}

	utils.SetJWTSecret("secretB")
	if _, err := utils.ParseToken(token); err == nil {
		t.Fatal("token must be invalid after secret change")
	}
}

func TestSetJWTSecretIgnoresEmpty(t *testing.T) {
	defer utils.SetJWTSecret("imSystem-secret")

	utils.SetJWTSecret("keepme")
	token, _ := utils.GenerateToken("u", time.Now().Add(time.Hour))

	utils.SetJWTSecret("") // 空字符串应被忽略，保留 keepme
	if _, err := utils.ParseToken(token); err != nil {
		t.Fatalf("empty SetJWTSecret should not change secret: %v", err)
	}
}
