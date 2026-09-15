// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/cache"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/contextmap"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/database"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/repository"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/log"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/types/constants"
)

// PutPageLinks put_page 链接操作统计
type PutPageLinks struct {
	Created int `json:"created"`
	Removed int `json:"removed"`
	Errors  int `json:"errors"`
}

// PutPageTimeline put_page 时间线操作统计
type PutPageTimeline struct {
	Created int `json:"created"`
}

// PutPageResult put_page 操作结果
type PutPageResult struct {
	Slug     string           `json:"slug"`
	Status   string           `json:"status"`
	Chunks   int              `json:"chunks,omitempty"`
	Links    *PutPageLinks    `json:"links,omitempty"`
	Timeline *PutPageTimeline `json:"timeline,omitempty"`
}

// BrainService brain 模块服务接口
type BrainService interface {
	// PutPage 写入/更新 brain page（双写：MySQL + context-map）
	PutPage(ctx context.Context, kbUUID, operator string, isAdmin bool, slug, content string) (*PutPageResult, error)
	// GetPage 读取单页全文（透传 context-map）
	GetPage(ctx context.Context, kbUUID, operator string, isAdmin bool, slug string, fuzzy bool) ([]byte, int, error)
	// SearchPages 关键词搜索（透传 context-map）
	SearchPages(ctx context.Context, kbUUID, operator string, isAdmin bool, q string, limit, offset int) ([]byte, int, error)
	// QueryPages 混合检索（透传 context-map）
	QueryPages(ctx context.Context, kbUUID, operator string, isAdmin bool, q string, limit int, expand bool, detail string) ([]byte, int, error)
	// ListPages 枚举页面（透传 context-map）
	ListPages(ctx context.Context, kbUUID, operator string, isAdmin bool, pageType, tag string, limit, offset int) ([]byte, int, error)
	// GetBacklinks 获取反向链接（透传 context-map）
	GetBacklinks(ctx context.Context, kbUUID, operator string, isAdmin bool, slug string) ([]byte, int, error)
	// TraverseGraph 图谱遍历（透传 context-map）
	TraverseGraph(ctx context.Context, kbUUID, operator string, isAdmin bool, slug, linkType, direction string, depth int) ([]byte, int, error)
	// GetTimeline 读取时间线（透传 context-map）
	GetTimeline(ctx context.Context, kbUUID, operator string, isAdmin bool, slug, since, until string, limit int) ([]byte, int, error)
	// AddTimelineEntry 追加单条时间线事件（不触发分块/embedding/autolink）
	AddTimelineEntry(ctx context.Context, kbUUID, operator string, isAdmin bool, slug, date, summary, detail, source string) error
	// DeletePage 删除单页（双删：MySQL 软删 + context-map 删除）
	DeletePage(ctx context.Context, kbUUID, operator string, isAdmin bool, slug string) ([]byte, int, error)
	// GetDefaultModel 返回 brain 模块全局默认模型名（面向 notify 子进程等场景）。
	GetDefaultModel(ctx context.Context) string
	// SetDefaultModel 设置 brain 模块全局默认模型名（管理员接口）。
	SetDefaultModel(ctx context.Context, model string) (string, error)
}

// brainServiceImpl brain 服务实现
type brainServiceImpl struct {
	kbRepo            repository.KnowledgebaseRepository
	dirRepo           repository.KnowledgebaseDirectoryRepository
	knowRepo          repository.KnowledgeRepository
	versionRepo       repository.KnowledgeVersionRepository
	logRepo           repository.KnowledgebaseOperationLogRepository
	permissionService PermissionService
	contextMapClient  contextmap.ContextMapClient
	txManager         *database.TransactionManager
}

// NewBrainService 创建 brain 服务实例
func NewBrainService(
	kbRepo repository.KnowledgebaseRepository,
	dirRepo repository.KnowledgebaseDirectoryRepository,
	knowRepo repository.KnowledgeRepository,
	permissionService PermissionService,
	contextMapClient contextmap.ContextMapClient,
	txManager *database.TransactionManager,
	versionRepo repository.KnowledgeVersionRepository,
	logRepo repository.KnowledgebaseOperationLogRepository,
) BrainService {
	return &brainServiceImpl{
		kbRepo:            kbRepo,
		dirRepo:           dirRepo,
		knowRepo:          knowRepo,
		versionRepo:       versionRepo,
		logRepo:           logRepo,
		permissionService: permissionService,
		contextMapClient:  contextMapClient,
		txManager:         txManager,
	}
}

