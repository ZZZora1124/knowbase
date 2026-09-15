// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package handler

import (
	"github.com/gin-gonic/gin"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/response"
)

// HealthCheck 健康检查
func HealthCheck(c *gin.Context) {
	response.Success(c, gin.H{
		"status": "ok",
		"service": "knowbase-server",
	})
}
