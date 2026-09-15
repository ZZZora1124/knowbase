// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/request"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain/handler"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/database"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/storage"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/repository"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/log"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/types/constants"
)

// KnowledgeWithTags 带标签的知识
type KnowledgeWithTags struct {
	Knowledge     *domain.Knowledge
	Tags          []*domain.KnowledgebaseTag
	MatchedTags   []*domain.KnowledgebaseTag // 匹配的标签（用于标签查询场景）
	DirectoryName string                     // 目录名称
	DirPath       string                     // 目录完整路径（如 "父目录/子目录"）
}

// KnowledgeService 知识服务接口
type KnowledgeService interface {
	// CreateKnowledge 创建知识
	CreateKnowledge(ctx context.Context, req *request.CreateKnowledgeRequest, operator string, isAdmin bool) (*domain.Knowledge, error)

	// GetKnowledge 查询知识详情
	GetKnowledge(ctx context.Context, uuid, operator string, isAdmin bool) (*domain.Knowledge, []*domain.KnowledgebaseTag, string, string, error)

	// GetKnowledgeForOpenAPI OpenAPI场景查询知识详情（对README类型追加知识列表）
	GetKnowledgeForOpenAPI(ctx context.Context, uuid, operator string, isAdmin bool) (*domain.Knowledge, []*domain.KnowledgebaseTag, string, string, error)

	// GetKnowledgeBatch 批量查询知识详情
	GetKnowledgeBatch(ctx context.Context, uuids []string, operator string, isAdmin bool) ([]*KnowledgeWithTags, error)

	// ListKnowledge 查询知识列表
	ListKnowledge(ctx context.Context, req *request.ListKnowledgeRequest, operator string, isAdmin bool) ([]*KnowledgeWithTags, int64, error)

	// ListKnowledgeForOpenAPI OpenAPI场景查询知识列表（对README类型追加知识列表）
	ListKnowledgeForOpenAPI(ctx context.Context, req *request.ListKnowledgeRequest, operator string, isAdmin bool) ([]*KnowledgeWithTags, int64, error)

	// ListKnowledgeByTags 根据标签查询知识
	ListKnowledgeByTags(ctx context.Context, req *request.ListKnowledgeByTagsRequest, operator string, isAdmin bool) ([]*KnowledgeWithTags, int64, error)

	// UpdateKnowledge 更新知识
	UpdateKnowledge(ctx context.Context, uuid string, req *request.UpdateKnowledgeRequest, operator string, isAdmin bool) (*domain.Knowledge, error)

	// DeleteKnowledge 删除知识
	DeleteKnowledge(ctx context.Context, uuid, operator string, isAdmin bool) error

	// MoveKnowledge 移动知识到其他目录
	MoveKnowledge(ctx context.Context, uuid, dirUUID, operator string, isAdmin bool) error

	// BindTags 绑定标签
	BindTags(ctx context.Context, uuid string, tagUUIDs []string, operator string, isAdmin bool) error

	// UnbindTags 解绑标签
	UnbindTags(ctx context.Context, uuid string, tagUUIDs []string, operator string, isAdmin bool) ([]*domain.KnowledgebaseTag, error)

	// GetTags 查询知识的标签
	GetTags(ctx context.Context, uuid, operator string, isAdmin bool) ([]*domain.KnowledgebaseTag, error)
}

// maxDBContentSize MySQL TEXT字段最大存储大小（65535字节）
// 超过此限制时，content字段存空字符串，查询时通过fileUrl实时fetch
const maxDBContentSize = 65535

// knowledgeServiceImpl 知识服务实现
type knowledgeServiceImpl struct {
	kbRepo            repository.KnowledgebaseRepository
	dirRepo           repository.KnowledgebaseDirectoryRepository
	knowRepo          repository.KnowledgeRepository
	tagRepo           repository.KnowledgebaseTagRepository
	tagRelRepo        repository.KnowledgeTagRelationRepository
	logRepo           repository.KnowledgebaseOperationLogRepository
	versionRepo       repository.KnowledgeVersionRepository
	handlerFactory    *handler.HandlerFactory
	permissionService PermissionService
	txManager         *database.TransactionManager
	bosClient         *storage.BOSClient
}

// NewKnowledgeService 创建知识服务实例
func NewKnowledgeService(
	kbRepo repository.KnowledgebaseRepository,
	dirRepo repository.KnowledgebaseDirectoryRepository,
	knowRepo repository.KnowledgeRepository,
	tagRepo repository.KnowledgebaseTagRepository,
	tagRelRepo repository.KnowledgeTagRelationRepository,
	logRepo repository.KnowledgebaseOperationLogRepository,
	versionRepo repository.KnowledgeVersionRepository,
	handlerFactory *handler.HandlerFactory,
	permissionService PermissionService,
	bosClient *storage.BOSClient,
) KnowledgeService {
	return &knowledgeServiceImpl{
		kbRepo:            kbRepo,
		dirRepo:           dirRepo,
		knowRepo:          knowRepo,
		tagRepo:           tagRepo,
		tagRelRepo:        tagRelRepo,
		logRepo:           logRepo,
		versionRepo:       versionRepo,
		handlerFactory:    handlerFactory,
		permissionService: permissionService,
		txManager:         database.NewTransactionManager(),
		bosClient:         bosClient,
	}
}