// parseSlug 解析 slug，返回 (dirSegments, title)
// 例: "people/alice" -> (["people"], "alice")
//
//	"a/b/c"        -> (["a","b"], "c")
//	"alice"        -> ([], "alice")
func parseSlug(slug string) ([]string, string) {
	parts := strings.Split(strings.Trim(slug, "/"), "/")
	if len(parts) == 1 {
		return nil, parts[0]
	}
	return parts[:len(parts)-1], parts[len(parts)-1]
}

// ensureArkBrainDir 在事务中确保 ark-brain/[segments...] 目录存在，返回最深层目录 UUID
// 如果 segments 为空，则返回 ark-brain 目录 UUID
func (s *brainServiceImpl) ensureArkBrainDir(txCtx *database.TxContext, kbUUID string, segments []string) (string, error) {
	// 确保 ark-brain 根目录存在
	arkRoot, err := s.dirRepo.FindByKBUUIDAndNameAndParent(txCtx, kbUUID, constants.ArkBrainRootDir, nil)
	if err != nil {
		return "", fmt.Errorf("查询 ark-brain 目录失败: %w", err)
	}
	if arkRoot == nil {
		arkRoot = &domain.KnowledgebaseDirectory{
			UUID:              uuid.New().String(),
			KnowledgebaseUUID: kbUUID,
			Name:              constants.ArkBrainRootDir,
			ParentUUID:        nil,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}
		if err := s.dirRepo.Create(txCtx, arkRoot); err != nil {
			return "", fmt.Errorf("创建 ark-brain 目录失败: %w", err)
		}
	}

	current := arkRoot
	for _, seg := range segments {
		parentUUID := &current.UUID
		dir, err := s.dirRepo.FindByKBUUIDAndNameAndParent(txCtx, kbUUID, seg, parentUUID)
		if err != nil {
			return "", fmt.Errorf("查询目录 %s 失败: %w", seg, err)
		}
		if dir == nil {
			dir = &domain.KnowledgebaseDirectory{
				UUID:              uuid.New().String(),
				KnowledgebaseUUID: kbUUID,
				Name:              seg,
				ParentUUID:        parentUUID,
				CreatedAt:         time.Now(),
				UpdatedAt:         time.Now(),
			}
			if err := s.dirRepo.Create(txCtx, dir); err != nil {
				return "", fmt.Errorf("创建目录 %s 失败: %w", seg, err)
			}
		}
		current = dir
	}
	return current.UUID, nil
}

