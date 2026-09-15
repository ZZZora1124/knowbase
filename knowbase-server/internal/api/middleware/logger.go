// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/log"
)

// Logger 日志中间件
// 注意：必须在Trace中间件之后注册，以获取TraceID
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		traceID := GetTraceID(c)

		// 处理请求
		c.Next()

		// 记录日志
		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method

		log.Logger.Infof("[%s] HTTP请求, method:%s, path:%s, query:%s, status:%d, ip:%s, latency:%v, user_agent:%s",
			traceID, method, path, query, statusCode, clientIP, latency, c.Request.UserAgent(),
		)
	}
}
