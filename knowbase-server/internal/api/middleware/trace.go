// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/trace"
)

const (
	// TraceIDHeader HTTP请求头中的TraceID字段名
	TraceIDHeader = "x-baidu-int-request-id"
	// TraceIDKey gin.Context中存储TraceID的key
	TraceIDKey = "traceId"
)

// Trace TraceID中间件
// 为每个请求生成或提取TraceID，用于日志关联
func Trace() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 优先从请求头获取（支持链路追踪传递）
		traceID := c.GetHeader(TraceIDHeader)
		if traceID == "" || len(traceID) < 8 {
			traceID = trace.NewTraceID()
		}

		// 存入gin.Context, 取8位方便阅读
		c.Set(TraceIDKey, traceID[:8])

		// 设置响应头，便于前端/调用方关联
		c.Header(TraceIDHeader, traceID)

		c.Next()
	}
}

// GetTraceID 从gin.Context获取TraceID
func GetTraceID(c *gin.Context) string {
	if v, exists := c.Get(TraceIDKey); exists {
		if traceID, ok := v.(string); ok {
			return traceID
		}
	}
	return ""
}

// GetContext 从gin.Context创建带TraceID的context.Context
func GetContext(c *gin.Context) context.Context {
	traceID := GetTraceID(c)
	return trace.WithTraceID(c.Request.Context(), traceID)
}