// PutPage 写入/更新 brain page
func (s *brainServiceImpl) PutPage(ctx context.Context, kbUUID, operator string, isAdmin bool, slug, content string) (*PutPageResult, error) {

	if slug == "" {
		return nil, errors.New(errors.ErrCodeInvalidParam, "slug 不能为空")
	}

	dirSegments, title := parseSlug(slug)

	// 校验深度：ark-brain(1) + segments + title <= DirectoryMaxDepth(5)
	// 实际目录层级 = 1(ark-brain) + len(dirSegments)，知识不占目录层，无需限制
	// 但目录最大深度为 5，ark-brain 是第 1 层，所以 segments 最多 4 层
	if len(dirSegments) >= constants.DirectoryMaxDepth {
		return nil, errors.New(errors.ErrCodeInvalidParam,
			fmt.Sprintf("slug 目录层级超过最大限制 %d", constants.DirectoryMaxDepth-1))
	}

	status := "created"
	metadata := map[string]interface{}{}
	metadata["length"] = len(content)
	metadataStr, _ := json.Marshal(metadata)

	// 最佳努力写入 MySQL（失败不报错，继续转发 context-map）
	if s.txManager != nil {
		dbErr := s.txManager.ExecuteInTransactionWithContext(ctx, func(txCtx *database.TxContext) error {
			dirUUID, err := s.ensureArkBrainDir(txCtx, kbUUID, dirSegments)
			if err != nil {
				return err
			}

			existing, err := s.knowRepo.FindByKbUUIDAndDirUUIDAndTitle(txCtx, kbUUID, &dirUUID, title)
			if err != nil {
				return fmt.Errorf("查询知识失败: %w", err)
			}

			now := time.Now()
			if existing != nil {
				// 更新
				newVersion := GenerateNextVersion(existing.CurrentVersion)
				existing.Content = content
				existing.UpdaterUsername = operator
				existing.UpdatedAt = now
				existing.CurrentVersion = newVersion
				existing.Metadata = string(metadataStr)
				if err := s.knowRepo.Update(txCtx, existing); err != nil {
					return fmt.Errorf("更新知识失败: %w", err)
				}
				status = "updated"

				// 写入版本快照
				verRecord := &domain.KnowledgeVersion{
					UUID:          uuid.New().String(),
					KnowledgeUUID: existing.UUID,
					Version:       newVersion,
					Title:         existing.Title,
					Content:       content,
					Metadata:      existing.Metadata,
					Operator:      operator,
					Operation:     domain.VersionOperationUpdate,
				}
				if err := s.versionRepo.Create(txCtx, verRecord); err != nil {
					return fmt.Errorf("创建版本记录失败: %w", err)
				}

				// 写入操作日志
				detail, _ := json.Marshal(map[string]interface{}{
					"title":   existing.Title,
					"slug":    slug,
					"version": newVersion,
				})
				opLog := &domain.KnowledgebaseOperationLog{
					KnowledgebaseUUID: kbUUID,
					Operator:          operator,
					Operation:         constants.OperationUpdateKnowledge,
					ResourceType:      constants.ResourceTypeKnowledge,
					ResourceUUID:      existing.UUID,
					Detail:            string(detail),
				}
				return s.logRepo.Create(txCtx, opLog)
			}

			// 新建
			initVersion := GetDefaultVersion()
			k := &domain.Knowledge{
				UUID:              uuid.New().String(),
				KnowledgebaseUUID: kbUUID,
				DirectoryUUID:     &dirUUID,
				CreatorUsername:   operator,
				UpdaterUsername:   operator,
				Title:             title,
				Metadata:          string(metadataStr),
				KnowledgeType:     constants.KnowledgeTypeContent,
				KnowledgeSource:   constants.SourceTypeBrain,
				Content:           content,
				CurrentVersion:    initVersion,
				CreatedAt:         now,
				UpdatedAt:         now,
			}
			if err := s.knowRepo.Create(txCtx, k); err != nil {
				return fmt.Errorf("创建知识失败: %w", err)
			}

			// 写入初始版本快照
			initialVer := &domain.KnowledgeVersion{
				UUID:          uuid.New().String(),
				KnowledgeUUID: k.UUID,
				Version:       initVersion,
				Title:         k.Title,
				Content:       content,
				Metadata:      k.Metadata,
				Operator:      operator,
				Operation:     domain.VersionOperationCreate,
				Remark:        "初始版本",
			}
			if err := s.versionRepo.Create(txCtx, initialVer); err != nil {
				return fmt.Errorf("创建版本记录失败: %w", err)
			}

			// 写入操作日志
			detail, _ := json.Marshal(map[string]interface{}{
				"title":   k.Title,
				"slug":    slug,
				"version": initVersion,
			})
			opLog := &domain.KnowledgebaseOperationLog{
				KnowledgebaseUUID: kbUUID,
				Operator:          operator,
				Operation:         constants.OperationCreateKnowledge,
				ResourceType:      constants.ResourceTypeKnowledge,
				ResourceUUID:      k.UUID,
				Detail:            string(detail),
			}
			return s.logRepo.Create(txCtx, opLog)
		})

		if dbErr != nil {
			log.Logger.Warnf("brain put_page 写入 MySQL 失败（将继续转发 context-map）: slug=%s, err=%v", slug, dbErr)
			status = "skipped"
		}
	}

	// 转发到 context-map（失败返回 502）
	if s.contextMapClient == nil {
		return &PutPageResult{Slug: slug, Status: status}, nil
	}

	rawBytes, statusCode, err := s.contextMapClient.PutPage(ctx, kbUUID, slug, content)
	if err != nil || statusCode >= 500 {
		errMsg := ""
		if err != nil {
			errMsg = err.Error()
		} else {
			errMsg = fmt.Sprintf("context-map 返回 %d: %s", statusCode, string(rawBytes))
		}
		return nil, errors.New(errors.ErrCodeInternalError, "context-map 服务异常: "+errMsg)
	}

	// 解析 context-map 返回的结构化字段
	type cmResp struct {
		Status   string           `json:"status"`
		Chunks   int              `json:"chunks"`
		Links    *PutPageLinks    `json:"links"`
		Timeline *PutPageTimeline `json:"timeline"`
	}
	var cm cmResp
	if len(rawBytes) > 0 {
		_ = json.Unmarshal(rawBytes, &cm)
	}
	// 优先使用 context-map 返回的 status（权威数据源），回退到 MySQL 派生的 status
	if cm.Status != "" {
		status = cm.Status
	}

	return &PutPageResult{
		Slug:     slug,
		Status:   status,
		Chunks:   cm.Chunks,
		Links:    cm.Links,
		Timeline: cm.Timeline,
	}, nil
}