// CreateKnowledge 创建知识
func (s *knowledgeServiceImpl) CreateKnowledge(ctx context.Context, req *request.CreateKnowledgeRequest, operator string, isAdmin bool) (*domain.Knowledge, error) {
	// 1. 检查知识本是否存在
	kb, err := s.kbRepo.FindByUUID(nil, req.KnowledgebaseUUID)
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "查询知识本失败: %v", err)
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询知识本失败", err)
	}
	if kb == nil {
		return nil, errors.ErrKnowledgebaseNotFound
	}

	// 2. 权限检查
	if err := s.permissionService.CheckCreatePermission(kb, req.KnowledgebaseUUID, operator, isAdmin); err != nil {
		return nil, err
	}

	// 3. 验证知识源和目录的关系
	if err := s.validateSourceAndDirectory(req); err != nil {
		return nil, err
	}

	// 3.5 自定义知识若未指定目录，自动绑定默认目录
	if req.KnowledgeSource == constants.SourceTypeCustom && (req.DirectoryUUID == nil || *req.DirectoryUUID == "") {
		defaultDir, err := s.dirRepo.FindDefaultByKBUUID(nil, req.KnowledgebaseUUID)
		if err != nil {
			return nil, errors.Wrap(errors.ErrCodeInternalError, "查询默认目录失败", err)
		}
		if defaultDir != nil {
			req.DirectoryUUID = &defaultDir.UUID
		}
	}

	// 4. 检查同一目录下是否存在相同标题的知识
	exists, err := s.knowRepo.ExistsByDirectoryAndTitle(nil, req.KnowledgebaseUUID, req.DirectoryUUID, req.Title)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "检查知识标题失败", err)
	}
	if exists {
		return nil, errors.ErrKnowledgeTitleDuplicate
	}

	// 5. 如果是 readme 类型，检查知识本是否已存在 readme
	if req.KnowledgeType == constants.KnowledgeTypeReadme {
		readmeExists, err := s.knowRepo.ExistsByKnowledgebaseAndType(nil, req.KnowledgebaseUUID, constants.KnowledgeTypeReadme)
		if err != nil {
			return nil, errors.Wrap(errors.ErrCodeInternalError, "检查README是否存在失败", err)
		}
		if readmeExists {
			return nil, errors.ErrReadmeAlreadyExists
		}
	}

	// 6. 获取处理器并验证
	h, err := s.handlerFactory.Create(req.KnowledgeType)
	if err != nil {
		return nil, err
	}

	processData := &handler.KnowledgeProcessData{
		Content:           req.Content,
		File:              req.File,
		Metadata:          req.Metadata,
		KnowledgeSource:   req.KnowledgeSource,
		KnowledgeType:     req.KnowledgeType,
		KnowledgebaseUUID: req.KnowledgebaseUUID,
	}

	if err := h.Validate(processData); err != nil {
		return nil, err
	}

	// 7. 处理知识（上传文件等）
	result, err := h.Process(processData)
	if err != nil {
		return nil, err
	}

	// 8. 构建知识对象
	knowledge := &domain.Knowledge{
		UUID:              uuid.New().String(),
		KnowledgebaseUUID: req.KnowledgebaseUUID,
		DirectoryUUID:     req.DirectoryUUID,
		CreatorUsername:   operator,
		Title:             req.Title,
		KnowledgeType:     req.KnowledgeType,
		TemplateName:      req.TemplateName,
		KnowledgeSource:   req.KnowledgeSource,
		Content:           result.Content,
		FilePath:          result.FilePath,
		FileSize:          result.FileSize,
		FileFormat:        result.FileFormat,
		SourceMethod:      req.SourceMethod,
		CurrentVersion:    GetDefaultVersion(), // 设置初始版本号
	}

	if result.Metadata != nil {
		metadataBytes, _ := json.Marshal(result.Metadata)
		knowledge.Metadata = string(metadataBytes)
	}

	// 兜底：若Content仍为空且为文本文件类型且有fileUrl，尝试从URL获取内容（忽略错误，文件内容可能超过DB字段大小限制）
	if knowledge.Content == "" && knowledge.KnowledgeType == constants.KnowledgeTypeTextFile && knowledge.Metadata != "" {
		if fileURL := extractFileURLFromMetadata(knowledge.Metadata); fileURL != "" {
			if content, err := fetchContentFromURL(fileURL); err == nil && len(content) <= maxDBContentSize {
				knowledge.Content = content
			}
		}
	}

	// 若Content超过DB TEXT字段大小限制(65535 bytes)则清空，避免写入失败
	// 查询时会通过fileUrl实时fetch，不影响前端展示
	if len(knowledge.Content) > maxDBContentSize {
		knowledge.Content = ""
	}

	// 9. 构建初始版本记录
	initialVersion := &domain.KnowledgeVersion{
		UUID:          uuid.New().String(),
		KnowledgeUUID: knowledge.UUID,
		Version:       knowledge.CurrentVersion,
		Title:         knowledge.Title,
		Content:       knowledge.Content,
		FilePath:      knowledge.FilePath,
		FileSize:      knowledge.FileSize,
		FileFormat:    knowledge.FileFormat,
		Metadata:      knowledge.Metadata,
		Operator:      operator,
		Operation:     domain.VersionOperationCreate,
		Remark:        "初始版本",
	}

	// 10. 事务创建知识、版本记录、绑定标签、记录日志
	err = s.txManager.ExecuteInTransactionWithContext(ctx, func(txCtx *database.TxContext) error {
		// 创建知识
		if err := s.knowRepo.Create(txCtx, knowledge); err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "创建知识失败", err)
		}

		// 创建初始版本记录
		if err := s.versionRepo.Create(txCtx, initialVersion); err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "创建版本记录失败", err)
		}

		// 绑定标签
		if len(req.TagUUIDs) > 0 {
			relations := make([]*domain.KnowledgeTagRelation, len(req.TagUUIDs))
			for i, tagUUID := range req.TagUUIDs {
				relations[i] = &domain.KnowledgeTagRelation{
					UUID:          uuid.New().String(),
					KnowledgeUUID: knowledge.UUID,
					TagUUID:       tagUUID,
				}
			}
			if err := s.tagRelRepo.BatchCreate(txCtx, relations); err != nil {
				return errors.Wrap(errors.ErrCodeInternalError, "绑定标签失败", err)
			}
		}

		// 记录操作日志
		detail, _ := json.Marshal(map[string]interface{}{
			"title":           knowledge.Title,
			"knowledgeType":   knowledge.KnowledgeType,
			"knowledgeSource": knowledge.KnowledgeSource,
			"version":         knowledge.CurrentVersion,
		})
		opLog := &domain.KnowledgebaseOperationLog{
			KnowledgebaseUUID: req.KnowledgebaseUUID,
			Operator:          operator,
			Operation:         constants.OperationCreateKnowledge,
			ResourceType:      constants.ResourceTypeKnowledge,
			ResourceUUID:      knowledge.UUID,
			Detail:            string(detail),
		}
		return s.logRepo.Create(txCtx, opLog)
	})

	if err != nil {
		log.Logger.ErrorfCtx(ctx, "创建知识事务失败: %v", err)
		return nil, err
	}

	// 清除知识数量缓存
	s.knowRepo.InvalidateKnowledgeCountCache(req.KnowledgebaseUUID)

	log.Logger.InfofCtx(ctx, "创建知识成功: uuid=%s, title=%s", knowledge.UUID, knowledge.Title)
	return knowledge, nil
}

// GetKnowledge 查询知识详情
func (s *knowledgeServiceImpl) GetKnowledge(ctx context.Context, uuid, operator string, isAdmin bool) (*domain.Knowledge, []*domain.KnowledgebaseTag, string, string, error) {
	// 1. 查询知识
	knowledge, err := s.knowRepo.FindByUUID(nil, uuid)
	if err != nil {
		return nil, nil, "", "", errors.Wrap(errors.ErrCodeInternalError, "查询知识失败", err)
	}
	if knowledge == nil {
		return nil, nil, "", "", errors.ErrKnowledgeNotFound
	}

	// 2. 检查知识本
	kb, err := s.kbRepo.FindByUUID(nil, knowledge.KnowledgebaseUUID)
	if err != nil {
		return nil, nil, "", "", errors.Wrap(errors.ErrCodeInternalError, "查询知识本失败", err)
	}
	if kb == nil {
		return nil, nil, "", "", errors.ErrKnowledgebaseNotFound
	}

	// 3. 权限检查
	if !s.permissionService.CheckViewPermission(kb, knowledge.KnowledgebaseUUID, operator, isAdmin) {
		return nil, nil, "", "", errors.ErrNotKnowledgebaseMember
	}

	// 4. 查询标签
	tags, err := s.tagRelRepo.FindTagsByKnowledgeUUID(nil, uuid)
	if err != nil {
		return nil, nil, "", "", errors.Wrap(errors.ErrCodeInternalError, "查询标签失败", err)
	}

	// 5. 如果Content为空则从metadata中的fileUrl获取内容填充到Content字段
	if knowledge.Content == "" && knowledge.Metadata != "" {
		fileURL := extractFileURLFromMetadata(knowledge.Metadata)
		if fileURL != "" {
			content, err := fetchContentFromURL(fileURL)
			if err != nil {
				log.Logger.WarnfCtx(ctx, "从fileUrl获取文件内容失败: uuid=%s, fileUrl=%s, error=%v", uuid, fileURL, err)
			} else {
				knowledge.Content = content
			}
		}
	}

	// 6. 加载知识本所有目录，构建目录名称和完整路径
	var directoryName string
	var dirPath string
	if knowledge.DirectoryUUID != nil && *knowledge.DirectoryUUID != "" {
		allDirs, err := s.dirRepo.ListByKBUUID(nil, knowledge.KnowledgebaseUUID)
		if err != nil {
			log.Logger.WarnfCtx(ctx, "查询目录列表失败: kbUUID=%s, error=%v", knowledge.KnowledgebaseUUID, err)
		} else {
			dirMap := make(map[string]*domain.KnowledgebaseDirectory, len(allDirs))
			for _, d := range allDirs {
				dirMap[d.UUID] = d
			}
			if d, ok := dirMap[*knowledge.DirectoryUUID]; ok {
				directoryName = d.Name
			}
			dirPath = BuildDirPath(knowledge.DirectoryUUID, dirMap)
		}
	}

	return knowledge, tags, directoryName, dirPath, nil
}

