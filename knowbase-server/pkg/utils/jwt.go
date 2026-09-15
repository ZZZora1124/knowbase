// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package utils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
)

var (
	// JWTSecret JWT密钥（应该从配置文件读取）
	JWTSecret = []byte("your-secret-key-change-in-production")
	// TokenExpireDuration Token过期时间
	TokenExpireDuration = time.Hour * 24 * 7 // 7天
)

// Claims JWT Claims
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// GenerateToken 生成JWT Token
func GenerateToken(username string) (string, error) {
	claims := Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExpireDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "knowbase-server",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(JWTSecret)
	if err != nil {
		return "", errors.Wrap(errors.ErrCodeInternalError, "生成Token失败", err)
	}

	return tokenString, nil
}

// ParseToken 解析JWT Token
func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return JWTSecret, nil
	})

	if err != nil {
		if err == jwt.ErrTokenExpired {
			return nil, errors.ErrTokenExpired
		}
		return nil, errors.Wrap(errors.ErrCodeInvalidToken, "解析Token失败", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.ErrInvalidToken
}

// SetJWTSecret 设置JWT密钥
func SetJWTSecret(secret string) {
	JWTSecret = []byte(secret)
}