// GetPage 读取单页全文
func (s *brainServiceImpl) GetPage(ctx context.Context, kbUUID, operator string, isAdmin bool, slug string, fuzzy bool) ([]byte, int, error) {
	if s.contextMapClient == nil {
		return nil, http.StatusServiceUnavailable, errors.New(errors.ErrCodeInternalError, "context-map 未配置")
	}
	return s.contextMapClient.GetPage(ctx, kbUUID, slug, fuzzy)
}

// SearchPages 关键词搜索
func (s *brainServiceImpl) SearchPages(ctx context.Context, kbUUID, operator string, isAdmin bool, q string, limit, offset int) ([]byte, int, error) {
	if s.contextMapClient == nil {
		return nil, http.StatusServiceUnavailable, errors.New(errors.ErrCodeInternalError, "context-map 未配置")
	}
	return s.contextMapClient.SearchPages(ctx, kbUUID, q, limit, offset)
}

// QueryPages 混合检索
func (s *brainServiceImpl) QueryPages(ctx context.Context, kbUUID, operator string, isAdmin bool, q string, limit int, expand bool, detail string) ([]byte, int, error) {
	if s.contextMapClient == nil {
		return nil, http.StatusServiceUnavailable, errors.New(errors.ErrCodeInternalError, "context-map 未配置")
	}
	return s.contextMapClient.QueryPages(ctx, kbUUID, q, limit, expand, detail)
}

// ListPages 枚举页面
func (s *brainServiceImpl) ListPages(ctx context.Context, kbUUID, operator string, isAdmin bool, pageType, tag string, limit, offset int) ([]byte, int, error) {
	if s.contextMapClient == nil {
		return nil, http.StatusServiceUnavailable, errors.New(errors.ErrCodeInternalError, "context-map 未配置")
	}
	return s.contextMapClient.ListPages(ctx, kbUUID, pageType, tag, limit, offset)
}

// GetBacklinks 获取反向链接
func (s *brainServiceImpl) GetBacklinks(ctx context.Context, kbUUID, operator string, isAdmin bool, slug string) ([]byte, int, error) {
	if s.contextMapClient == nil {
		return nil, http.StatusServiceUnavailable, errors.New(errors.ErrCodeInternalError, "context-map 未配置")
	}
	return s.contextMapClient.GetBacklinks(ctx, kbUUID, slug)
}

// TraverseGraph 图谱遍历
func (s *brainServiceImpl) TraverseGraph(ctx context.Context, kbUUID, operator string, isAdmin bool, slug, linkType, direction string, depth int) ([]byte, int, error) {
	if s.contextMapClient == nil {
		return nil, http.StatusServiceUnavailable, errors.New(errors.ErrCodeInternalError, "context-map 未配置")
	}
	return s.contextMapClient.TraverseGraph(ctx, kbUUID, slug, linkType, direction, depth)
}

// GetTimeline 读取时间线
func (s *brainServiceImpl) GetTimeline(ctx context.Context, kbUUID, operator string, isAdmin bool, slug, since, until string, limit int) ([]byte, int, error) {
	if s.contextMapClient == nil {
		return nil, http.StatusServiceUnavailable, errors.New(errors.ErrCodeInternalError, "context-map 未配置")
	}
	return s.contextMapClient.GetTimeline(ctx, kbUUID, slug, since, until, limit)
}

// findArkBrainDir 在事务中沿 ark-brain/[segments...] 路径查找已有目录，全部存在则返回最深层 UUID
// 若任一段不存在则返回 ("", false, nil)；segments 为空时返回 ark-brain 根目录 UUID
func (s *brainServiceImpl) findArkBrainDir(txCtx *database.TxContext, kbUUID string, segments []string) (string, bool, error) {
	arkRoot, err := s.dirRepo.FindByKBUUIDAndNameAndParent(txCtx, kbUUID, constants.ArkBrainRootDir, nil)
	if err != nil {
		return "", false, fmt.Errorf("查询 ark-brain 目录失败: %w", err)
	}
	if arkRoot == nil {
		return "", false, nil
	}

	current := arkRoot
	for _, seg := range segments {
		parentUUID := &current.UUID
		dir, err := s.dirRepo.FindByKBUUIDAndNameAndParent(txCtx, kbUUID, seg, parentUUID)
		if err != nil {
			return "", false, fmt.Errorf("查询目录 %s 失败: %w", seg, err)
		}
		if dir == nil {
			return "", false, nil
		}
		current = dir
	}
	return current.UUID, true, nil
}

