// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
)

// Response 统一响应结构
type Response struct {
	Code    errors.ErrorCode `json:"code"`
	Message string           `json:"message"`
	Data    interface{}      `json:"data,omitempty"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    errors.ErrCodeSuccess,
		Message: "success",
		Data:    data,
	})
}

// SuccessSimple 简单格式成功响应
func SuccessSimple(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, data)
}

// Error 错误响应
func Error(c *gin.Context, err error) {
	appErr, ok := err.(*errors.AppError)
	if !ok {
		// 如果不是AppError，返回内部错误
		c.JSON(http.StatusInternalServerError, Response{
			Code:    errors.ErrCodeInternalError,
			Message: err.Error(),
		})
		return
	}

	// 根据错误码确定HTTP状态码
	statusCode := getHTTPStatusCode(appErr.Code)
	c.JSON(statusCode, Response{
		Code:    appErr.Code,
		Message: appErr.Message,
	})
}

// ErrorWithMessage 带自定义消息的错误响应
func ErrorWithMessage(c *gin.Context, code errors.ErrorCode, message string) {
	statusCode := getHTTPStatusCode(code)
	c.JSON(statusCode, Response{
		Code:    code,
		Message: message,
	})
}

// getHTTPStatusCode 根据错误码获取HTTP状态码
func getHTTPStatusCode(code errors.ErrorCode) int {
	switch code {
	case errors.ErrCodeSuccess:
		return http.StatusOK
	case errors.ErrCodeInvalidParam, errors.ErrCodeInvalidKnowledgebaseName, errors.ErrCodeInvalidVisibility, errors.ErrCodeInvalidKnowledgeType:
		return http.StatusBadRequest
	case errors.ErrCodeFileTooLarge:
		return http.StatusRequestEntityTooLarge
	case errors.ErrCodeUnauthorized, errors.ErrCodeInvalidToken, errors.ErrCodeTokenExpired:
		return http.StatusUnauthorized
	case errors.ErrCodeForbidden, errors.ErrCodeNoPermission, errors.ErrCodeNotKnowledgebaseMember, errors.ErrCodeNotKnowledgebaseOwner:
		return http.StatusForbidden
	case errors.ErrCodeNotFound, errors.ErrCodeProjectNotFound, errors.ErrCodeKnowledgeNotFound, errors.ErrCodeKnowledgebaseNotFound:
		return http.StatusNotFound
	case errors.ErrCodeUserNotFound, errors.ErrCodeDepartmentNotFound, errors.ErrCodeEmailGroupNotFound, errors.ErrCodeChatGroupNotFound:
		return http.StatusNotFound
	case errors.ErrCodeAlreadyExists, errors.ErrCodeProjectAlreadyExists, errors.ErrCodeKnowledgeAlreadyExists, errors.ErrCodeKnowledgebaseAlreadyExists:
		return http.StatusConflict
	case errors.ErrCodeInfoFlowAPIError:
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

// PageData 分页数据响应
type PageData struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
}

// SuccessWithPage 分页成功响应
func SuccessWithPage(c *gin.Context, list interface{}, total int64, page, pageSize int) {
	Success(c, PageData{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}