// GetKnowledgeBatch 批量查询知识详情
func (s *knowledgeServiceImpl) GetKnowledgeBatch(ctx context.Context, uuids []string, operator string, isAdmin bool) ([]*KnowledgeWithTags, error) {
	if len(uuids) == 0 {
		return []*KnowledgeWithTags{}, nil
	}

	// 1. 批量查询知识
	knowledgeList, err := s.knowRepo.FindByUUIDs(nil, uuids)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "批量查询知识失败", err)
	}

	if len(knowledgeList) == 0 {
		return []*KnowledgeWithTags{}, nil
	}

	// 2. 按知识本分组，用于批量权限校验
	kbUUIDSet := make(map[string]bool)
	for _, k := range knowledgeList {
		kbUUIDSet[k.KnowledgebaseUUID] = true
	}

	// 3. 批量查询知识本并校验权限
	accessibleKBUUIDs := make(map[string]bool)
	for kbUUID := range kbUUIDSet {
		kb, err := s.kbRepo.FindByUUID(nil, kbUUID)
		if err != nil {
			return nil, errors.Wrap(errors.ErrCodeInternalError, "查询知识本失败", err)
		}
		if kb != nil && s.permissionService.CheckViewPermission(kb, kbUUID, operator, isAdmin) {
			accessibleKBUUIDs[kbUUID] = true
		}
	}

	// 4. 过滤出有权限访问的知识
	var accessibleKnowledge []*domain.Knowledge
	var accessibleUUIDs []string
	for _, k := range knowledgeList {
		if accessibleKBUUIDs[k.KnowledgebaseUUID] {
			accessibleKnowledge = append(accessibleKnowledge, k)
			accessibleUUIDs = append(accessibleUUIDs, k.UUID)
		}
	}

	if len(accessibleKnowledge) == 0 {
		return []*KnowledgeWithTags{}, nil
	}

	// 5. 批量查询标签
	tagsMap, err := s.tagRelRepo.FindTagsByKnowledgeUUIDs(nil, accessibleUUIDs)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "批量查询标签失败", err)
	}

	// 6. 批量查询目录（按知识本分组，避免重复查询）
	// 收集所有涉及的知识本 UUID
	kbDirMap := make(map[string]map[string]*domain.KnowledgebaseDirectory) // kbUUID -> dirMap
	for kbUUID := range kbUUIDSet {
		if !accessibleKBUUIDs[kbUUID] {
			continue
		}
		dirs, err := s.dirRepo.ListByKBUUID(nil, kbUUID)
		if err != nil {
			log.Logger.WarnfCtx(ctx, "批量查询目录失败: kbUUID=%s, %v", kbUUID, err)
			kbDirMap[kbUUID] = map[string]*domain.KnowledgebaseDirectory{}
			continue
		}
		dm := make(map[string]*domain.KnowledgebaseDirectory, len(dirs))
		for _, d := range dirs {
			dm[d.UUID] = d
		}
		kbDirMap[kbUUID] = dm
	}

	// 7. 组装结果
	result := make([]*KnowledgeWithTags, 0, len(accessibleKnowledge))
	for _, k := range accessibleKnowledge {
		dirMap := kbDirMap[k.KnowledgebaseUUID]
		var dirName string
		if k.DirectoryUUID != nil && *k.DirectoryUUID != "" {
			if d, ok := dirMap[*k.DirectoryUUID]; ok {
				dirName = d.Name
			}
		}
		result = append(result, &KnowledgeWithTags{
			Knowledge:     k,
			Tags:          tagsMap[k.UUID],
			DirectoryName: dirName,
			DirPath:       BuildDirPath(k.DirectoryUUID, dirMap),
		})
	}

	return result, nil
}

// ListKnowledge 查询知识列表
func (s *knowledgeServiceImpl) ListKnowledge(ctx context.Context, req *request.ListKnowledgeRequest, operator string, isAdmin bool) ([]*KnowledgeWithTags, int64, error) {
	// 1. 检查知识本是否存在
	kb, err := s.kbRepo.FindByUUID(nil, req.KnowledgebaseUUID)
	if err != nil {
		return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "查询知识本失败", err)
	}
	if kb == nil {
		return nil, 0, errors.ErrKnowledgebaseNotFound
	}

	// 2. 权限检查（readme/config 类型对所有人公开，跳过鉴权）
	skipAuth := req.KnowledgeType == constants.KnowledgeTypeReadme ||
		req.KnowledgeType == constants.KnowledgeTypeConfig
	if !skipAuth {
		if !s.permissionService.CheckViewPermission(kb, req.KnowledgebaseUUID, operator, isAdmin) {
			return nil, 0, errors.ErrNotKnowledgebaseMember
		}
	}

	// 3. 处理分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 500 {
		req.PageSize = 500
	}
	offset := (req.Page - 1) * req.PageSize

	// 4. 构建目录过滤：指定目录时级联包含所有子目录
	query := &repository.ListKnowledgeQuery{
		KnowledgebaseUUID: req.KnowledgebaseUUID,
		KnowledgeSource:   req.KnowledgeSource,
		KnowledgeType:     req.KnowledgeType,
		TemplateName:      req.TemplateName,
		Keyword:           req.Keyword,
		CreatorUsername:   req.CreatorUsername,
		Offset:            offset,
		Limit:             req.PageSize,
	}

	// cachedAllDirs 缓存本次请求的目录列表，避免后续构建 dirMap 时重复查询
	var cachedAllDirs []*domain.KnowledgebaseDirectory

	if req.DirectoryUUID != "" && req.DirectoryUUID != constants.DirectoryRootUUID {
		// 查询该知识本所有目录，收集指定目录及其全部后代
		allDirs, err := s.dirRepo.ListByKBUUID(nil, req.KnowledgebaseUUID)
		if err != nil {
			return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "查询目录列表失败", err)
		}
		cachedAllDirs = allDirs

		// 判断是否为 is_default 目录（需兼容 directory_uuid=NULL 的历史知识）
		var targetDir *domain.KnowledgebaseDirectory
		for _, d := range allDirs {
			if d.UUID == req.DirectoryUUID {
				targetDir = d
				break
			}
		}

		subtreeUUIDs := CollectSubtreeUUIDs(req.DirectoryUUID, allDirs)
		allDirUUIDs := append([]string{req.DirectoryUUID}, subtreeUUIDs...)

		// 统一使用 DirectoryUUIDs 多目录模式；
		// 若目标目录是 is_default，额外包含 directory_uuid=NULL 的历史知识
		query.DirectoryUUIDs = allDirUUIDs
		if targetDir != nil && targetDir.IsDefault {
			query.IncludeNullDirectory = true
		}
	} else if req.DirectoryUUID == constants.DirectoryRootUUID {
		query.DirectoryUUID = req.DirectoryUUID
	}

	list, total, err := s.knowRepo.ListByConditions(nil, query)
	if err != nil {
		return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "查询知识列表失败", err)
	}

	// 4.5 如果查询类型为 readme 且结果为空（且无其他过滤条件），自动创建默认 README
	if req.KnowledgeType == constants.KnowledgeTypeReadme &&
		total == 0 &&
		req.DirectoryUUID == "" &&
		req.Keyword == "" &&
		req.CreatorUsername == "" &&
		req.KnowledgeSource == "" &&
		req.TemplateName == "" {
		item, createErr := s.createDefaultReadme(ctx, kb)
		if createErr != nil {
			log.Logger.WarnfCtx(ctx, "自动创建默认README失败，返回空列表: kbUUID=%s, err=%v", req.KnowledgebaseUUID, createErr)
		} else {
			return []*KnowledgeWithTags{item}, 1, nil
		}
	}

	// 5. 批量查询标签（优化：避免N+1查询）
	result := make([]*KnowledgeWithTags, 0, len(list))
	if len(list) > 0 {
		// 收集所有知识UUID
		knowledgeUUIDs := make([]string, 0, len(list))
		for _, k := range list {
			knowledgeUUIDs = append(knowledgeUUIDs, k.UUID)
		}

		// 批量查询标签
		tagsMap, err := s.tagRelRepo.FindTagsByKnowledgeUUIDs(nil, knowledgeUUIDs)
		if err != nil {
			return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "查询标签失败", err)
		}

		// 加载知识本所有目录，构建 dirMap（复用目录过滤阶段已查询的结果）
		if cachedAllDirs == nil {
			cachedAllDirs, err = s.dirRepo.ListByKBUUID(nil, req.KnowledgebaseUUID)
			if err != nil {
				log.Logger.WarnfCtx(ctx, "查询目录列表失败: kbUUID=%s, error=%v", req.KnowledgebaseUUID, err)
				cachedAllDirs = nil
			}
		}
		dirMap := make(map[string]*domain.KnowledgebaseDirectory, len(cachedAllDirs))
		for _, d := range cachedAllDirs {
			dirMap[d.UUID] = d
		}

		// 组装结果
		for _, k := range list {
			result = append(result, &KnowledgeWithTags{
				Knowledge: k,
				Tags:      tagsMap[k.UUID],
				DirPath:   BuildDirPath(k.DirectoryUUID, dirMap),
			})
		}
	}

	return result, total, nil
}