// DeletePage 删除单页（双删：MySQL 软删 + context-map 删除）
func (s *brainServiceImpl) DeletePage(ctx context.Context, kbUUID, operator string, isAdmin bool, slug string) ([]byte, int, error) {

	if slug == "" {
		return nil, 0, errors.New(errors.ErrCodeInvalidParam, "slug 不能为空")
	}

	if s.contextMapClient == nil {
		return nil, http.StatusServiceUnavailable, errors.New(errors.ErrCodeInternalError, "context-map 未配置")
	}

	rawBytes, statusCode, err := s.contextMapClient.DeletePage(ctx, kbUUID, slug)
	if err != nil || statusCode >= 500 {
		errMsg := ""
		if err != nil {
			errMsg = err.Error()
		} else {
			errMsg = fmt.Sprintf("context-map 返回 %d: %s", statusCode, string(rawBytes))
		}
		return nil, 0, errors.New(errors.ErrCodeInternalError, "context-map 服务异常: "+errMsg)
	}
	return rawBytes, statusCode, nil
}

// AddTimelineEntry 追加单条时间线事件（不触发分块/embedding/autolink）
func (s *brainServiceImpl) AddTimelineEntry(ctx context.Context, kbUUID, operator string, isAdmin bool, slug, date, summary, detail, source string) error {
	if s.contextMapClient == nil {
		return errors.New(errors.ErrCodeInternalError, "context-map 未配置")
	}
	_, statusCode, err := s.contextMapClient.AddTimelineEntry(ctx, kbUUID, slug, date, summary, detail, source)
	if err != nil || statusCode >= 500 {
		errMsg := ""
		if err != nil {
			errMsg = err.Error()
		} else {
			errMsg = fmt.Sprintf("context-map 返回 %d", statusCode)
		}
		return errors.New(errors.ErrCodeInternalError, "context-map 服务异常: "+errMsg)
	}
	return nil
}

// GetDefaultModel 返回 brain 模块全局默认模型名
func (s *brainServiceImpl) GetDefaultModel(ctx context.Context) string {
	model, reason := s.lookupDefaultModel(ctx)
	if reason != "" {
		log.Logger.WarnfCtx(ctx, "brain default model fallback to %s: %s", constants.BrainDefaultModelFallback, reason)
		return constants.BrainDefaultModelFallback
	}
	return model
}

// lookupDefaultModel 从 cache 解析默认模型；非空 reason 表示需要走 fallback。
func (s *brainServiceImpl) lookupDefaultModel(ctx context.Context) (model, reason string) {
	cacheClient := cache.GetCache()
	if cacheClient == nil {
		return "", "cache not configured"
	}
	raw, err := cacheClient.Get(ctx, constants.CacheKeyBrainDefaultModel)
	if err != nil {
		return "", fmt.Sprintf("cache get failed: %v", err)
	}
	model = strings.TrimSpace(raw)
	if model == "" {
		return "", "cache value empty"
	}
	return model, ""
}

// SetDefaultModel 设置 brain 模块全局默认模型名（管理员接口）
func (s *brainServiceImpl) SetDefaultModel(ctx context.Context, model string) (string, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		return "", errors.New(errors.ErrCodeInvalidParam, "model 不能为空")
	}

	cacheClient := cache.GetCache()
	if cacheClient == nil {
		log.Logger.ErrorfCtx(ctx, "brain default model: cache not configured, cannot persist model=%s", model)
		return "", errors.New(errors.ErrCodeInternalError, "缓存未配置，无法持久化默认模型")
	}

	// TTL=0 即 redis 的 "no expiration"，持久化配置
	if err := cacheClient.Set(ctx, constants.CacheKeyBrainDefaultModel, model, 0); err != nil {
		log.Logger.ErrorfCtx(ctx, "brain default model: cache set failed: %v", err)
		return "", errors.New(errors.ErrCodeInternalError, "写入缓存失败: "+err.Error())
	}
	log.Logger.InfofCtx(ctx, "brain default model updated to %s", model)
	return model, nil
}
