// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/config"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/cache"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/database"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/repository"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/log"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/response"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/types/constants"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/utils"
)

// openAPIClientCache OpenAPI客户端缓存结构（仅包含鉴权所需字段）
type openAPIClientCache struct {
	UUID       string     `json:"uuid"`
	ClientName string     `json:"clientName"`
	Owner      string     `json:"owner"`
	Status     int8       `json:"status"`
	IsAdmin    int8       `json:"isAdmin"`
	ExpiresAt  *time.Time `json:"expiresAt"`
	DeletedAt  *time.Time `json:"deletedAt"`
}

// isActive 检查缓存的客户端是否有效
func (c *openAPIClientCache) isActive() bool {
	if c.Status != 1 {
		return false
	}
	if c.ExpiresAt != nil && c.ExpiresAt.Before(time.Now()) {
		return false
	}
	if c.DeletedAt != nil {
		return false
	}
	return true
}

// AuthJWT JWT认证中间件
func AuthJWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Error(c, errors.ErrUnauthorized)
			c.Abort()
			return
		}

		// 检查格式是否为 "Bearer {token}"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Error(c, errors.ErrInvalidToken)
			c.Abort()
			return
		}

		token := parts[1]

		// 解析Token
		claims, err := utils.ParseToken(token)
		if err != nil {
			response.Error(c, err)
			c.Abort()
			return
		}

		// 将用户信息存入上下文
		c.Set(constants.AuthUsername, claims.Username)
		c.Set(constants.AuthType, constants.AuthTypeJWT)
		c.Next()
	}
}

// AuthCloud 云上百度认证中间件
// 给云上百度web端使用
func AuthCloud() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取Authorization header
		username := c.GetHeader("X-BAIDU-INT-USERNAME")
		if username == "" {
			response.Error(c, errors.ErrUnauthorized)
			c.Abort()
			return
		}

		// 将用户信息存入上下文
		c.Set(constants.AuthUsername, username)
		c.Set(constants.AuthType, constants.AuthTypeCloud)
		c.Next()
	}
}

// AuthHeader Header Key-Value认证中间件
// 支持两种认证方式：
// 1.x-auth-client-token 给平台单独使用，比如上报如流群消息
// 2. x-ac-Authorization 给对接统一鉴权的业务使用，比如查询如流群消息
func AuthHeader() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取两个header，如果都不存在，则报错
		authClientKey := c.GetHeader(constants.HeaderAuthClientKey)
		authACKey := c.GetHeader(constants.HeaderAuthACKey)
		if authClientKey == "" && authACKey == "" {
			response.ErrorWithMessage(c, errors.ErrCodeUnauthorized, "缺少认证Header")
			c.Abort()
			return
		}
		// 对应的key非空即可认为是有效的
		if authClientKey != "" {
			if !ValidateClientToken(c, authClientKey) {
				return
			}
		}
		if authACKey != "" {
			// 校验acToken
			username, plat, err := ValidateACToken(authACKey)
			if err != nil {
				response.ErrorWithMessage(c, errors.ErrCodeInvalidToken, err.Error())
				c.Abort()
				return
			}
			if username == "" {
				response.ErrorWithMessage(c, errors.ErrCodeInvalidToken, "ACToken校验失败")
				c.Abort()
				return
			}
			c.Set(constants.AuthUsername, username)
			c.Set(constants.AuthPlatform, plat)
		}

		// 标记认证方式
		c.Set(constants.AuthType, constants.AuthTypeHeader)
		c.Next()
	}
}

// GetUsername 从上下文获取用户名（仅JWT认证时有效）
func GetUsername(c *gin.Context) string {
	username, exists := c.Get(constants.AuthUsername)
	if !exists {
		return ""
	}
	return username.(string)
}

// GetAuthType 从上下文获取认证类型
func GetAuthType(c *gin.Context) string {
	authType, exists := c.Get(constants.AuthType)
	if !exists {
		return "none"
	}
	return authType.(string)
}

// GetClientIsAdmin 从上下文获取客户端Admin权限标识
func GetClientIsAdmin(c *gin.Context) bool {
	isAdmin, exists := c.Get(constants.AuthClientIsAdmin)
	if !exists {
		return false
	}
	return isAdmin.(int8) == 1
}