// ListKnowledgeByTags 根据标签查询知识
func (s *knowledgeServiceImpl) ListKnowledgeByTags(ctx context.Context, req *request.ListKnowledgeByTagsRequest, operator string, isAdmin bool) ([]*KnowledgeWithTags, int64, error) {
	// 1. 检查知识本是否存在
	kb, err := s.kbRepo.FindByUUID(nil, req.KnowledgebaseUUID)
	if err != nil {
		return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "查询知识本失败", err)
	}
	if kb == nil {
		return nil, 0, errors.ErrKnowledgebaseNotFound
	}

	// 2. 权限检查
	if !s.permissionService.CheckViewPermission(kb, req.KnowledgebaseUUID, operator, isAdmin) {
		return nil, 0, errors.ErrNotKnowledgebaseMember
	}

	// 3. 处理目录参数（DirectoryUUIDs 和 DirectoryNames）
	var directoryUUIDs []string
	includeNullDir := false

	if len(req.DirectoryUUIDs) > 0 {
		directoryUUIDs = req.DirectoryUUIDs
	}
	if len(req.DirectoryNames) > 0 {
		// 多级目录下同名目录可能存在于不同层级，使用批量查询返回所有匹配
		dirs, err := s.dirRepo.FindByKBUUIDAndNames(nil, req.KnowledgebaseUUID, req.DirectoryNames)
		if err != nil {
			return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "根据目录名称查询失败", err)
		}
		for _, dir := range dirs {
			directoryUUIDs = append(directoryUUIDs, dir.UUID)
			if dir.IsDefault {
				includeNullDir = true
			}
		}
		// 如果没有找到任何匹配的目录，返回空结果
		if len(directoryUUIDs) == 0 {
			return []*KnowledgeWithTags{}, 0, nil
		}
	}
	// 通过 UUID 列表过滤时，也检查是否包含 is_default 目录
	if !includeNullDir && len(req.DirectoryUUIDs) > 0 {
		for _, dirUUID := range req.DirectoryUUIDs {
			dir, err := s.dirRepo.FindByUUID(nil, dirUUID)
			if err == nil && dir != nil && dir.IsDefault {
				includeNullDir = true
				break
			}
		}
	}

	// 4. 处理标签参数
	var tagUUIDs []string
	if len(req.TagUUIDs) > 0 {
		tagUUIDs = req.TagUUIDs
	}
	if len(req.TagNames) > 0 {
		// 根据标签名称查询UUID
		for _, name := range req.TagNames {
			tag, err := s.tagRepo.FindByKBUUIDAndName(nil, req.KnowledgebaseUUID, name)
			if err == nil && tag != nil {
				tagUUIDs = append(tagUUIDs, tag.UUID)
			}
		}
		// 没有找到任何匹配的标签，返回空结果
		if len(tagUUIDs) == 0 {
			return []*KnowledgeWithTags{}, 0, nil
		}
	}

	// 5. 处理分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 500 {
		req.PageSize = 500
	}
	offset := (req.Page - 1) * req.PageSize

	// 6. 查询知识列表
	matchType := req.MatchType
	if matchType == "" {
		matchType = "any"
	}

	query := &repository.ListByTagsQuery{
		KnowledgebaseUUID:    req.KnowledgebaseUUID,
		DirectoryUUIDs:       directoryUUIDs,
		IncludeNullDirectory: includeNullDir,
		TagUUIDs:             tagUUIDs,
		MatchType:            matchType,
		KnowledgeSource:      req.KnowledgeSource,
		KnowledgeType:        req.KnowledgeType,
		TemplateName:         req.TemplateName,
		Offset:               offset,
		Limit:                req.PageSize,
	}

	list, total, err := s.knowRepo.ListByTags(nil, query)
	if err != nil {
		return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "查询知识列表失败", err)
	}

	// 7. 批量查询标签（优化：避免N+1查询）
	result := make([]*KnowledgeWithTags, 0, len(list))
	if len(list) > 0 {
		// 收集所有知识UUID
		knowledgeUUIDs := make([]string, 0, len(list))
		for _, k := range list {
			knowledgeUUIDs = append(knowledgeUUIDs, k.UUID)
		}

		// 批量查询标签
		tagsMap, err := s.tagRelRepo.FindTagsByKnowledgeUUIDs(nil, knowledgeUUIDs)
		if err != nil {
			return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "查询标签失败", err)
		}

		// 加载知识本所有目录，构建 dirMap
		allDirs, err := s.dirRepo.ListByKBUUID(nil, req.KnowledgebaseUUID)
		if err != nil {
			log.Logger.WarnfCtx(ctx, "查询目录列表失败: kbUUID=%s, error=%v", req.KnowledgebaseUUID, err)
			allDirs = nil
		}
		dirMap := make(map[string]*domain.KnowledgebaseDirectory, len(allDirs))
		for _, d := range allDirs {
			dirMap[d.UUID] = d
		}

		// 构建请求标签UUID的Set，用于快速匹配
		requestedTagUUIDSet := make(map[string]bool)
		for _, tagUUID := range tagUUIDs {
			requestedTagUUIDSet[tagUUID] = true
		}

		// 组装结果
		for _, k := range list {
			allTags := tagsMap[k.UUID]
			// 计算匹配的标签
			var matchedTags []*domain.KnowledgebaseTag
			for _, tag := range allTags {
				if requestedTagUUIDSet[tag.UUID] {
					matchedTags = append(matchedTags, tag)
				}
			}

			result = append(result, &KnowledgeWithTags{
				Knowledge:   k,
				Tags:        allTags,
				MatchedTags: matchedTags,
				DirPath:     BuildDirPath(k.DirectoryUUID, dirMap),
			})
		}
	}

	return result, total, nil
}

