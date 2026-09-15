// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package middleware

import (
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/log"
	"net/http"

	"github.com/gin-gonic/gin"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/response"
)

// Recovery 错误恢复中间件
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 记录panic错误
				log.Logger.Errorf("Panic错误, err:%v, path:%s, method:%s",
					err,
					c.Request.URL.Path,
					c.Request.Method,
				)

				// 返回错误响应
				c.JSON(http.StatusInternalServerError, response.Response{
					Code:    errors.ErrCodeInternalError,
					Message: "服务器内部错误",
				})
				c.Abort()
			}
		}()

		c.Next()
	}
}
