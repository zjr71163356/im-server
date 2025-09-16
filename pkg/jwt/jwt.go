package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims JWT 自定义声明
type Claims struct {
	UID uint64 `json:"uid"` // 用户ID
	DID uint64 `json:"did"` // 设备ID
	jwt.RegisteredClaims
}

// GenerateJWT 生成 JWT token
func GenerateJWT(uid, did uint64, secret []byte, ttl time.Duration, iss, aud string) (string, error) {
	now := time.Now()
	claims := Claims{
		UID: uid,
		DID: did,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    iss,
			Audience:  jwt.ClaimStrings{aud},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// ParseJWT 解析并验证 JWT token
func ParseJWT(tokenStr string, secret []byte, iss, aud string) (uid, did uint64, err error) {
	tok, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		// 验证签名算法，防止 alg 攻击
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithLeeway(30*time.Second),
		jwt.WithIssuer(iss),
		jwt.WithAudience(aud),
	)
	if err != nil {
		return 0, 0, err
	}
	if !tok.Valid {
		return 0, 0, errors.New("invalid token")
	}
	claims, ok := tok.Claims.(*Claims)
	if !ok {
		return 0, 0, errors.New("invalid claims type")
	}
	return claims.UID, claims.DID, nil
}