// UpdateKnowledge 更新知识
func (s *knowledgeServiceImpl) UpdateKnowledge(ctx context.Context, knowledgeUUID string, req *request.UpdateKnowledgeRequest, operator string, isAdmin bool) (*domain.Knowledge, error) {
	// 1. 查询知识
	knowledge, err := s.knowRepo.FindByUUID(nil, knowledgeUUID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询知识失败", err)
	}
	if knowledge == nil {
		return nil, errors.ErrKnowledgeNotFound
	}

	// 2. 检查知识本
	kb, err := s.kbRepo.FindByUUID(nil, knowledge.KnowledgebaseUUID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询知识本失败", err)
	}
	if kb == nil {
		return nil, errors.ErrKnowledgebaseNotFound
	}

	// 3. 权限检查
	if err := s.permissionService.CheckEditPermissionWithResource(kb, knowledge.KnowledgebaseUUID, knowledge.CreatorUsername, operator, isAdmin); err != nil {
		return nil, err
	}

	// 4. 处理目录更新（如果指定了新目录）
	var directoryChanged bool
	if req.DirectoryUUID != nil {
		var targetDirUUID *string

		// 处理根目录情况：null、空字符串或"/"都表示根目录
		if *req.DirectoryUUID == "" || *req.DirectoryUUID == constants.DirectoryRootUUID {
			targetDirUUID = nil
		} else {
			// 验证目录是否存在且属于当前知识本
			dir, err := s.dirRepo.FindByUUID(nil, *req.DirectoryUUID)
			if err != nil {
				return nil, errors.Wrap(errors.ErrCodeInternalError, "查询目录失败", err)
			}
			if dir == nil || dir.KnowledgebaseUUID != knowledge.KnowledgebaseUUID {
				return nil, errors.New(errors.ErrCodeNotFound, "目录不存在或不属于当前知识本")
			}
			targetDirUUID = req.DirectoryUUID
		}

		// 判断目录是否真的变化了
		if (targetDirUUID == nil && knowledge.DirectoryUUID != nil) ||
			(targetDirUUID != nil && knowledge.DirectoryUUID == nil) ||
			(targetDirUUID != nil && knowledge.DirectoryUUID != nil && *targetDirUUID != *knowledge.DirectoryUUID) {
			directoryChanged = true

			// 检查目标目录下是否存在相同标题的知识
			// 使用更新后的标题（如果有）或当前标题
			checkTitle := knowledge.Title
			if req.Title != "" {
				checkTitle = req.Title
			}

			exists, err := s.knowRepo.ExistsByDirectoryAndTitle(
				nil,
				knowledge.KnowledgebaseUUID,
				targetDirUUID,
				checkTitle,
				knowledge.UUID,
			)
			if err != nil {
				return nil, errors.Wrap(errors.ErrCodeInternalError, "检查目标目录下的知识标题失败", err)
			}
			if exists {
				return nil, errors.New(errors.ErrCodeInvalidParam, "目标目录下已存在相同标题的知识")
			}

			// 更新目录
			knowledge.DirectoryUUID = targetDirUUID
			log.Logger.InfofCtx(ctx, "知识 %s 的目录从 %v 更新到 %v", knowledgeUUID, knowledge.DirectoryUUID, targetDirUUID)
		}
	}

	// 5. 检查标题是否重复（如果只更新了标题，没有更新目录）
	if req.Title != "" && req.Title != knowledge.Title && !directoryChanged {
		exists, err := s.knowRepo.ExistsByDirectoryAndTitle(nil, knowledge.KnowledgebaseUUID, knowledge.DirectoryUUID, req.Title, knowledge.UUID)
		if err != nil {
			return nil, errors.Wrap(errors.ErrCodeInternalError, "检查知识标题失败", err)
		}
		if exists {
			return nil, errors.ErrKnowledgeTitleDuplicate
		}
	}

	// 6. 更新字段
	knowledge.UpdaterUsername = operator
	if req.Title != "" {
		knowledge.Title = req.Title
	}
	if req.Content != nil {
		knowledge.Content = *req.Content
		knowledge.Metadata = fmt.Sprintf(`{"length": "%d"}`, len(*req.Content))
	}

	if req.Metadata != nil {
		metadataBytes, _ := json.Marshal(req.Metadata)
		knowledge.Metadata = string(metadataBytes)
	}

	// 7. 处理文件更新（如果上传了新文件）
	var oldFilePath string
	var newFilePath string
	var fileHandler handler.KnowledgeHandler

	if req.File != nil {
		knowledge.KnowledgeType = constants.KnowledgeTypeTextFile
		h, err := s.handlerFactory.Create(knowledge.KnowledgeType)
		if err != nil {
			return nil, err
		}

		processData := &handler.KnowledgeProcessData{
			File:              req.File,
			KnowledgeSource:   knowledge.KnowledgeSource,
			KnowledgeType:     knowledge.KnowledgeType,
			KnowledgebaseUUID: knowledge.KnowledgebaseUUID,
		}

		if err := h.Validate(processData); err != nil {
			return nil, err
		}

		// 上传新文件
		result, err := h.Process(processData)
		if err != nil {
			return nil, err
		}

		// 保存旧文件路径（不删除）
		oldFilePath = knowledge.FilePath
		newFilePath = result.FilePath
		fileHandler = h

		// 更新知识记录
		knowledge.FilePath = result.FilePath
		knowledge.FileSize = result.FileSize
		knowledge.FileFormat = result.FileFormat
		knowledge.Content = result.Content // 同步更新文件内容

		if result.Metadata != nil {
			metadataBytes, _ := json.Marshal(result.Metadata)
			knowledge.Metadata = string(metadataBytes)
		}
	}

	// 兜底：若Content仍为空且为文本文件类型且有fileUrl，尝试从URL获取内容（忽略错误，文件内容可能超过DB字段大小限制）
	if knowledge.Content == "" && knowledge.KnowledgeType == constants.KnowledgeTypeTextFile && knowledge.Metadata != "" {
		if fileURL := extractFileURLFromMetadata(knowledge.Metadata); fileURL != "" {
			if content, err := fetchContentFromURL(fileURL); err == nil && len(content) <= maxDBContentSize {
				knowledge.Content = content
			}
		}
	}

	// 若Content超过DB TEXT字段大小限制(65535 bytes)则清空，避免写入失败
	// 查询时会通过fileUrl实时fetch，不影响前端展示
	if len(knowledge.Content) > maxDBContentSize {
		knowledge.Content = ""
	}

	// 8. 生成新版本号
	newVersion := GenerateNextVersion(knowledge.CurrentVersion)
	knowledge.CurrentVersion = newVersion

	// 9. 构建新版本记录
	versionRecord := &domain.KnowledgeVersion{
		UUID:          uuid.New().String(),
		KnowledgeUUID: knowledge.UUID,
		Version:       newVersion,
		Title:         knowledge.Title,
		Content:       knowledge.Content,
		FilePath:      knowledge.FilePath,
		FileSize:      knowledge.FileSize,
		FileFormat:    knowledge.FileFormat,
		Metadata:      knowledge.Metadata,
		Operator:      operator,
		Operation:     domain.VersionOperationUpdate,
		Remark:        "",
	}

	// 10. 更新知识和创建版本记录（事务）
	err = s.txManager.ExecuteInTransactionWithContext(ctx, func(txCtx *database.TxContext) error {
		if err := s.knowRepo.Update(txCtx, knowledge); err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "更新知识失败", err)
		}

		// 创建版本记录
		if err := s.versionRepo.Create(txCtx, versionRecord); err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "创建版本记录失败", err)
		}

		// 记录操作日志
		detail, _ := json.Marshal(map[string]interface{}{
			"title":   knowledge.Title,
			"version": newVersion,
		})
		opLog := &domain.KnowledgebaseOperationLog{
			KnowledgebaseUUID: knowledge.KnowledgebaseUUID,
			Operator:          operator,
			Operation:         constants.OperationUpdateKnowledge,
			ResourceType:      constants.ResourceTypeKnowledge,
			ResourceUUID:      knowledgeUUID,
			Detail:            string(detail),
		}
		return s.logRepo.Create(txCtx, opLog)
	})

	if err != nil {
		// 事务失败，删除新上传的文件
		if newFilePath != "" && fileHandler != nil {
			fileHandler.Delete(&domain.Knowledge{FilePath: newFilePath})
		}
		return nil, err
	}

	// 11. 事务成功后，不删除旧文件，支持回滚
	if oldFilePath != "" && fileHandler != nil {
		// go fileHandler.Delete(&domain.Knowledge{FilePath: oldFilePath})
	}

	return knowledge, nil
}

