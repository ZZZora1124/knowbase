// Package errors 定义恢复后的业务错误契约。数值为重建约定，非原始错误码表。
package errors

import "fmt"

type ErrorCode int

const (
	ErrCodeSuccess                    ErrorCode = 0
	ErrCodeInvalidParam               ErrorCode = 400
	ErrCodeInvalidKnowledgebaseName   ErrorCode = 40001
	ErrCodeInvalidVisibility          ErrorCode = 40002
	ErrCodeInvalidKnowledgeType       ErrorCode = 40003
	ErrCodeFileTooLarge               ErrorCode = 40004
	ErrCodeUnauthorized               ErrorCode = 401
	ErrCodeInvalidToken               ErrorCode = 40101
	ErrCodeTokenExpired               ErrorCode = 40102
	ErrCodeForbidden                  ErrorCode = 403
	ErrCodeNoPermission               ErrorCode = 40301
	ErrCodeNotKnowledgebaseMember     ErrorCode = 40302
	ErrCodeNotKnowledgebaseOwner      ErrorCode = 40303
	ErrCodeNotFound                   ErrorCode = 404
	ErrCodeProjectNotFound            ErrorCode = 40401
	ErrCodeKnowledgeNotFound          ErrorCode = 40402
	ErrCodeKnowledgebaseNotFound      ErrorCode = 40403
	ErrCodeUserNotFound               ErrorCode = 40404
	ErrCodeDepartmentNotFound         ErrorCode = 40405
	ErrCodeEmailGroupNotFound         ErrorCode = 40406
	ErrCodeChatGroupNotFound          ErrorCode = 40407
	ErrCodeAlreadyExists              ErrorCode = 409
	ErrCodeProjectAlreadyExists       ErrorCode = 40901
	ErrCodeKnowledgeAlreadyExists     ErrorCode = 40902
	ErrCodeKnowledgebaseAlreadyExists ErrorCode = 40903
	ErrCodeInternalError              ErrorCode = 500
	ErrCodeUploadFailed               ErrorCode = 50001
	ErrCodeInfoFlowAPIError           ErrorCode = 50201
)

type AppError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Cause   error     `json:"-"`
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *AppError) Unwrap() error { return e.Cause }

func New(code ErrorCode, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

func Wrap(code ErrorCode, message string, cause error) *AppError {
	return &AppError{Code: code, Message: message, Cause: cause}
}

var (
	ErrInvalidParam               = New(ErrCodeInvalidParam, "参数不合法")
	ErrInvalidRequest             = New(ErrCodeInvalidParam, "请求不合法")
	ErrInvalidKnowledgebaseName   = New(ErrCodeInvalidKnowledgebaseName, "知识本名称不合法")
	ErrInvalidVisibility          = New(ErrCodeInvalidVisibility, "知识本可见性不合法")
	ErrInvalidMemberType          = New(ErrCodeInvalidParam, "成员类型不合法")
	ErrInvalidRoleType            = New(ErrCodeInvalidParam, "成员角色不合法")
	ErrInvalidTagName             = New(ErrCodeInvalidParam, "标签名称不合法")
	ErrDuplicateMemberInRequest   = New(ErrCodeInvalidParam, "请求中包含重复成员")
	ErrUnauthorized               = New(ErrCodeUnauthorized, "未认证")
	ErrInvalidToken               = New(ErrCodeInvalidToken, "认证令牌无效")
	ErrTokenExpired               = New(ErrCodeTokenExpired, "认证令牌已过期")
	ErrNoPermission               = New(ErrCodeNoPermission, "没有操作权限")
	ErrNotKnowledgebaseMember     = New(ErrCodeNotKnowledgebaseMember, "不是知识本成员")
	ErrNoPermissionToManageMember = New(ErrCodeNoPermission, "没有管理成员的权限")
	ErrNoPermissionToManageTag    = New(ErrCodeNoPermission, "没有管理标签的权限")
	ErrNotGroupMember             = New(ErrCodeNoPermission, "不是群成员")
	ErrNotInChatGroup             = New(ErrCodeNoPermission, "操作者不在该群中")
	ErrCannotManagePersonalKB     = New(ErrCodeNoPermission, "不能管理个人知识本成员")
	ErrCannotModifyOwnerRole      = New(ErrCodeNoPermission, "不能修改知识本所有者角色")
	ErrCannotRemoveOwner          = New(ErrCodeNoPermission, "不能移除知识本所有者")
	ErrCannotRemoveLastAdmin      = New(ErrCodeInvalidParam, "不能移除最后一个管理员")
	ErrKnowledgebaseNotFound      = New(ErrCodeKnowledgebaseNotFound, "知识本不存在")
	ErrKnowledgeNotFound          = New(ErrCodeKnowledgeNotFound, "知识不存在")
	ErrMemberNotFound             = New(ErrCodeNotFound, "成员不存在")
	ErrTagNotFound                = New(ErrCodeNotFound, "标签不存在")
	ErrUserNotFound               = New(ErrCodeUserNotFound, "用户不存在")
	ErrDepartmentNotFound         = New(ErrCodeDepartmentNotFound, "部门不存在")
	ErrEmailGroupNotFound         = New(ErrCodeEmailGroupNotFound, "邮件组不存在")
	ErrKnowledgebaseAlreadyExists = New(ErrCodeKnowledgebaseAlreadyExists, "知识本已存在")
	ErrKnowledgeTitleDuplicate    = New(ErrCodeKnowledgeAlreadyExists, "目录下已存在同名知识")
	ErrReadmeAlreadyExists        = New(ErrCodeKnowledgeAlreadyExists, "知识本已存在README")
	ErrMemberAlreadyExists        = New(ErrCodeAlreadyExists, "成员已存在")
	ErrTagAlreadyExists           = New(ErrCodeAlreadyExists, "标签已存在")
	ErrInternalError              = New(ErrCodeInternalError, "内部服务错误")
	ErrUUAPClientInit             = New(ErrCodeInternalError, "UUAP客户端初始化失败")
)
