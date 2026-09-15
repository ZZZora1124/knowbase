// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package uuap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/config"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/cache"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/log"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/types/constants"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/utils"
)

// UUAP缓存过期时间常量
const (
	// 邮件组成员缓存过期时间（2小时）
	EmailGroupMemberCacheTTL = 2 * time.Hour
	// 部门成员缓存过期时间（2小时）
	DepartmentMemberCacheTTL = 2 * time.Hour
)

// UUAPClient UUAP API客户端
// 基于UUAP（统一身份认证平台）提供用户、邮件组、部门等API
type UUAPClient struct {
	baseURL    string
	appKey     string
	secretKey  string
	retryCount int
	timeout    time.Duration
}

var (
	defaultClient     *UUAPClient
	defaultClientOnce sync.Once
	initError         error
)

// GetClient 获取全局UUAP客户端单例
// 基于 sync.Once 保证只初始化一次，配置不存在时返回错误
func GetClient() (*UUAPClient, error) {
	defaultClientOnce.Do(func() {
		cfg, err := config.GetUUAPConfig()
		if err != nil {
			initError = fmt.Errorf("UUAP配置未找到: %w", err)
			log.Logger.Errorf("UUAP配置未找到，服务启动失败: %v", err)
			return
		}

		defaultClient = NewUUAPClient(cfg)
		log.Logger.Info("UUAP客户端初始化成功")
	})

	if initError != nil {
		return nil, initError
	}

	return defaultClient, nil
}

// NewUUAPClient 创建UUAP客户端
func NewUUAPClient(cfg *config.UUAPConfig) *UUAPClient {
	return &UUAPClient{
		baseURL:    cfg.URL,
		appKey:     cfg.ClientId,
		secretKey:  cfg.ClientSecret,
		retryCount: cfg.RetryCount,
		timeout:    time.Duration(cfg.Timeout) * time.Second,
	}
}