// DeleteKnowledge 删除知识
func (s *knowledgeServiceImpl) DeleteKnowledge(ctx context.Context, uuid, operator string, isAdmin bool) error {
	// 1. 查询知识
	knowledge, err := s.knowRepo.FindByUUID(nil, uuid)
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternalError, "查询知识失败", err)
	}
	if knowledge == nil {
		return errors.ErrKnowledgeNotFound
	}

	// 2. 检查知识本
	kb, err := s.kbRepo.FindByUUID(nil, knowledge.KnowledgebaseUUID)
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternalError, "查询知识本失败", err)
	}
	if kb == nil {
		return errors.ErrKnowledgebaseNotFound
	}

	// 3. 权限检查
	if err := s.permissionService.CheckDeletePermissionWithResource(kb, knowledge.KnowledgebaseUUID, knowledge.CreatorUsername, operator, isAdmin); err != nil {
		return err
	}

	// 4. 获取处理器用于清理资源
	h, err := s.handlerFactory.Create(knowledge.KnowledgeType)
	if err != nil {
		return err
	}

	// 5. 事务删除
	err = s.txManager.ExecuteInTransaction(func(ctx *database.TxContext) error {
		// 删除知识-标签关联
		if err := s.tagRelRepo.DeleteByKnowledgeUUID(ctx, uuid); err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "删除标签关联失败", err)
		}

		// 软删除知识
		if err := s.knowRepo.SoftDelete(ctx, uuid); err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "删除知识失败", err)
		}

		// 清理外部资源（如BOS文件）
		h.Delete(knowledge)

		// 记录操作日志
		detail, _ := json.Marshal(map[string]string{"title": knowledge.Title})
		log := &domain.KnowledgebaseOperationLog{
			KnowledgebaseUUID: knowledge.KnowledgebaseUUID,
			Operator:          operator,
			Operation:         constants.OperationDeleteKnowledge,
			ResourceType:      constants.ResourceTypeKnowledge,
			ResourceUUID:      uuid,
			Detail:            string(detail),
		}
		return s.logRepo.Create(ctx, log)
	})

	if err != nil {
		return err
	}

	// 清除知识数量缓存
	s.knowRepo.InvalidateKnowledgeCountCache(knowledge.KnowledgebaseUUID)

	return nil
}

// MoveKnowledge 移动知识到其他目录
func (s *knowledgeServiceImpl) MoveKnowledge(ctx context.Context, uuid, dirUUID, operator string, isAdmin bool) error {
	// 1. 查询知识
	knowledge, err := s.knowRepo.FindByUUID(nil, uuid)
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternalError, "查询知识失败", err)
	}
	if knowledge == nil {
		return errors.ErrKnowledgeNotFound
	}

	// 2. 检查知识本
	kb, err := s.kbRepo.FindByUUID(nil, knowledge.KnowledgebaseUUID)
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternalError, "查询知识本失败", err)
	}
	if kb == nil {
		return errors.ErrKnowledgebaseNotFound
	}

	// 3. 权限检查
	if err := s.permissionService.CheckEditPermissionWithResource(kb, knowledge.KnowledgebaseUUID, knowledge.CreatorUsername, operator, isAdmin); err != nil {
		return err
	}

	// 4. 验证目录（如果指定了目录）
	var targetDirUUID *string
	if dirUUID != "" {
		dir, err := s.dirRepo.FindByUUID(nil, dirUUID)
		if err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "查询目录失败", err)
		}
		if dir == nil || dir.KnowledgebaseUUID != knowledge.KnowledgebaseUUID {
			return errors.New(errors.ErrCodeNotFound, "目录不存在")
		}
		targetDirUUID = &dirUUID
	}

	// 5. 检查目标目录下是否存在相同标题的知识
	exists, err := s.knowRepo.ExistsByDirectoryAndTitle(nil, knowledge.KnowledgebaseUUID, targetDirUUID, knowledge.Title, knowledge.UUID)
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternalError, "检查知识标题失败", err)
	}
	if exists {
		return errors.ErrKnowledgeTitleDuplicate
	}

	// 6. 更新知识的目录
	return s.txManager.ExecuteInTransaction(func(ctx *database.TxContext) error {
		if err := s.knowRepo.UpdateDirectory(ctx, uuid, dirUUID); err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "移动知识失败", err)
		}

		// 记录操作日志
		detail, _ := json.Marshal(map[string]interface{}{
			"knowledgeUuid": uuid,
			"directoryUuid": dirUUID,
		})
		log := &domain.KnowledgebaseOperationLog{
			KnowledgebaseUUID: knowledge.KnowledgebaseUUID,
			Operator:          operator,
			Operation:         constants.OperationMoveKnowledge,
			ResourceType:      constants.ResourceTypeKnowledge,
			ResourceUUID:      uuid,
			Detail:            string(detail),
		}
		return s.logRepo.Create(ctx, log)
	})
}

// BindTags 绑定标签（增量添加）
func (s *knowledgeServiceImpl) BindTags(ctx context.Context, knowledgeUUID string, tagUUIDs []string, operator string, isAdmin bool) error {
	// 1. 查询知识
	knowledge, err := s.knowRepo.FindByUUID(nil, knowledgeUUID)
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternalError, "查询知识失败", err)
	}
	if knowledge == nil {
		return errors.ErrKnowledgeNotFound
	}

	// 2. 检查知识本
	kb, err := s.kbRepo.FindByUUID(nil, knowledge.KnowledgebaseUUID)
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternalError, "查询知识本失败", err)
	}
	if kb == nil {
		return errors.ErrKnowledgebaseNotFound
	}

	// 3. 权限检查
	if err := s.permissionService.CheckEditPermissionWithResource(kb, knowledge.KnowledgebaseUUID, knowledge.CreatorUsername, operator, isAdmin); err != nil {
		return err
	}

	// 4. 如果没有标签要绑定，直接返回
	if len(tagUUIDs) == 0 {
		return nil
	}

	// 5. 查询已存在的标签关联（避免重复绑定）
	existingTagUUIDSet := make(map[string]bool)
	existingTags, err := s.tagRelRepo.FindTagsByKnowledgeUUID(nil, knowledgeUUID)
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternalError, "查询已有标签失败", err)
	}
	for _, tag := range existingTags {
		existingTagUUIDSet[tag.UUID] = true
	}

	// 6. 过滤出需要新增的标签，并验证标签存在性和归属
	var newTagUUIDs []string
	for _, tagUUID := range tagUUIDs {
		// 跳过已绑定的标签
		if existingTagUUIDSet[tagUUID] {
			continue
		}

		// 验证标签存在性和归属
		tag, err := s.tagRepo.FindByUUID(nil, tagUUID)
		if err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "查询标签失败", err)
		}
		if tag == nil {
			return errors.New(errors.ErrCodeNotFound, "标签不存在: "+tagUUID)
		}
		if tag.KnowledgebaseUUID != knowledge.KnowledgebaseUUID {
			return errors.New(errors.ErrCodeInvalidParam, "标签不属于当前知识本")
		}

		newTagUUIDs = append(newTagUUIDs, tagUUID)
	}

	// 7. 如果没有新标签需要绑定，直接返回
	if len(newTagUUIDs) == 0 {
		return nil
	}

	// 8. 事务绑定新标签
	return s.txManager.ExecuteInTransaction(func(ctx *database.TxContext) error {
		// 创建新的标签关联
		relations := make([]*domain.KnowledgeTagRelation, len(newTagUUIDs))
		for i, tagUUID := range newTagUUIDs {
			relations[i] = &domain.KnowledgeTagRelation{
				UUID:          uuid.New().String(),
				KnowledgeUUID: knowledgeUUID,
				TagUUID:       tagUUID,
			}
		}
		if err := s.tagRelRepo.BatchCreate(ctx, relations); err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "绑定标签失败", err)
		}

		// 记录操作日志
		detail, _ := json.Marshal(map[string]interface{}{
			"tagUuids": newTagUUIDs,
		})
		log := &domain.KnowledgebaseOperationLog{
			KnowledgebaseUUID: knowledge.KnowledgebaseUUID,
			Operator:          operator,
			Operation:         constants.OperationBindTag,
			ResourceType:      constants.ResourceTypeKnowledge,
			ResourceUUID:      knowledgeUUID,
			Detail:            string(detail),
		}
		return s.logRepo.Create(ctx, log)
	})
}