func ValidateACToken(token string) (string, string, error) {
	// 先尝试从Redis缓存中获取用户名
	cacher := cache.GetCache()
	var username string
	var plat string

	if cacher != nil {
		cachedUserPlat, err := cacher.Get(context.Background(), fmt.Sprintf("%s%s", constants.CacheKeyACUserPlat, token))
		if err == nil && cachedUserPlat != "" {
			strs := strings.Split(cachedUserPlat, "#")
			if len(strs) >= 2 {
				username = strs[0]
				plat = strs[1]
			}
			log.Logger.Debugf("Found username in cache: %s, token: %s", username, token)
		}
	}

	// 如果缓存中没有找到，调用第三方认证接口
	if username == "" {
		log.Logger.Debugf("Username not found in cache, calling authentication service, token: %s, requestId: %s", token)

		// 获取认证配置
		authConfig, err := config.GetAuthenticationConfig()
		if err != nil {
			log.Logger.Errorf("Failed to get authentication config: %v", err)
			return "", "", fmt.Errorf("failed to get authentication config: %v", err)
		}

		// 调用第三方认证接口
		username, plat, err = utils.ValidateAccessToken(authConfig.URL, authConfig.ClientId, authConfig.ClientSecret, authConfig.RetryCount, authConfig.Timeout, token)
		if err != nil {
			log.Logger.Errorf("Failed to validate access token: %v, token: %s", err, token)
			return "", "", fmt.Errorf("failed to validate access token: %v", err)
		}

		// 将用户名存入Redis缓存
		if cacher != nil {
			err = cacher.Set(context.Background(), fmt.Sprintf("%s%s", constants.CacheKeyACUserPlat, token), fmt.Sprintf("%s#%s", username, plat), 24*time.Hour)
			if err != nil {
				log.Logger.Errorf("Failed to cache username: %v, token: %s, username: %s", err, token, username)
				// 缓存失败不影响主要流程，继续执行
			} else {
				log.Logger.Debugf("Successfully cached username: %s, token: %s", username, token)
			}
		}
	}
	return username, plat, nil
}

// AuthOpenAPI OpenAPI认证中间件
// 认证头: X-Auth-Client-Token: <client_token>
// 用于业务方系统调用 OpenAPI 接口，同时兼容actoken认证
func AuthOpenAPI() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 兼容actoken认证
		authACKey := c.GetHeader(constants.HeaderAuthACKey)
		if authACKey != "" {
			// 校验acToken
			username, plat, err := ValidateACToken(authACKey)
			if err != nil {
				response.ErrorWithMessage(c, errors.ErrCodeInvalidToken, err.Error())
				c.Abort()
				return
			}
			if username == "" {
				response.ErrorWithMessage(c, errors.ErrCodeInvalidToken, "ACToken校验失败")
				c.Abort()
				return
			}
			c.Set(constants.AuthUsername, username)
			c.Set(constants.AuthPlatform, plat)
			c.Set(constants.AuthType, constants.AuthTypeHeader)
			c.Next()
			return
		}

		// 校验Client Token
		token := c.GetHeader(constants.HeaderOpenAPIClientToken)
		if token == "" {
			response.ErrorWithMessage(c, errors.ErrCodeUnauthorized, "缺少认证Header")
			c.Abort()
			return
		}

		if !ValidateClientToken(c, token) {
			return
		}
		c.Next()
	}
}