// buildUUAPOAuthParams 构建UUAP鉴权参数
func (c *UUAPClient) buildUUAPOAuthParams(params map[string]interface{}) map[string]string {
	// 1. 添加 appKey
	result := make(map[string]string)
	result["appKey"] = c.appKey

	// 2. 添加 timestamp（当前时间戳，单位：秒）
	timestamp := time.Now().Unix()
	result["timestamp"] = fmt.Sprintf("%d", timestamp)

	// 3. 添加 sRandom（随机盐，使用UUID保证唯一）
	sRandom := uuid.New().String()
	result["sRandom"] = sRandom

	// 4. 将业务参数转换为字符串并添加到结果中
	for k, v := range params {
		result[k] = fmt.Sprintf("%v", v)
	}

	// 5. 将参数名按字典顺序排序
	keys := make([]string, 0, len(result))
	for k := range result {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 6. 将参数的Value值按排序顺序拼成字符串
	var values []string
	for _, k := range keys {
		values = append(values, result[k])
	}
	beforeSign := strings.Join(values, "")

	// 7. 将 secretKey 追加到字符串最后
	beforeSign = beforeSign + c.secretKey

	// 8. 进行 SHA-256 加密（转换为小写的16进制数据）
	hash := sha256.Sum256([]byte(beforeSign))
	sign := hex.EncodeToString(hash[:])

	// 9. 将 sign 追加到参数中
	result["sign"] = sign

	return result
}

// UUAPResponse UUAP API通用响应结构
type UUAPResponse[T any] struct {
	Code   int    `json:"code"`
	Desc   string `json:"desc"`
	Result T      `json:"result"`
}

// callAPI 统一的API调用方法
// endpoint: API路径（如 /restful/v1/userInfo/getUserByUsername）
// businessParams: 业务参数
// response: 响应结构体指针
func (c *UUAPClient) callAPI(ctx context.Context, endpoint string, businessParams map[string]interface{}, response interface{}) error {
	// 添加UUAP鉴权参数
	authParams := c.buildUUAPOAuthParams(businessParams)

	// 合并业务参数和鉴权参数
	allParams := make(map[string]interface{})
	for k, v := range businessParams {
		allParams[k] = v
	}
	allParams["appKey"] = authParams["appKey"]
	allParams["timestamp"] = authParams["timestamp"]
	allParams["sRandom"] = authParams["sRandom"]
	allParams["sign"] = authParams["sign"]

	// 将参数序列化为JSON请求体
	body, err := json.Marshal(allParams)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	request := &utils.Request{
		URL:         fmt.Sprintf("%s%s", c.baseURL, endpoint),
		Method:      "POST",
		ContentType: utils.ContentTypeJSON,
		Body:        body,
		Timeout:     c.timeout,
		Retry:       c.retryCount,
	}

	_, statusCode, err := utils.SendRequest(request, response)
	if err != nil {
		return fmt.Errorf("failed to call uuap api: %w", err)
	}

	if statusCode != 200 {
		return fmt.Errorf("uuap api returned status: %d", statusCode)
	}

	return nil
}

// checkUUAPResponse 检查UUAP响应码
func checkUUAPResponse(code int, desc string) error {
	if code != 200 {
		return fmt.Errorf("uuap api error: code=%d, desc=%s", code, desc)
	}
	return nil
}

// GetUserEmailGroupNames 根据用户名获取其所在邮件组的名称列表
// 接口: POST /restful/v1/emailGroupInfo/getAllEmailGroupNamesByUsername
// 参数: username - 用户名
// 返回: 邮件组名称列表
func (c *UUAPClient) GetUserEmailGroupNames(ctx context.Context, username string) ([]string, error) {
	businessParams := map[string]interface{}{
		"username": username,
	}

	var response UUAPResponse[[]string]
	if err := c.callAPI(ctx, "/restful/v1/emailGroupInfo/getAllEmailGroupNamesByUsername", businessParams, &response); err != nil {
		return nil, err
	}

	if err := checkUUAPResponse(response.Code, response.Desc); err != nil {
		return nil, err
	}

	log.Logger.Debugf("User %s belongs to %d email groups (by name)", username, len(response.Result))
	return response.Result, nil
}

// GetUserEmailGroupAddresses 根据用户名获取其所在邮件组的地址列表
// 接口: POST /restful/v1/emailGroupInfo/getAllEmailGroupsByUsername
// 参数: username - 用户名
// 返回: 邮件组地址列表
func (c *UUAPClient) GetUserEmailGroupAddresses(ctx context.Context, username string) ([]string, error) {
	businessParams := map[string]interface{}{
		"username": username,
	}

	var response UUAPResponse[[]string]
	if err := c.callAPI(ctx, "/restful/v1/emailGroupInfo/getAllEmailGroupsByUsername", businessParams, &response); err != nil {
		return nil, err
	}

	if err := checkUUAPResponse(response.Code, response.Desc); err != nil {
		return nil, err
	}

	log.Logger.Debugf("User %s belongs to %d email groups (by name)", username, len(response.Result))
	return response.Result, nil
}

// IsUserInEmailGroup 根据用户名判断其是否属于某个邮件组
// 接口: POST /restful/v1/emailGroupInfo/hasMemberByUsername
// 参数: username - 用户名, emailGroupAddress - 邮件组email地址
// 返回: true-用户在邮件组中, false-用户不在邮件组中
func (c *UUAPClient) IsUserInEmailGroup(ctx context.Context, username, emailGroupAddress string) (bool, error) {
	businessParams := map[string]interface{}{
		"username":          username,
		"emailGroupAddress": emailGroupAddress,
	}

	var response UUAPResponse[bool]
	if err := c.callAPI(ctx, "/restful/v1/emailGroupInfo/hasMemberByUsername", businessParams, &response); err != nil {
		return false, err
	}

	if err := checkUUAPResponse(response.Code, response.Desc); err != nil {
		return false, err
	}

	log.Logger.Debugf("User %s is in email group %s: %v", username, emailGroupAddress, response.Result)
	return response.Result, nil
}

// EmailGroupDTO 邮件组信息
type EmailGroupDTO struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Info  string `json:"info"`
	Type  string `json:"type"`
}