// UnbindTags 解绑标签
func (s *knowledgeServiceImpl) UnbindTags(ctx context.Context, knowledgeUUID string, tagUUIDs []string, operator string, isAdmin bool) ([]*domain.KnowledgebaseTag, error) {
	// 1. 查询知识
	knowledge, err := s.knowRepo.FindByUUID(nil, knowledgeUUID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询知识失败", err)
	}
	if knowledge == nil {
		return nil, errors.ErrKnowledgeNotFound
	}

	// 2. 检查知识本
	kb, err := s.kbRepo.FindByUUID(nil, knowledge.KnowledgebaseUUID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询知识本失败", err)
	}
	if kb == nil {
		return nil, errors.ErrKnowledgebaseNotFound
	}

	// 3. 权限检查
	if err := s.permissionService.CheckEditPermissionWithResource(kb, knowledge.KnowledgebaseUUID, knowledge.CreatorUsername, operator, isAdmin); err != nil {
		return nil, err
	}

	// 4. 如果没有标签要解绑，直接返回当前标签
	if len(tagUUIDs) == 0 {
		tags, err := s.tagRelRepo.FindTagsByKnowledgeUUID(nil, knowledgeUUID)
		if err != nil {
			return nil, errors.Wrap(errors.ErrCodeInternalError, "查询标签失败", err)
		}
		return tags, nil
	}

	// 5. 查询已绑定的标签
	existingTags, err := s.tagRelRepo.FindTagsByKnowledgeUUID(nil, knowledgeUUID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询已绑定标签失败", err)
	}

	// 构建已绑定标签UUID的Set
	existingTagUUIDSet := make(map[string]bool)
	for _, tag := range existingTags {
		existingTagUUIDSet[tag.UUID] = true
	}

	// 6. 验证要解绑的标签是否已绑定
	var validTagUUIDs []string
	for _, tagUUID := range tagUUIDs {
		// 检查标签是否存在
		tag, err := s.tagRepo.FindByUUID(nil, tagUUID)
		if err != nil {
			return nil, errors.Wrap(errors.ErrCodeInternalError, "查询标签失败", err)
		}
		if tag == nil {
			return nil, errors.New(errors.ErrCodeNotFound, "标签不存在: "+tagUUID)
		}
		// 检查标签是否已绑定到该知识
		if !existingTagUUIDSet[tagUUID] {
			return nil, errors.New(errors.ErrCodeInvalidParam, "标签未绑定到该知识: "+tagUUID)
		}
		validTagUUIDs = append(validTagUUIDs, tagUUID)
	}

	// 7. 事务解绑标签
	err = s.txManager.ExecuteInTransaction(func(ctx *database.TxContext) error {
		// 批量删除指定的知识-标签关联
		if err := s.tagRelRepo.BatchDeleteByKnowledgeAndTags(ctx, knowledgeUUID, validTagUUIDs); err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "解绑标签失败", err)
		}

		// 记录操作日志
		detail, _ := json.Marshal(map[string]interface{}{
			"tagUuids": validTagUUIDs,
		})
		log := &domain.KnowledgebaseOperationLog{
			KnowledgebaseUUID: knowledge.KnowledgebaseUUID,
			Operator:          operator,
			Operation:         constants.OperationUnbindTag,
			ResourceType:      constants.ResourceTypeKnowledge,
			ResourceUUID:      knowledgeUUID,
			Detail:            string(detail),
		}
		return s.logRepo.Create(ctx, log)
	})

	if err != nil {
		return nil, err
	}

	// 8. 返回解绑后的标签列表
	tags, err := s.tagRelRepo.FindTagsByKnowledgeUUID(nil, knowledgeUUID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询标签失败", err)
	}

	return tags, nil
}

// GetTags 查询知识的标签
func (s *knowledgeServiceImpl) GetTags(ctx context.Context, knowledgeUUID, operator string, isAdmin bool) ([]*domain.KnowledgebaseTag, error) {
	// 1. 查询知识
	knowledge, err := s.knowRepo.FindByUUID(nil, knowledgeUUID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询知识失败", err)
	}
	if knowledge == nil {
		return nil, errors.ErrKnowledgeNotFound
	}

	// 2. 检查知识本
	kb, err := s.kbRepo.FindByUUID(nil, knowledge.KnowledgebaseUUID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询知识本失败", err)
	}
	if kb == nil {
		return nil, errors.ErrKnowledgebaseNotFound
	}

	// 3. 权限检查
	if !s.permissionService.CheckViewPermission(kb, knowledge.KnowledgebaseUUID, operator, isAdmin) {
		return nil, errors.ErrNotKnowledgebaseMember
	}

	// 4. 查询标签
	tags, err := s.tagRelRepo.FindTagsByKnowledgeUUID(nil, knowledgeUUID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询标签失败", err)
	}

	return tags, nil
}

// validateSourceAndDirectory 验证知识源和目录的关系
func (s *knowledgeServiceImpl) validateSourceAndDirectory(req *request.CreateKnowledgeRequest) error {
	// 自定义知识：可以指定目录，且知识类型不能是config
	if req.KnowledgeSource == constants.SourceTypeCustom {
		if req.KnowledgeType == "config" {
			return errors.New(errors.ErrCodeInvalidKnowledgeType, "自定义知识不能使用config类型")
		}
		// directory_uuid可以为nil或有效值
		if req.DirectoryUUID != nil && *req.DirectoryUUID != "" {
			dir, err := s.dirRepo.FindByUUID(nil, *req.DirectoryUUID)
			if err != nil {
				return errors.Wrap(errors.ErrCodeInternalError, "查询目录失败", err)
			}
			if dir == nil || dir.KnowledgebaseUUID != req.KnowledgebaseUUID {
				return errors.New(errors.ErrCodeNotFound, "目录不存在")
			}
		}
		return nil
	}

	// 其他知识源：directory_uuid必须为NULL
	if req.DirectoryUUID != nil && *req.DirectoryUUID != "" {
		return errors.New(errors.ErrCodeInvalidParam, "非自定义知识不能指定目录")
	}

	return nil
}

// GetKnowledgeForOpenAPI OpenAPI场景查询知识详情（对README类型追加知识列表）
func (s *knowledgeServiceImpl) GetKnowledgeForOpenAPI(ctx context.Context, uuid, operator string, isAdmin bool) (*domain.Knowledge, []*domain.KnowledgebaseTag, string, string, error) {
	// 1. 先获取普通知识详情
	knowledge, tags, directoryName, dirPath, err := s.GetKnowledge(ctx, uuid, operator, isAdmin)
	if err != nil {
		return nil, nil, "", "", err
	}

	// 2. 仅对 README 类型追加知识列表
	if knowledge.KnowledgeType != constants.KnowledgeTypeReadme {
		return knowledge, tags, directoryName, dirPath, nil
	}

	// 3. 构建追加内容
	appendContent, err := s.appendKnowledgeListToReadme(ctx, knowledge.KnowledgebaseUUID)
	if err != nil {
		log.Logger.WarnfCtx(ctx, "追加知识列表失败: %v", err)
		return knowledge, tags, directoryName, dirPath, nil // 不影响主流程
	}

	// 4. 追加到 content 末尾
	if appendContent != "" {
		knowledge.Content += "\n" + appendContent
	}

	return knowledge, tags, directoryName, dirPath, nil
}