// ValidateClientToken 校验 Client Token，校验通过后将客户端信息写入上下文。
// 返回 true 表示校验成功；返回 false 表示校验失败（已调用 response.ErrorWithMessage 和 c.Abort()）。
func ValidateClientToken(c *gin.Context, token string) bool {
	cacher := cache.GetCache()
	cacheKey := fmt.Sprintf("%s%s", constants.CacheKeyOpenAPIClient, token)
	prefix := tokenPrefix(token)

	if cacher != nil {
		cachedVal, err := cacher.Get(context.Background(), cacheKey)
		if err == nil && cachedVal != "" {
			var cachedClient openAPIClientCache
			if jsonErr := json.Unmarshal([]byte(cachedVal), &cachedClient); jsonErr == nil {
				if !cachedClient.isActive() {
					log.Logger.WarnfCtx(c, "OpenAPI client is inactive (from cache): clientName=%s, uuid=%s, token=%s",
						cachedClient.ClientName, cachedClient.UUID, prefix)
					response.ErrorWithMessage(c, errors.ErrCodeUnauthorized, "客户端已禁用或已过期")
					c.Abort()
					return false
				}
				log.Logger.InfofCtx(c, "OpenAPI client authenticated (from cache): clientName=%s, uuid=%s, owner=%s, token=%s",
					cachedClient.ClientName, cachedClient.UUID, cachedClient.Owner, prefix)
				// 异步更新最后使用时间
				/*
					go func(uuid string) {
						ormer := database.GetOrm()
						repo := repository.NewOpenAPIClientRepository(ormer)
						_ = repo.UpdateLastUsed(uuid)
					}(cachedClient.UUID) */
				c.Set(constants.AuthUsername, cachedClient.Owner)
				c.Set(constants.AuthType, constants.AuthTypeOpenAPI)
				c.Set(constants.AuthClientName, cachedClient.ClientName)
				c.Set(constants.AuthClientUUID, cachedClient.UUID)
				c.Set(constants.AuthClientIsAdmin, cachedClient.IsAdmin)
				return true
			}
			log.Logger.WarnfCtx(c, "OpenAPI client cache unmarshal failed, fallback to DB, token=%s", prefix)
		}
	}

	// 缓存未命中，从数据库验证 Token
	log.Logger.InfofCtx(c, "OpenAPI client not found in cache, querying DB, token=%s", prefix)
	ormer := database.GetOrm()
	repo := repository.NewOpenAPIClientRepository(ormer)
	client, err := repo.FindByToken(token)
	if err != nil {
		log.Logger.ErrorfCtx(c, "OpenAPI token validation failed: %v, token=%s", err, prefix)
		response.ErrorWithMessage(c, errors.ErrCodeInvalidToken, "无效的Client Token")
		c.Abort()
		return false
	}

	if client == nil {
		log.Logger.WarnfCtx(c, "OpenAPI client not found for token=%s", prefix)
		response.ErrorWithMessage(c, errors.ErrCodeInvalidToken, "无效的Client Token")
		c.Abort()
		return false
	}

	if !client.IsActive() {
		log.Logger.WarnfCtx(c, "OpenAPI client is inactive: clientName=%s, uuid=%s, token=%s",
			client.ClientName, client.UUID, prefix)
		response.ErrorWithMessage(c, errors.ErrCodeUnauthorized, "客户端已禁用或已过期")
		c.Abort()
		return false
	}

	// 将客户端信息写入缓存（TTL=24h）
	if cacher != nil {
		cachedClient := openAPIClientCache{
			UUID:       client.UUID,
			ClientName: client.ClientName,
			Owner:      client.Owner,
			Status:     client.Status,
			IsAdmin:    client.IsAdmin,
			ExpiresAt:  client.ExpiresAt,
			DeletedAt:  client.DeletedAt,
		}
		if jsonBytes, jsonErr := json.Marshal(cachedClient); jsonErr == nil {
			if setErr := cacher.Set(context.Background(), cacheKey, string(jsonBytes), 24*time.Hour); setErr != nil {
				log.Logger.ErrorfCtx(c, "Failed to cache OpenAPI client: %v, clientName=%s, token=%s", setErr, client.ClientName, prefix)
			} else {
				log.Logger.Debugf("Successfully cached OpenAPI client: clientName=%s, token=%s", client.ClientName, prefix)
			}
		}
	}

	// 更新最后使用时间（异步）
	go func() {
		_ = repo.UpdateLastUsed(client.UUID)
	}()

	log.Logger.InfofCtx(c, "OpenAPI client authenticated: clientName=%s, uuid=%s, owner=%s, token=%s",
		client.ClientName, client.UUID, client.Owner, prefix)

	c.Set(constants.AuthUsername, client.Owner)
	c.Set(constants.AuthType, constants.AuthTypeOpenAPI)
	c.Set(constants.AuthClientName, client.ClientName)
	c.Set(constants.AuthClientUUID, client.UUID)
	c.Set(constants.AuthClientIsAdmin, client.IsAdmin)
	return true
}

// tokenPrefix 返回 token 的前8位用于日志（脱敏）
func tokenPrefix(token string) string {
	if len(token) <= 8 {
		return token
	}
	return token[:8] + "..."
}