// GetEmailGroupByAddress 根据邮件组地址获取其详细信息
// 接口: POST /restful/v1/emailGroupInfo/getEmailGroupByAddress
// 参数: emailGroupAddress - 邮件组email地址
// 返回: 邮件组信息，如果邮件组不存在则返回nil
func (c *UUAPClient) GetEmailGroupByAddress(ctx context.Context, emailGroupAddress string) (*EmailGroupDTO, error) {
	businessParams := map[string]interface{}{
		"emailGroupAddress": emailGroupAddress,
	}

	var response UUAPResponse[*EmailGroupDTO]
	if err := c.callAPI(ctx, "/restful/v1/emailGroupInfo/getEmailGroupByAddress", businessParams, &response); err != nil {
		return nil, err
	}

	if err := checkUUAPResponse(response.Code, response.Desc); err != nil {
		return nil, err
	}

	if response.Result == nil {
		log.Logger.Debugf("Email group %s not found", emailGroupAddress)
		return nil, nil
	}

	log.Logger.Debugf("Email group %s found: %+v", emailGroupAddress, response.Result)
	return response.Result, nil
}

// UserDTO 用户信息
type UserDTO struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	Name        string `json:"name"`
	HiAccountId int64  `json:"hiAccountId"`
}

// GetUserByUsername 根据用户名获取用户信息
// 接口: POST /restful/v1/userInfo/getUserByUsername
// 参数: username - 用户名
// 返回: 用户信息，如果用户不存在则返回nil
func (c *UUAPClient) GetUserByUsername(ctx context.Context, username string) (*UserDTO, error) {
	businessParams := map[string]interface{}{
		"username": username,
	}

	var response UUAPResponse[*UserDTO]
	if err := c.callAPI(ctx, "/restful/v1/userInfo/getUserByUsername", businessParams, &response); err != nil {
		return nil, err
	}

	if err := checkUUAPResponse(response.Code, response.Desc); err != nil {
		return nil, err
	}

	if response.Result == nil {
		log.Logger.Debugf("User %s not found", username)
		return nil, nil
	}

	log.Logger.Debugf("User %s found: %+v", username, response.Result)
	return response.Result, nil
}

// DepartmentDTO 部门信息
type DepartmentDTO struct {
	ID                int64  `json:"id"`
	Code              string `json:"code"`
	Name              string `json:"name"`
	Type              string `json:"type"`
	EnglishName       string `json:"englishName"`
	Abbreviation      string `json:"abbreviation"`
	ParentDepId       int64  `json:"parentDepId"`
	ParentName        string `json:"parentName"`
	BusinessGroupId   int64  `json:"businessGroupId"`
	BusinessGroupName string `json:"businessGroupName"`
	// 组织层级信息
	Org0Id    int64  `json:"org0Id"`
	Org0Name  string `json:"org0Name"`
	Org1Id    int64  `json:"org1Id"`
	Org1Name  string `json:"org1Name"`
	Org2Id    int64  `json:"org2Id"`
	Org2Name  string `json:"org2Name"`
	Org3Id    int64  `json:"org3Id"`
	Org3Code  string `json:"org3Code"`
	Org3Name  string `json:"org3Name"`
	OrgLevel  int    `json:"orgLevel"`
	OrgIdList string `json:"orgIdList"`
	GudtType  string `json:"gudtType"`
}

// GetAllDepartmentKeys 获取用户所属的所有部门标识列表（包含各级上级部门）
// 返回格式: []string{"部门ID:部门名称", ...}
// 例如: ["2319:百度", "29563:技术中台群组", "30354:企业效率平台", "1639:企业IT平台部"]
func (d *DepartmentDTO) GetAllDepartmentKeys() []string {
	if d == nil {
		return nil
	}

	var keys []string

	// 按层级添加部门信息
	if d.Org0Id > 0 && d.Org0Name != "" {
		keys = append(keys, fmt.Sprintf("%d:%s", d.Org0Id, d.Org0Name))
	}
	if d.Org1Id > 0 && d.Org1Name != "" {
		keys = append(keys, fmt.Sprintf("%d:%s", d.Org1Id, d.Org1Name))
	}
	if d.Org2Id > 0 && d.Org2Name != "" {
		keys = append(keys, fmt.Sprintf("%d:%s", d.Org2Id, d.Org2Name))
	}
	if d.Org3Id > 0 && d.Org3Name != "" {
		keys = append(keys, fmt.Sprintf("%d:%s", d.Org3Id, d.Org3Name))
	}

	return keys
}