// appendKnowledgeListToReadme 并发查询两类知识源，构建追加到 README 的知识列表文本
func (s *knowledgeServiceImpl) appendKnowledgeListToReadme(ctx context.Context, kbUUID string) (string, error) {
	// 两个查询相互独立，并发执行以减少延迟
	type queryResult struct {
		list []*domain.Knowledge
		err  error
	}
	sysCh := make(chan queryResult, 1)
	custCh := make(chan queryResult, 1)

	go func() {
		list, err := s.knowRepo.ListByKnowledgebaseAndSourceNotCustom(nil, kbUUID, constants.OpenAPIReadmeAppendLimitSystem)
		sysCh <- queryResult{list, err}
	}()
	go func() {
		list, err := s.knowRepo.ListByKnowledgebaseAndSourceCustomTopN(nil, kbUUID, constants.OpenAPIReadmeAppendLimitCustom)
		custCh <- queryResult{list, err}
	}()

	sysRes := <-sysCh
	if sysRes.err != nil {
		return "", errors.Wrap(errors.ErrCodeInternalError, "查询系统知识失败", sysRes.err)
	}
	custRes := <-custCh
	if custRes.err != nil {
		return "", errors.Wrap(errors.ErrCodeInternalError, "查询自定义知识失败", custRes.err)
	}

	var sb strings.Builder

	if len(sysRes.list) > 0 {
		sb.WriteString("\n# System Knowledge\n")
		for _, k := range sysRes.list {
			sb.WriteString(fmt.Sprintf("- 知识名称：%s uuid：%s", k.Title, k.UUID))
			if k.Metadata != "" {
				sb.WriteString(fmt.Sprintf(" 详细信息：%s", k.Metadata))
			}
			sb.WriteByte('\n')
		}
		if len(sysRes.list) > constants.OpenAPIReadmeAppendLimitSystem {
			sb.WriteString(fmt.Sprintf("> 注：以上仅展示前 %d 条系统知识，实际数量可能更多。\n", constants.OpenAPIReadmeAppendLimitSystem))
		}
	}

	if len(custRes.list) > 0 {
		if sb.Len() > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(fmt.Sprintf("# Custom Knowledge (Latest %d)\n", constants.OpenAPIReadmeAppendLimitCustom))
		for _, k := range custRes.list {
			sb.WriteString(fmt.Sprintf("- 知识名称：%s uuid：%s\n", k.Title, k.UUID))
		}
		if len(custRes.list) > constants.OpenAPIReadmeAppendLimitCustom {
			sb.WriteString(fmt.Sprintf("> 注：以上仅展示最新 %d 条自定义知识，实际数量可能更多。\n", constants.OpenAPIReadmeAppendLimitCustom))
		}
	}

	return sb.String(), nil
}

// ListKnowledgeForOpenAPI OpenAPI场景查询知识列表（对README类型追加知识列表）
func (s *knowledgeServiceImpl) ListKnowledgeForOpenAPI(ctx context.Context, req *request.ListKnowledgeRequest, operator string, isAdmin bool) ([]*KnowledgeWithTags, int64, error) {
	list, total, err := s.ListKnowledge(ctx, req, operator, isAdmin)
	if err != nil {
		return nil, 0, err
	}

	for _, item := range list {
		if item.Knowledge.KnowledgeType != constants.KnowledgeTypeReadme {
			continue
		}
		appendContent, err := s.appendKnowledgeListToReadme(ctx, item.Knowledge.KnowledgebaseUUID)
		if err != nil {
			log.Logger.WarnfCtx(ctx, "追加知识列表失败: %v", err)
			continue
		}
		if appendContent != "" {
			item.Knowledge.Content += "\n" + appendContent
		}
	}

	return list, total, nil
}

// extractFileURLFromMetadata 从metadata JSON字符串中提取fileUrl字段
func extractFileURLFromMetadata(metadata string) string {
	if metadata == "" {
		return ""
	}
	var metadataMap map[string]interface{}
	if err := json.Unmarshal([]byte(metadata), &metadataMap); err != nil {
		return ""
	}
	if fileURL, ok := metadataMap["fileUrl"].(string); ok {
		return fileURL
	}
	return ""
}

// fetchContentFromURL 通过URL获取文件内容
func fetchContentFromURL(fileURL string) (string, error) {
	resp, err := http.Get(fileURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP status: %d", resp.StatusCode)
	}

	contentBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(contentBytes), nil
}

// createDefaultReadme 自动创建默认 README 知识
// 在 ListKnowledge 查询 readme 类型结果为空时调用
// 操作人为知识方舟的 owner，内容为知识方舟 uuid、名称和描述
func (s *knowledgeServiceImpl) createDefaultReadme(ctx context.Context, kb *domain.Knowledgebase) (*KnowledgeWithTags, error) {
	kbUUID := kb.UUID
	owner := kb.Owner
	content := fmt.Sprintf("本知识方舟uuid: %s, 名称: %s, 描述: %s", kb.UUID, kb.Name, kb.Description)

	// 二次确认，防止并发重复创建
	exists, err := s.knowRepo.ExistsByKnowledgebaseAndType(nil, kbUUID, constants.KnowledgeTypeReadme)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "检查README是否存在失败", err)
	}
	if exists {
		// 已存在则直接查询返回
		query := &repository.ListKnowledgeQuery{
			KnowledgebaseUUID: kbUUID,
			KnowledgeType:     constants.KnowledgeTypeReadme,
			Offset:            0,
			Limit:             1,
		}
		list, _, err := s.knowRepo.ListByConditions(nil, query)
		if err != nil || len(list) == 0 {
			return nil, errors.Wrap(errors.ErrCodeInternalError, "查询已有README失败", err)
		}
		return &KnowledgeWithTags{
			Knowledge: list[0],
			Tags:      []*domain.KnowledgebaseTag{},
		}, nil
	}

	// 构建默认 README 知识对象
	metadata := fmt.Sprintf(`{"length": "%d"}`, len(content))
	knowledge := &domain.Knowledge{
		UUID:              uuid.New().String(),
		KnowledgebaseUUID: kbUUID,
		CreatorUsername:   owner,
		Title:             "README",
		KnowledgeType:     constants.KnowledgeTypeReadme,
		KnowledgeSource:   "custom",
		Content:           content,
		Metadata:          metadata,
		SourceMethod:      "manual",
		CurrentVersion:    GetDefaultVersion(),
	}

	initialVersion := &domain.KnowledgeVersion{
		UUID:          uuid.New().String(),
		KnowledgeUUID: knowledge.UUID,
		Version:       knowledge.CurrentVersion,
		Title:         knowledge.Title,
		Content:       knowledge.Content,
		Metadata:      metadata,
		Operator:      owner,
		Operation:     domain.VersionOperationCreate,
		Remark:        "自动创建默认README",
	}

	err = s.txManager.ExecuteInTransactionWithContext(ctx, func(txCtx *database.TxContext) error {
		if err := s.knowRepo.Create(txCtx, knowledge); err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "创建默认README失败", err)
		}
		if err := s.versionRepo.Create(txCtx, initialVersion); err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "创建默认README版本记录失败", err)
		}
		detail, _ := json.Marshal(map[string]interface{}{
			"title":           knowledge.Title,
			"knowledgeType":   knowledge.KnowledgeType,
			"knowledgeSource": knowledge.KnowledgeSource,
			"version":         knowledge.CurrentVersion,
		})
		opLog := &domain.KnowledgebaseOperationLog{
			KnowledgebaseUUID: kbUUID,
			Operator:          owner,
			Operation:         constants.OperationCreateKnowledge,
			ResourceType:      constants.ResourceTypeKnowledge,
			ResourceUUID:      knowledge.UUID,
			Detail:            string(detail),
		}
		return s.logRepo.Create(txCtx, opLog)
	})
	if err != nil {
		return nil, err
	}

	s.knowRepo.InvalidateKnowledgeCountCache(kbUUID)
	log.Logger.InfofCtx(ctx, "自动创建默认README成功: kbUUID=%s, knowledgeUUID=%s", kbUUID, knowledge.UUID)

	return &KnowledgeWithTags{
		Knowledge: knowledge,
		Tags:      []*domain.KnowledgebaseTag{},
	}, nil
}