// GetUserBusinessDeptOrg 根据用户名查询用户所属部门
// 接口: POST /restful/v1/departmentInfo/getUserBusinessDeptOrg
// 参数: username - 用户名
// 返回: 用户所属部门信息，如果用户没有部门则返回nil
func (c *UUAPClient) GetUserBusinessDeptOrg(ctx context.Context, username string) (*DepartmentDTO, error) {
	businessParams := map[string]interface{}{
		"username": username,
	}

	var response UUAPResponse[*DepartmentDTO]
	if err := c.callAPI(ctx, "/restful/v1/departmentInfo/getUserBusinessDeptOrg", businessParams, &response); err != nil {
		return nil, err
	}

	if err := checkUUAPResponse(response.Code, response.Desc); err != nil {
		return nil, err
	}

	if response.Result == nil {
		log.Logger.Debugf("User %s has no department", username)
		return nil, nil
	}

	log.Logger.Debugf("User %s belongs to department: %+v", username, response.Result)
	return response.Result, nil
}

// GetDepartmentByIdAndName 根据部门ID和名称获取部门信息
// 接口: POST /restful/v1/departmentInfo/getDepartmentByIdAndName
// 参数: departmentId - 部门ID, departmentName - 部门名称
// 返回: 部门信息，如果部门不存在则返回nil
func (c *UUAPClient) GetDepartmentByIdAndName(ctx context.Context, departmentId int64, departmentName string) (*DepartmentDTO, error) {
	businessParams := map[string]interface{}{
		"departmentId":   departmentId,
		"departmentName": departmentName,
	}

	var response UUAPResponse[*DepartmentDTO]
	if err := c.callAPI(ctx, "/restful/v1/departmentInfo/getDepartmentByIdAndName", businessParams, &response); err != nil {
		return nil, err
	}

	if err := checkUUAPResponse(response.Code, response.Desc); err != nil {
		return nil, err
	}

	if response.Result == nil {
		log.Logger.Debugf("Department %d:%s not found", departmentId, departmentName)
		return nil, nil
	}

	log.Logger.Debugf("Department %d:%s found: %+v", departmentId, departmentName, response.Result)
	return response.Result, nil
}

// GetUserNamesByDepartment 根据部门ID和名称获取成员用户名列表
// 接口: POST /restful/v1/departmentInfo/getUserNamesByDepartmentIdAndName
// 参数: departmentId - 部门ID, departmentName - 部门名称
// 返回: 用户名列表
func (c *UUAPClient) GetUserNamesByDepartment(ctx context.Context, departmentId int64, departmentName string) ([]string, error) {
	businessParams := map[string]interface{}{
		"departmentId":   departmentId,
		"departmentName": departmentName,
	}

	var response UUAPResponse[[]string]
	if err := c.callAPI(ctx, "/restful/v1/departmentInfo/getUserNamesByDepartmentIdAndName", businessParams, &response); err != nil {
		return nil, err
	}

	if err := checkUUAPResponse(response.Code, response.Desc); err != nil {
		return nil, err
	}

	log.Logger.Debugf("Department %d:%s has %d members", departmentId, departmentName, len(response.Result))
	return response.Result, nil
}

// GetUserNamesByEmailGroup 根据邮件组地址获取成员用户名列表
// 接口: POST /restful/v1/emailGroupInfo/getUserNamesByEmailGroupAddress
// 参数: emailGroupAddress - 邮件组地址
// 返回: 用户名列表
func (c *UUAPClient) GetUserNamesByEmailGroup(ctx context.Context, emailGroupAddress string) ([]string, error) {
	businessParams := map[string]interface{}{
		"emailGroupAddress": emailGroupAddress,
	}

	var response UUAPResponse[[]string]
	if err := c.callAPI(ctx, "/restful/v1/emailGroupInfo/getUserNamesByEmailGroupAddress", businessParams, &response); err != nil {
		return nil, err
	}

	if err := checkUUAPResponse(response.Code, response.Desc); err != nil {
		return nil, err
	}

	log.Logger.Debugf("Email group %s has %d members", emailGroupAddress, len(response.Result))
	return response.Result, nil
}

// GetUserNamesByEmailGroupWithCache 带缓存的邮件组成员查询
// 由于UUAP不支持批量查询，通过缓存减少重复调用
// 缓存时间默认15分钟
func (c *UUAPClient) GetUserNamesByEmailGroupWithCache(ctx context.Context, emailGroupAddress string) ([]string, error) {
	cacher := cache.GetCache()
	if cacher == nil {
		// 缓存不可用，直接调用API
		return c.GetUserNamesByEmailGroup(ctx, emailGroupAddress)
	}

	cacheKey := fmt.Sprintf("%semail_group_members:%s", constants.CacheKeyUUAPReult, emailGroupAddress)

	// 尝试从缓存获取
	cached, err := cacher.Get(ctx, cacheKey)
	if err == nil && cached != "" {
		var users []string
		if err := json.Unmarshal([]byte(cached), &users); err == nil {
			log.Logger.Debugf("Cache hit for email group %s, %d members", emailGroupAddress, len(users))
			return users, nil
		}
	}

	// 缓存未命中，调用API
	users, err := c.GetUserNamesByEmailGroup(ctx, emailGroupAddress)
	if err != nil {
		return nil, err
	}

	// 写入缓存
	if data, err := json.Marshal(users); err == nil {
		if err := cacher.Set(ctx, cacheKey, string(data), EmailGroupMemberCacheTTL); err != nil {
			log.Logger.Warnf("Failed to cache email group members: %v", err)
		}
	}

	return users, nil
}

// GetUserNamesByDepartmentWithCache 带缓存的部门成员查询
// 由于UUAP不支持批量查询，通过缓存减少重复调用
// 缓存时间默认15分钟
func (c *UUAPClient) GetUserNamesByDepartmentWithCache(ctx context.Context, departmentId int64, departmentName string) ([]string, error) {
	cacher := cache.GetCache()
	if cacher == nil {
		// 缓存不可用，直接调用API
		return c.GetUserNamesByDepartment(ctx, departmentId, departmentName)
	}

	cacheKey := fmt.Sprintf("%sdepartment_members:%d:%s", constants.CacheKeyUUAPReult, departmentId, departmentName)

	// 尝试从缓存获取
	cached, err := cacher.Get(ctx, cacheKey)
	if err == nil && cached != "" {
		var users []string
		if err := json.Unmarshal([]byte(cached), &users); err == nil {
			log.Logger.Debugf("Cache hit for department %d:%s, %d members", departmentId, departmentName, len(users))
			return users, nil
		}
	}

	// 缓存未命中，调用API
	users, err := c.GetUserNamesByDepartment(ctx, departmentId, departmentName)
	if err != nil {
		return nil, err
	}

	// 写入缓存
	if data, err := json.Marshal(users); err == nil {
		if err := cacher.Set(ctx, cacheKey, string(data), DepartmentMemberCacheTTL); err != nil {
			log.Logger.Warnf("Failed to cache department members: %v", err)
		}
	}

	return users, nil
}

// InvalidateEmailGroupMemberCache 清除邮件组成员缓存
func (c *UUAPClient) InvalidateEmailGroupMemberCache(ctx context.Context, emailGroupAddress string) error {
	cacher := cache.GetCache()
	if cacher == nil {
		return nil
	}

	cacheKey := fmt.Sprintf("%semail_group_members:%s", constants.CacheKeyUUAPReult, emailGroupAddress)
	return cacher.Delete(ctx, cacheKey)
}

// InvalidateDepartmentMemberCache 清除部门成员缓存
func (c *UUAPClient) InvalidateDepartmentMemberCache(ctx context.Context, departmentId int64, departmentName string) error {
	cacher := cache.GetCache()
	if cacher == nil {
		return nil
	}

	cacheKey := fmt.Sprintf("%sdepartment_members:%d:%s", constants.CacheKeyUUAPReult, departmentId, departmentName)
	return cacher.Delete(ctx, cacheKey)
}
