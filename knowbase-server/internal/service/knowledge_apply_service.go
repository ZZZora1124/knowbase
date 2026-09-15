// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/request"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/database"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/repository"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/log"
)

// AppliedKnowledgeInfo 已应用的知识信息（展开后）
type AppliedKnowledgeInfo struct {
	KnowledgebaseUUID string    `json:"knowledgebaseUuid"`
	KnowledgeUUID     string    `json:"knowledgeUuid"`
	KnowledgeVersion  string    `json:"knowledgeVersion"`
	KnowledgeTitle    string    `json:"knowledgeTitle"`
	KnowledgeContent  string    `json:"knowledgeContent"`
	KnowledgeType     string    `json:"knowledgeType"`
	TemplateName      string    `json:"templateName"`
	Operator          string    `json:"operator"`
	PushStatus        string    `json:"pushStatus"`
	// SourceEntryType 该知识来自 knowledge(直选) 或 directory(目录展开)
	SourceEntryType string `json:"sourceEntryType"`
	// SourceDirectoryUUID 当 SourceEntryType=directory 时，来源目录 UUID
	SourceDirectoryUUID string `json:"sourceDirectoryUuid,omitempty"`
	// SourceDirectoryName 当 SourceEntryType=directory 时，来源目录名称
	SourceDirectoryName string    `json:"sourceDirectoryName,omitempty"`
	CreatedAt           time.Time `json:"createdAt"`
}

// ConfiguredEntryInfo 原始配置条目（知识+目录混合），描述用户保存配置时选择的内容
type ConfiguredEntryInfo struct {
	EntryType        string    `json:"entryType"`
	KnowledgeUUID    string    `json:"knowledgeUuid,omitempty"`
	KnowledgeVersion string    `json:"knowledgeVersion,omitempty"`
	KnowledgeTitle   string    `json:"knowledgeTitle,omitempty"`
	DirectoryUUID    string    `json:"directoryUuid,omitempty"`
	DirectoryName    string    `json:"directoryName,omitempty"`
	Operator         string    `json:"operator"`
	CreatedAt        time.Time `json:"createdAt"`
}

// ApplyConfigWithKnowledge 应用配置及其知识列表
type ApplyConfigWithKnowledge struct {
	Config                  *domain.KnowledgeApplyConfig `json:"config"`
	KnowledgebaseName       string                       `json:"knowledgebaseName"`
	KnowledgebaseVisibility string                       `json:"knowledgebaseVisibility"`
	// ConfiguredEntries 原始配置条目（含知识和目录）
	ConfiguredEntries []*ConfiguredEntryInfo `json:"configuredEntries"`
	// KnowledgeList 展开后的具体知识列表（目录条目已展开到其子树下的所有知识）
	KnowledgeList []*AppliedKnowledgeInfo `json:"knowledgeList"`
}

// ApplyConfigHistory 配置历史版本
type ApplyConfigHistory struct {
	Version           string                  `json:"version"`
	Operator          string                  `json:"operator"`
	CreatedAt         time.Time               `json:"createdAt"`
	ConfiguredEntries []*ConfiguredEntryInfo  `json:"configuredEntries"`
	KnowledgeList     []*AppliedKnowledgeInfo `json:"knowledgeList"`
}

// ApplyResult 应用结果
type ApplyResult struct {
	ConfigUUID    string `json:"configUuid"`
	ConfigVersion string `json:"configVersion"`
	TotalCount    int    `json:"totalCount"`
	SuccessCount  int    `json:"successCount"`
	FailedCount   int    `json:"failedCount"`
}

// RetryResult 重试结果
type RetryResult struct {
	RetryCount   int `json:"retryCount"`
	SuccessCount int `json:"successCount"`
	FailedCount  int `json:"failedCount"`
}

// KnowledgeApplyService 知识应用服务接口
type KnowledgeApplyService interface {
	// CreateOrUpdateConfig 创建或更新应用配置（全量替换知识列表）
	CreateOrUpdateConfig(ctx context.Context, req *request.CreateOrUpdateApplyConfigRequest, operator string, isAdmin bool) (*ApplyResult, error)

	// GetConfig 获取应用配置详情（含当前生效的知识列表）
	GetConfig(ctx context.Context, configUUID, operator string, isAdmin bool) (*ApplyConfigWithKnowledge, error)

	// ListConfigs 查询应用配置列表
	ListConfigs(ctx context.Context, req *request.ListApplyConfigRequest, operator string) ([]*ApplyConfigWithKnowledge, int64, error)

	// GetConfigHistory 获取配置变更历史
	GetConfigHistory(ctx context.Context, configUUID, operator string) ([]*ApplyConfigHistory, error)

	// AddKnowledge 增量添加知识到配置
	AddKnowledge(ctx context.Context, req *request.AddKnowledgeToConfigRequest, operator string, isAdmin bool) (*ApplyResult, error)

	// RemoveKnowledge 从配置中移除知识
	RemoveKnowledge(ctx context.Context, req *request.RemoveKnowledgeFromConfigRequest, operator string, isAdmin bool) error

	// UpdateConfigStatus 更新配置状态（启用/禁用）
	UpdateConfigStatus(ctx context.Context, configUUID, status, operator string, isAdmin bool) error

	// DeleteConfig 删除配置
	DeleteConfig(ctx context.Context, configUUID, operator string, isAdmin bool) error

	// RetryPush 重试推送失败的记录
	RetryPush(ctx context.Context, configUUID, operator string) (*RetryResult, error)
}

// knowledgeApplyServiceImpl 知识应用服务实现
type knowledgeApplyServiceImpl struct {
	kbRepo            repository.KnowledgebaseRepository
	knowRepo          repository.KnowledgeRepository
	versionRepo       repository.KnowledgeVersionRepository
	configRepo        repository.KnowledgeApplyConfigRepository
	recordRepo        repository.KnowledgeApplyRecordRepository
	dirRepo           repository.KnowledgebaseDirectoryRepository
	permissionService PermissionService
	agentNotifier     AgentNotifier
	txManager         *database.TransactionManager
}

// NewKnowledgeApplyService 创建知识应用服务实例
func NewKnowledgeApplyService(
	kbRepo repository.KnowledgebaseRepository,
	knowRepo repository.KnowledgeRepository,
	versionRepo repository.KnowledgeVersionRepository,
	configRepo repository.KnowledgeApplyConfigRepository,
	recordRepo repository.KnowledgeApplyRecordRepository,
	dirRepo repository.KnowledgebaseDirectoryRepository,
	permissionService PermissionService,
	agentNotifier AgentNotifier,
) KnowledgeApplyService {
	return &knowledgeApplyServiceImpl{
		kbRepo:            kbRepo,
		knowRepo:          knowRepo,
		versionRepo:       versionRepo,
		configRepo:        configRepo,
		recordRepo:        recordRepo,
		dirRepo:           dirRepo,
		permissionService: permissionService,
		agentNotifier:     agentNotifier,
		txManager:         database.NewTransactionManager(),
	}
}

// CreateOrUpdateConfig 创建或更新应用配置
func (s *knowledgeApplyServiceImpl) CreateOrUpdateConfig(ctx context.Context, req *request.CreateOrUpdateApplyConfigRequest, operator string, isAdmin bool) (*ApplyResult, error) {
	// 1. 检查知识本是否存在
	kb, err := s.kbRepo.FindByUUID(nil, req.KnowledgebaseUUID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询知识本失败", err)
	}
	if kb == nil {
		return nil, errors.ErrKnowledgebaseNotFound
	}

	// 2. 权限检查
	if err := s.permissionService.CheckEditPermission(kb, req.KnowledgebaseUUID, operator, isAdmin); err != nil {
		return nil, err
	}

	// 3. 检查是否已存在配置
	existingConfig, err := s.configRepo.FindByTarget(nil, req.KnowledgebaseUUID, req.AgentCode, req.Scene, req.TargetType, req.TargetID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询配置失败", err)
	}

	// 4. 校验条目：按 entryType 分类校验
	knowledgeMap, err := s.validateEntryItems(req.KnowledgebaseUUID, req.KnowledgeList)
	if err != nil {
		return nil, err
	}

	// 5. 生成版本号
	var configUUID string
	version := req.Version
	if existingConfig != nil {
		configUUID = existingConfig.UUID
		if version == "" {
			version = GenerateNextVersion(existingConfig.Version)
		}
	} else {
		configUUID = uuid.New().String()
		if version == "" {
			version = GetDefaultVersion()
		}
	}
	if err := ValidateVersion(version); err != nil {
		return nil, err
	}

	// 6. 查询旧记录，获取已存在条目的创建时间（按 entryType 维度保留时间戳）
	oldCreatedAtMap := make(map[string]time.Time) // key: entryKey(record)
	if existingConfig != nil {
		oldRecords, _ := s.recordRepo.ListByConfigUUID(nil, configUUID)
		for _, oldRec := range oldRecords {
			key := entryKeyFromRecord(oldRec)
			if existingTime, exists := oldCreatedAtMap[key]; !exists || oldRec.CreatedAt.After(existingTime) {
				oldCreatedAtMap[key] = oldRec.CreatedAt
			}
		}
	}

	// 7. 构建应用记录
	records := make([]*domain.KnowledgeApplyRecord, len(req.KnowledgeList))
	for i, item := range req.KnowledgeList {
		records[i] = s.buildRecordFromItem(configUUID, version, req.KnowledgebaseUUID, operator, item, knowledgeMap, oldCreatedAtMap)
	}

	// 8. 序列化 TargetExtra
	var targetExtra string
	if len(req.TargetExtra) > 0 {
		extraBytes, err := json.Marshal(req.TargetExtra)
		if err != nil {
			return nil, errors.Wrap(errors.ErrCodeInvalidParam, "TargetExtra 格式错误", err)
		}
		targetExtra = string(extraBytes)
	}

	// 9. 事务操作
	err = s.txManager.ExecuteInTransactionWithContext(ctx, func(txCtx *database.TxContext) error {
		if existingConfig != nil {
			// 更新配置
			existingConfig.AgentName = req.AgentName
			existingConfig.TargetName = req.TargetName
			existingConfig.TargetExtra = targetExtra
			existingConfig.Version = version
			existingConfig.Operator = operator
			existingConfig.Remark = req.Remark
			if err := s.configRepo.Update(txCtx, existingConfig); err != nil {
				return errors.Wrap(errors.ErrCodeInternalError, "更新配置失败", err)
			}
		} else {
			// 创建配置
			config := &domain.KnowledgeApplyConfig{
				UUID:              configUUID,
				KnowledgebaseUUID: req.KnowledgebaseUUID,
				AgentCode:         req.AgentCode,
				AgentName:         req.AgentName,
				Scene:             req.Scene,
				TargetType:        req.TargetType,
				TargetID:          req.TargetID,
				TargetName:        req.TargetName,
				TargetExtra:       targetExtra,
				Version:           version,
				Status:            domain.ApplyConfigStatusActive,
				Operator:          operator,
				Remark:            req.Remark,
			}
			if err := s.configRepo.Create(txCtx, config); err != nil {
				return errors.Wrap(errors.ErrCodeInternalError, "创建配置失败", err)
			}
		}

		// 创建应用记录
		if err := s.recordRepo.BatchCreate(txCtx, records); err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "创建应用记录失败", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 10. 异步推送通知
	go s.pushNotification(configUUID, version, existingConfig == nil)

	log.Logger.InfofCtx(ctx, "创建或更新应用配置成功: configUUID=%s, version=%s", configUUID, version)

	return &ApplyResult{
		ConfigUUID:    configUUID,
		ConfigVersion: version,
		TotalCount:    len(records),
		SuccessCount:  0,
		FailedCount:   0,
	}, nil
}

// normalizeEntryType 归一化 entryType，为空则默认 knowledge
func normalizeEntryType(entryType string) string {
	if entryType == "" {
		return domain.EntryTypeKnowledge
	}
	return entryType
}

// entryKeyFromRecord 根据记录构造唯一 key，用于去重和保留旧时间戳
func entryKeyFromRecord(rec *domain.KnowledgeApplyRecord) string {
	entryType := normalizeEntryType(rec.EntryType)
	if entryType == domain.EntryTypeDirectory {
		return entryType + ":" + rec.DirectoryUUID
	}
	return entryType + ":" + rec.KnowledgeUUID + "-" + rec.KnowledgeVersion
}

// entryKeyFromItem 根据请求 item 构造唯一 key
// knowledgeVersion 为最终确定的版本号（已经补全了默认值）
func entryKeyFromItem(item request.ApplyKnowledgeItem, knowledgeVersion string) string {
	entryType := normalizeEntryType(item.EntryType)
	if entryType == domain.EntryTypeDirectory {
		return entryType + ":" + item.DirectoryUUID
	}
	return entryType + ":" + item.KnowledgeUUID + "-" + knowledgeVersion
}

// validateEntryItems 校验条目列表，按 entryType 分类校验；返回 knowledge -> *Knowledge 映射
func (s *knowledgeApplyServiceImpl) validateEntryItems(kbUUID string, items []request.ApplyKnowledgeItem) (map[string]*domain.Knowledge, error) {
	if len(items) == 0 {
		return nil, errors.New(errors.ErrCodeInvalidParam, "条目列表为空")
	}

	knowledgeUUIDs := make([]string, 0, len(items))
	directoryUUIDs := make([]string, 0, len(items))

	for _, item := range items {
		entryType := normalizeEntryType(item.EntryType)
		switch entryType {
		case domain.EntryTypeKnowledge:
			if item.KnowledgeUUID == "" {
				return nil, errors.New(errors.ErrCodeInvalidParam, "entryType=knowledge 时 knowledgeUuid 必填")
			}
			knowledgeUUIDs = append(knowledgeUUIDs, item.KnowledgeUUID)
		case domain.EntryTypeDirectory:
			if item.DirectoryUUID == "" {
				return nil, errors.New(errors.ErrCodeInvalidParam, "entryType=directory 时 directoryUuid 必填")
			}
			directoryUUIDs = append(directoryUUIDs, item.DirectoryUUID)
		default:
			return nil, errors.New(errors.ErrCodeInvalidParam, "不支持的 entryType: "+item.EntryType)
		}
	}

	// 校验知识存在且属于该知识本
	knowledgeMap := make(map[string]*domain.Knowledge)
	if len(knowledgeUUIDs) > 0 {
		knowledgeList, err := s.knowRepo.FindByUUIDs(nil, knowledgeUUIDs)
		if err != nil {
			return nil, errors.Wrap(errors.ErrCodeInternalError, "查询知识失败", err)
		}
		for _, k := range knowledgeList {
			if k.KnowledgebaseUUID != kbUUID {
				return nil, errors.New(errors.ErrCodeInvalidParam, "知识不属于该知识本: "+k.UUID)
			}
			knowledgeMap[k.UUID] = k
		}
		// 去重后再校验数量
		uniqUUIDs := make(map[string]bool)
		for _, u := range knowledgeUUIDs {
			uniqUUIDs[u] = true
		}
		if len(knowledgeMap) != len(uniqUUIDs) {
			return nil, errors.New(errors.ErrCodeInvalidParam, "部分知识不存在")
		}
	}

	// 校验目录存在且属于该知识本
	if len(directoryUUIDs) > 0 {
		dirList, err := s.dirRepo.FindByUUIDs(nil, directoryUUIDs)
		if err != nil {
			return nil, errors.Wrap(errors.ErrCodeInternalError, "查询目录失败", err)
		}
		dirMap := make(map[string]bool)
		for _, d := range dirList {
			if d.KnowledgebaseUUID != kbUUID {
				return nil, errors.New(errors.ErrCodeInvalidParam, "目录不属于该知识本: "+d.UUID)
			}
			dirMap[d.UUID] = true
		}
		uniqDirUUIDs := make(map[string]bool)
		for _, u := range directoryUUIDs {
			uniqDirUUIDs[u] = true
		}
		if len(dirMap) != len(uniqDirUUIDs) {
			return nil, errors.New(errors.ErrCodeInvalidParam, "部分目录不存在")
		}
	}

	return knowledgeMap, nil
}

// buildRecordFromItem 由 item 构造一条应用记录
func (s *knowledgeApplyServiceImpl) buildRecordFromItem(
	configUUID, configVersion, kbUUID, operator string,
	item request.ApplyKnowledgeItem,
	knowledgeMap map[string]*domain.Knowledge,
	oldCreatedAtMap map[string]time.Time,
) *domain.KnowledgeApplyRecord {
	entryType := normalizeEntryType(item.EntryType)

	rec := &domain.KnowledgeApplyRecord{
		UUID:              uuid.New().String(),
		ConfigUUID:        configUUID,
		ConfigVersion:     configVersion,
		KnowledgebaseUUID: kbUUID,
		EntryType:         entryType,
		Operator:          operator,
		PushStatus:        domain.PushStatusPending,
	}

	if entryType == domain.EntryTypeDirectory {
		rec.DirectoryUUID = item.DirectoryUUID
	} else {
		knowledgeVersion := item.KnowledgeVersion
		if knowledgeVersion == "" {
			if k, ok := knowledgeMap[item.KnowledgeUUID]; ok {
				knowledgeVersion = k.CurrentVersion
			}
		}
		rec.KnowledgeUUID = item.KnowledgeUUID
		rec.KnowledgeVersion = knowledgeVersion
	}

	// 保留旧时间戳
	key := entryKeyFromRecord(rec)
	if oldCreatedAt, exists := oldCreatedAtMap[key]; exists {
		rec.CreatedAt = oldCreatedAt
	}

	return rec
}

// pushNotification 推送通知到Agent（目录条目会在推送前展开为具体知识）
func (s *knowledgeApplyServiceImpl) pushNotification(configUUID, configVersion string, isCreate bool) {
	if s.agentNotifier == nil {
		return
	}

	// 获取配置和记录
	config, _ := s.configRepo.FindByUUID(nil, configUUID)
	if config == nil {
		return
	}

	records, _ := s.recordRepo.ListByConfigUUIDAndVersion(nil, configUUID, configVersion)
	if len(records) == 0 {
		return
	}

	// 获取知识本信息
	kb, _ := s.kbRepo.FindByUUID(nil, config.KnowledgebaseUUID)
	if kb == nil {
		return
	}

	// 展开后构造推送条目
	expanded, err := s.expandRecords(records)
	if err != nil {
		log.Logger.Errorf("展开应用记录失败: configUUID=%s, err=%v", configUUID, err)
		// 展开失败时，所有 record 应标记为失败，避免推送空 knowledgeList 并被误标为 success
		recordUUIDs := make([]string, len(records))
		for i, rec := range records {
			recordUUIDs[i] = rec.UUID
		}
		s.recordRepo.BatchUpdatePushStatus(nil, recordUUIDs, domain.PushStatusFailed, "展开目录失败: "+err.Error())
		return
	}

	action := "update"
	if isCreate {
		action = "create"
	}

	knowledgeList := make([]*ApplyNotificationItem, 0, len(expanded))
	if len(expanded) > 0 {
		// 批量获取知识和版本信息，避免 N+1 查询
		knowledgeUUIDs := make([]string, 0, len(expanded))
		versionQueries := make([]repository.KnowledgeVersionQuery, 0, len(expanded))
		for _, e := range expanded {
			knowledgeUUIDs = append(knowledgeUUIDs, e.KnowledgeUUID)
			versionQueries = append(versionQueries, repository.KnowledgeVersionQuery{
				KnowledgeUUID: e.KnowledgeUUID,
				Version:       e.KnowledgeVersion,
			})
		}
		knowledges, _ := s.knowRepo.FindByUUIDs(nil, knowledgeUUIDs)
		knowledgeMap := make(map[string]*domain.Knowledge, len(knowledges))
		for _, k := range knowledges {
			knowledgeMap[k.UUID] = k
		}
		versions, _ := s.versionRepo.BatchFindByQueries(nil, versionQueries)
		versionMap := make(map[string]*domain.KnowledgeVersion, len(versions))
		for _, v := range versions {
			versionMap[v.KnowledgeUUID+":"+v.Version] = v
		}

		// 构造推送条目
		for _, e := range expanded {
			knowledge := knowledgeMap[e.KnowledgeUUID]
			if knowledge == nil {
				continue
			}
			item := &ApplyNotificationItem{
				KnowledgebaseUUID: e.KnowledgebaseUUID,
				KnowledgeUUID:     e.KnowledgeUUID,
				KnowledgeVersion:  e.KnowledgeVersion,
				KnowledgeTitle:    knowledge.Title,
				KnowledgeType:     knowledge.KnowledgeType,
			}
			if v := versionMap[e.KnowledgeUUID+":"+e.KnowledgeVersion]; v != nil {
				item.Content = v.Content
				item.FilePath = v.FilePath
				item.Metadata = v.Metadata
			}
			knowledgeList = append(knowledgeList, item)
		}
	}

	notification := &ApplyNotification{
		ConfigUUID:        configUUID,
		ConfigVersion:     configVersion,
		KnowledgebaseUUID: config.KnowledgebaseUUID,
		KnowledgebaseType: kb.Visibility,
		AgentCode:         config.AgentCode,
		Scene:             config.Scene,
		TargetType:        config.TargetType,
		TargetID:          config.TargetID,
		TargetName:        config.TargetName,
		Operator:          config.Operator,
		Action:            action,
		KnowledgeList:     knowledgeList,
		AppliedAt:         time.Now(),
	}

	// 发送通知
	notifyErr := s.agentNotifier.Notify(notification)

	// 按 record UUID 批量更新推送状态（目录类型 record 也一并写入成功/失败）
	recordUUIDs := make([]string, len(records))
	for i, rec := range records {
		recordUUIDs[i] = rec.UUID
	}

	if notifyErr != nil {
		s.recordRepo.BatchUpdatePushStatus(nil, recordUUIDs, domain.PushStatusFailed, notifyErr.Error())
	} else {
		s.recordRepo.BatchUpdatePushStatus(nil, recordUUIDs, domain.PushStatusSuccess, "")
	}
}

// GetConfig 获取应用配置详情
func (s *knowledgeApplyServiceImpl) GetConfig(ctx context.Context, configUUID, operator string, isAdmin bool) (*ApplyConfigWithKnowledge, error) {
	config, err := s.configRepo.FindByUUID(nil, configUUID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询配置失败", err)
	}
	if config == nil {
		return nil, errors.New(errors.ErrCodeNotFound, "配置不存在")
	}

	// 权限检查
	kb, err := s.kbRepo.FindByUUID(nil, config.KnowledgebaseUUID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询知识本失败", err)
	}
	if kb == nil {
		return nil, errors.ErrKnowledgebaseNotFound
	}
	if !s.permissionService.CheckViewPermission(kb, config.KnowledgebaseUUID, operator, isAdmin) {
		return nil, errors.ErrNotKnowledgebaseMember
	}

	// 获取当前版本的应用记录
	records, err := s.recordRepo.ListByConfigUUIDAndVersion(nil, configUUID, config.Version)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询应用记录失败", err)
	}

	// 构建原始条目 + 展开后的知识列表
	configuredEntries, err := s.buildConfiguredEntries(records)
	if err != nil {
		return nil, err
	}
	knowledgeList, err := s.buildKnowledgeInfoList(records)
	if err != nil {
		return nil, err
	}

	return &ApplyConfigWithKnowledge{
		Config:                  config,
		KnowledgebaseName:       kb.Name,
		KnowledgebaseVisibility: kb.Visibility,
		ConfiguredEntries:       configuredEntries,
		KnowledgeList:           knowledgeList,
	}, nil
}

// ListConfigs 查询应用配置列表
func (s *knowledgeApplyServiceImpl) ListConfigs(ctx context.Context, req *request.ListApplyConfigRequest, operator string) ([]*ApplyConfigWithKnowledge, int64, error) {
	// 处理分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	query := &repository.ListApplyConfigQuery{
		KnowledgebaseUUID: req.KnowledgebaseUUID,
		AgentCode:         req.AgentCode,
		Scene:             req.Scene,
		TargetType:        req.TargetType,
		TargetID:          req.TargetID,
		Status:            req.Status,
		Operator:          req.Operator,
		Offset:            (req.Page - 1) * req.PageSize,
		Limit:             req.PageSize,
	}

	configs, total, err := s.configRepo.ListByConditions(nil, query)
	if err != nil {
		return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "查询配置列表失败", err)
	}

	// 批量获取知识本信息
	kbUUIDs := make([]string, 0, len(configs))
	kbUUIDSet := make(map[string]bool)
	for _, config := range configs {
		if !kbUUIDSet[config.KnowledgebaseUUID] {
			kbUUIDs = append(kbUUIDs, config.KnowledgebaseUUID)
			kbUUIDSet[config.KnowledgebaseUUID] = true
		}
	}
	kbMap := make(map[string]*domain.Knowledgebase)
	for _, kbUUID := range kbUUIDs {
		kb, _ := s.kbRepo.FindByUUID(nil, kbUUID)
		if kb != nil {
			kbMap[kbUUID] = kb
		}
	}

	// 构建结果
	result := make([]*ApplyConfigWithKnowledge, 0, len(configs))
	for _, config := range configs {
		records, _ := s.recordRepo.ListByConfigUUIDAndVersion(nil, config.UUID, config.Version)
		configuredEntries, _ := s.buildConfiguredEntries(records)
		knowledgeList, _ := s.buildKnowledgeInfoList(records)

		item := &ApplyConfigWithKnowledge{
			Config:            config,
			ConfiguredEntries: configuredEntries,
			KnowledgeList:     knowledgeList,
		}
		// 填充知识本信息
		if kb, ok := kbMap[config.KnowledgebaseUUID]; ok {
			item.KnowledgebaseName = kb.Name
			item.KnowledgebaseVisibility = kb.Visibility
		}
		result = append(result, item)
	}

	return result, total, nil
}

// GetConfigHistory 获取配置变更历史
func (s *knowledgeApplyServiceImpl) GetConfigHistory(ctx context.Context, configUUID, operator string) ([]*ApplyConfigHistory, error) {
	config, err := s.configRepo.FindByUUID(nil, configUUID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询配置失败", err)
	}
	if config == nil {
		return nil, errors.New(errors.ErrCodeNotFound, "配置不存在")
	}

	// 获取所有版本号
	versions, err := s.recordRepo.GetDistinctVersionsByConfigUUID(nil, configUUID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询版本列表失败", err)
	}

	// 构建历史记录
	result := make([]*ApplyConfigHistory, 0, len(versions))
	for _, version := range versions {
		records, _ := s.recordRepo.ListByConfigUUIDAndVersion(nil, configUUID, version)
		if len(records) == 0 {
			continue
		}

		configuredEntries, _ := s.buildConfiguredEntries(records)
		knowledgeList, _ := s.buildKnowledgeInfoList(records)
		result = append(result, &ApplyConfigHistory{
			Version:           version,
			Operator:          records[0].Operator,
			CreatedAt:         records[0].CreatedAt,
			ConfiguredEntries: configuredEntries,
			KnowledgeList:     knowledgeList,
		})
	}

	return result, nil
}

// AddKnowledge 增量添加知识/目录到配置
func (s *knowledgeApplyServiceImpl) AddKnowledge(ctx context.Context, req *request.AddKnowledgeToConfigRequest, operator string, isAdmin bool) (*ApplyResult, error) {
	config, err := s.configRepo.FindByUUID(nil, req.ConfigUUID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询配置失败", err)
	}
	if config == nil {
		return nil, errors.New(errors.ErrCodeNotFound, "配置不存在")
	}

	// 权限检查
	kb, err := s.kbRepo.FindByUUID(nil, config.KnowledgebaseUUID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询知识本失败", err)
	}
	if kb == nil {
		return nil, errors.ErrKnowledgebaseNotFound
	}
	if err := s.permissionService.CheckEditPermission(kb, config.KnowledgebaseUUID, operator, isAdmin); err != nil {
		return nil, err
	}

	// 校验新增条目
	knowledgeMap, err := s.validateEntryItems(config.KnowledgebaseUUID, req.KnowledgeList)
	if err != nil {
		return nil, err
	}

	// 生成新版本号
	version := req.Version
	if version == "" {
		version = GenerateNextVersion(config.Version)
	}

	// 获取当前版本的记录
	currentRecords, err := s.recordRepo.ListByConfigUUIDAndVersion(nil, req.ConfigUUID, config.Version)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询当前记录失败", err)
	}

	// 获取该配置的所有历史记录，保留最近创建时间
	allOldRecords, err := s.recordRepo.ListByConfigUUID(nil, req.ConfigUUID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询历史记录失败", err)
	}
	oldCreatedAtMap := make(map[string]time.Time)
	for _, oldRec := range allOldRecords {
		key := entryKeyFromRecord(oldRec)
		if existingTime, exists := oldCreatedAtMap[key]; !exists || oldRec.CreatedAt.After(existingTime) {
			oldCreatedAtMap[key] = oldRec.CreatedAt
		}
	}

	// 已处理条目集合（用于决定哪些旧条目需要保留）
	processedKeys := make(map[string]bool)

	var records []*domain.KnowledgeApplyRecord

	// 1. 处理请求中的新增条目（新增或更新版本）
	for _, item := range req.KnowledgeList {
		record := s.buildRecordFromItem(req.ConfigUUID, version, config.KnowledgebaseUUID, operator, item, knowledgeMap, oldCreatedAtMap)
		records = append(records, record)
		processedKeys[entryKeyFromRecord(record)] = true
	}

	// 2. 保留原有但不在请求中的条目
	for _, rec := range currentRecords {
		key := entryKeyFromRecord(rec)
		if processedKeys[key] {
			continue
		}
		record := &domain.KnowledgeApplyRecord{
			UUID:              uuid.New().String(),
			ConfigUUID:        req.ConfigUUID,
			ConfigVersion:     version,
			KnowledgebaseUUID: config.KnowledgebaseUUID,
			EntryType:         normalizeEntryType(rec.EntryType),
			KnowledgeUUID:     rec.KnowledgeUUID,
			KnowledgeVersion:  rec.KnowledgeVersion,
			DirectoryUUID:     rec.DirectoryUUID,
			Operator:          operator,
			PushStatus:        domain.PushStatusPending,
		}
		if oldCreatedAt, exists := oldCreatedAtMap[key]; exists {
			record.CreatedAt = oldCreatedAt
		}
		records = append(records, record)
	}

	// 事务操作
	err = s.txManager.ExecuteInTransactionWithContext(ctx, func(txCtx *database.TxContext) error {
		if err := s.configRepo.UpdateVersion(txCtx, req.ConfigUUID, version, operator); err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "更新版本号失败", err)
		}
		return s.recordRepo.BatchCreate(txCtx, records)
	})

	if err != nil {
		return nil, err
	}

	// 异步推送
	go s.pushNotification(req.ConfigUUID, version, false)

	log.Logger.InfofCtx(ctx, "添加条目到配置成功: configUUID=%s, version=%s, count=%d", req.ConfigUUID, version, len(records))

	return &ApplyResult{
		ConfigUUID:    req.ConfigUUID,
		ConfigVersion: version,
		TotalCount:    len(records),
		SuccessCount:  0,
		FailedCount:   0,
	}, nil
}

// RemoveKnowledge 从配置中移除条目（知识或目录）
func (s *knowledgeApplyServiceImpl) RemoveKnowledge(ctx context.Context, req *request.RemoveKnowledgeFromConfigRequest, operator string, isAdmin bool) error {
	config, err := s.configRepo.FindByUUID(nil, req.ConfigUUID)
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternalError, "查询配置失败", err)
	}
	if config == nil {
		return errors.New(errors.ErrCodeNotFound, "配置不存在")
	}

	// 权限检查
	kb, err := s.kbRepo.FindByUUID(nil, config.KnowledgebaseUUID)
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternalError, "查询知识本失败", err)
	}
	if kb == nil {
		return errors.ErrKnowledgebaseNotFound
	}
	if err := s.permissionService.CheckEditPermission(kb, config.KnowledgebaseUUID, operator, isAdmin); err != nil {
		return err
	}

	// 前置校验：Entries 和 KnowledgeUUIDs 至少提供一个
	// 否则会原样复制当前 record，产生无意义的新版本和空内容推送
	if len(req.Entries) == 0 && len(req.KnowledgeUUIDs) == 0 {
		return errors.New(errors.ErrCodeInvalidParam, "entries 或 knowledgeUuids 至少填写一项")
	}

	// 生成新版本号
	version := req.Version
	if version == "" {
		version = GenerateNextVersion(config.Version)
	}

	// 获取当前版本的记录
	currentRecords, err := s.recordRepo.ListByConfigUUIDAndVersion(nil, req.ConfigUUID, config.Version)
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternalError, "查询当前记录失败", err)
	}

	// 获取所有历史记录，保留最近创建时间
	allOldRecords, err := s.recordRepo.ListByConfigUUID(nil, req.ConfigUUID)
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternalError, "查询历史记录失败", err)
	}
	oldCreatedAtMap := make(map[string]time.Time)
	for _, oldRec := range allOldRecords {
		key := entryKeyFromRecord(oldRec)
		if existingTime, exists := oldCreatedAtMap[key]; !exists || oldRec.CreatedAt.After(existingTime) {
			oldCreatedAtMap[key] = oldRec.CreatedAt
		}
	}

	// 构建要移除的集合：优先使用 Entries
	removeKnowledgeUUIDs := make(map[string]bool)
	removeDirectoryUUIDs := make(map[string]bool)
	if len(req.Entries) > 0 {
		for _, entry := range req.Entries {
			entryType := normalizeEntryType(entry.EntryType)
			switch entryType {
			case domain.EntryTypeKnowledge:
				if entry.KnowledgeUUID == "" {
					return errors.New(errors.ErrCodeInvalidParam, "entryType=knowledge 时 knowledgeUuid 必填")
				}
				removeKnowledgeUUIDs[entry.KnowledgeUUID] = true
			case domain.EntryTypeDirectory:
				if entry.DirectoryUUID == "" {
					return errors.New(errors.ErrCodeInvalidParam, "entryType=directory 时 directoryUuid 必填")
				}
				removeDirectoryUUIDs[entry.DirectoryUUID] = true
			default:
				return errors.New(errors.ErrCodeInvalidParam, "不支持的 entryType: "+entry.EntryType)
			}
		}
	} else {
		// 兼容旧接口：仅移除 knowledge 类型
		for _, u := range req.KnowledgeUUIDs {
			removeKnowledgeUUIDs[u] = true
		}
	}

	// 构建新记录（排除要移除的条目）
	var records []*domain.KnowledgeApplyRecord
	for _, rec := range currentRecords {
		entryType := normalizeEntryType(rec.EntryType)
		if entryType == domain.EntryTypeDirectory {
			if removeDirectoryUUIDs[rec.DirectoryUUID] {
				continue
			}
		} else {
			if removeKnowledgeUUIDs[rec.KnowledgeUUID] {
				continue
			}
		}

		record := &domain.KnowledgeApplyRecord{
			UUID:              uuid.New().String(),
			ConfigUUID:        req.ConfigUUID,
			ConfigVersion:     version,
			KnowledgebaseUUID: config.KnowledgebaseUUID,
			EntryType:         entryType,
			KnowledgeUUID:     rec.KnowledgeUUID,
			KnowledgeVersion:  rec.KnowledgeVersion,
			DirectoryUUID:     rec.DirectoryUUID,
			Operator:          operator,
			PushStatus:        domain.PushStatusPending,
		}

		key := entryKeyFromRecord(record)
		if oldCreatedAt, exists := oldCreatedAtMap[key]; exists {
			record.CreatedAt = oldCreatedAt
		}
		records = append(records, record)
	}

	// 事务操作
	err = s.txManager.ExecuteInTransactionWithContext(ctx, func(txCtx *database.TxContext) error {
		if err := s.configRepo.UpdateVersion(txCtx, req.ConfigUUID, version, operator); err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "更新版本号失败", err)
		}
		if len(records) > 0 {
			return s.recordRepo.BatchCreate(txCtx, records)
		}
		return nil
	})

	if err != nil {
		return err
	}

	// 异步推送
	go s.pushNotification(req.ConfigUUID, version, false)

	log.Logger.InfofCtx(ctx, "从配置移除条目成功: configUUID=%s, version=%s", req.ConfigUUID, version)

	return nil
}

// UpdateConfigStatus 更新配置状态
func (s *knowledgeApplyServiceImpl) UpdateConfigStatus(ctx context.Context, configUUID, status, operator string, isAdmin bool) error {
	config, err := s.configRepo.FindByUUID(nil, configUUID)
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternalError, "查询配置失败", err)
	}
	if config == nil {
		return errors.New(errors.ErrCodeNotFound, "配置不存在")
	}

	// 权限检查
	kb, err := s.kbRepo.FindByUUID(nil, config.KnowledgebaseUUID)
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternalError, "查询知识本失败", err)
	}
	if kb == nil {
		return errors.ErrKnowledgebaseNotFound
	}
	if err := s.permissionService.CheckEditPermission(kb, config.KnowledgebaseUUID, operator, isAdmin); err != nil {
		return err
	}

	return s.configRepo.UpdateStatus(nil, configUUID, status)
}

// DeleteConfig 删除配置
func (s *knowledgeApplyServiceImpl) DeleteConfig(ctx context.Context, configUUID, operator string, isAdmin bool) error {
	config, err := s.configRepo.FindByUUID(nil, configUUID)
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternalError, "查询配置失败", err)
	}
	if config == nil {
		return errors.New(errors.ErrCodeNotFound, "配置不存在")
	}

	// 权限检查
	kb, err := s.kbRepo.FindByUUID(nil, config.KnowledgebaseUUID)
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternalError, "查询知识本失败", err)
	}
	if kb == nil {
		return errors.ErrKnowledgebaseNotFound
	}
	if err := s.permissionService.CheckEditPermission(kb, config.KnowledgebaseUUID, operator, isAdmin); err != nil {
		return err
	}

	// 推送删除通知
	if s.agentNotifier != nil {
		notification := &ApplyNotification{
			ConfigUUID:        configUUID,
			ConfigVersion:     config.Version,
			KnowledgebaseUUID: config.KnowledgebaseUUID,
			KnowledgebaseType: kb.Visibility,
			AgentCode:         config.AgentCode,
			Scene:             config.Scene,
			TargetType:        config.TargetType,
			TargetID:          config.TargetID,
			TargetName:        config.TargetName,
			Operator:          operator,
			Action:            "delete",
			KnowledgeList:     []*ApplyNotificationItem{},
			AppliedAt:         time.Now(),
		}
		go s.agentNotifier.Notify(notification)
	}

	// 删除配置（应用记录会级联删除）
	return s.configRepo.Delete(nil, configUUID)
}

// RetryPush 重试推送失败的记录
func (s *knowledgeApplyServiceImpl) RetryPush(ctx context.Context, configUUID, operator string) (*RetryResult, error) {
	config, err := s.configRepo.FindByUUID(nil, configUUID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询配置失败", err)
	}
	if config == nil {
		return nil, errors.New(errors.ErrCodeNotFound, "配置不存在")
	}

	// 获取失败的记录
	failedRecords, err := s.recordRepo.ListFailedRecordsByConfigUUID(nil, configUUID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询失败记录失败", err)
	}

	if len(failedRecords) == 0 {
		return &RetryResult{
			RetryCount:   0,
			SuccessCount: 0,
			FailedCount:  0,
		}, nil
	}

	// 重置状态为pending
	uuids := make([]string, len(failedRecords))
	for i, rec := range failedRecords {
		uuids[i] = rec.UUID
	}
	s.recordRepo.BatchUpdatePushStatus(nil, uuids, domain.PushStatusPending, "")

	// 重新推送
	go s.pushNotification(configUUID, config.Version, false)

	return &RetryResult{
		RetryCount:   len(failedRecords),
		SuccessCount: 0,
		FailedCount:  0,
	}, nil
}

// expandBatchSize 目录展开的分页大小，用于 offset+limit 循环取回全量
const expandBatchSize = 2000

// expandedEntry 展开后的一条具体知识（带来源元信息）
type expandedEntry struct {
	KnowledgebaseUUID   string
	KnowledgeUUID       string
	KnowledgeVersion    string
	Operator            string
	PushStatus          string
	CreatedAt           time.Time
	SourceEntryType     string
	SourceDirectoryUUID string
	SourceDirectoryName string
}

// expandRecords 将记录列表动态展开：knowledge 类型直接返回，directory 类型展开为其子树下所有知识
// 去重规则：同一 knowledgeUUID 同时被直选和目录覆盖时，以直选为准（版本冻结语义优先）
func (s *knowledgeApplyServiceImpl) expandRecords(records []*domain.KnowledgeApplyRecord) ([]*expandedEntry, error) {
	if len(records) == 0 {
		return []*expandedEntry{}, nil
	}

	var directoryRecords []*domain.KnowledgeApplyRecord
	var knowledgeRecords []*domain.KnowledgeApplyRecord
	// 分组：按 entryType 分别收集
	for _, rec := range records {
		if normalizeEntryType(rec.EntryType) == domain.EntryTypeDirectory {
			directoryRecords = append(directoryRecords, rec)
		} else {
			knowledgeRecords = append(knowledgeRecords, rec)
		}
	}

	// 1. 先处理 knowledge 直选：这些条目的版本语义最具体，优先保留
	seen := make(map[string]bool) // key: knowledgeUUID
	result := make([]*expandedEntry, 0, len(records))
	for _, rec := range knowledgeRecords {
		if rec.KnowledgeUUID == "" || seen[rec.KnowledgeUUID] {
			continue
		}
		seen[rec.KnowledgeUUID] = true
		result = append(result, &expandedEntry{
			KnowledgebaseUUID: rec.KnowledgebaseUUID,
			KnowledgeUUID:     rec.KnowledgeUUID,
			KnowledgeVersion:  rec.KnowledgeVersion,
			Operator:          rec.Operator,
			PushStatus:        rec.PushStatus,
			CreatedAt:         rec.CreatedAt,
			SourceEntryType:   domain.EntryTypeKnowledge,
		})
	}

	// 2. 处理 directory 展开
	if len(directoryRecords) == 0 {
		return result, nil
	}

	// 同一 config 所有 records 必在同一个 kb 下，一次加载即可复用
	kbUUID := directoryRecords[0].KnowledgebaseUUID
	allDirs, err := s.dirRepo.ListByKBUUID(nil, kbUUID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询目录失败", err)
	}
	// 目录 UUID -> 目录对象 映射，用于查名称 / is_default
	dirByUUID := make(map[string]*domain.KnowledgebaseDirectory, len(allDirs))
	for _, d := range allDirs {
		dirByUUID[d.UUID] = d
	}

	for _, rec := range directoryRecords {
		rootDir, ok := dirByUUID[rec.DirectoryUUID]
		if !ok {
			// 目录已被删除，跳过
			continue
		}
		// 子树 UUID 集合（包含根）
		subtree := append([]string{rec.DirectoryUUID}, CollectSubtreeUUIDs(rec.DirectoryUUID, allDirs)...)

		// 如果子树里有 is_default 目录，需要额外匹配 directory_uuid IS NULL 的历史知识
		includeNull := false
		for _, u := range subtree {
			if d, exists := dirByUUID[u]; exists && d.IsDefault {
				includeNull = true
				break
			}
		}

		// 分页循环取回目录下全量知识，避免单次查询打爆内存
		offset := 0
		for {
			knowledges, total, err := s.knowRepo.ListByConditions(nil, &repository.ListKnowledgeQuery{
				KnowledgebaseUUID:    rec.KnowledgebaseUUID,
				DirectoryUUIDs:       subtree,
				IncludeNullDirectory: includeNull,
				Offset:               offset,
				Limit:                expandBatchSize,
			})
			if err != nil {
				return nil, errors.Wrap(errors.ErrCodeInternalError, "查询目录下知识失败", err)
			}

			for _, k := range knowledges {
				if seen[k.UUID] {
					continue // 已被直选覆盖
				}
				seen[k.UUID] = true
				result = append(result, &expandedEntry{
					KnowledgebaseUUID:   rec.KnowledgebaseUUID,
					KnowledgeUUID:       k.UUID,
					KnowledgeVersion:    k.CurrentVersion,
					Operator:            rec.Operator,
					PushStatus:          rec.PushStatus,
					CreatedAt:           rec.CreatedAt,
					SourceEntryType:     domain.EntryTypeDirectory,
					SourceDirectoryUUID: rec.DirectoryUUID,
					SourceDirectoryName: rootDir.Name,
				})
			}

			// 本批不足一页或已覆盖 total，结束循环
			offset += len(knowledges)
			if len(knowledges) < expandBatchSize || int64(offset) >= total {
				break
			}
		}
	}

	return result, nil
}

// buildConfiguredEntries 由记录构造原始配置条目（含目录名称）
func (s *knowledgeApplyServiceImpl) buildConfiguredEntries(records []*domain.KnowledgeApplyRecord) ([]*ConfiguredEntryInfo, error) {
	if len(records) == 0 {
		return []*ConfiguredEntryInfo{}, nil
	}

	// 批量获取目录名称
	dirUUIDs := make([]string, 0)
	dirUUIDSet := make(map[string]bool)
	knowledgeUUIDs := make([]string, 0)
	knowledgeUUIDSet := make(map[string]bool)
	for _, rec := range records {
		if normalizeEntryType(rec.EntryType) == domain.EntryTypeDirectory {
			if rec.DirectoryUUID != "" && !dirUUIDSet[rec.DirectoryUUID] {
				dirUUIDs = append(dirUUIDs, rec.DirectoryUUID)
				dirUUIDSet[rec.DirectoryUUID] = true
			}
		} else {
			if rec.KnowledgeUUID != "" && !knowledgeUUIDSet[rec.KnowledgeUUID] {
				knowledgeUUIDs = append(knowledgeUUIDs, rec.KnowledgeUUID)
				knowledgeUUIDSet[rec.KnowledgeUUID] = true
			}
		}
	}

	dirNameMap := make(map[string]string)
	if len(dirUUIDs) > 0 {
		dirs, err := s.dirRepo.FindByUUIDs(nil, dirUUIDs)
		if err != nil {
			return nil, errors.Wrap(errors.ErrCodeInternalError, "查询目录失败", err)
		}
		for _, d := range dirs {
			dirNameMap[d.UUID] = d.Name
		}
	}

	knowledgeTitleMap := make(map[string]string)
	if len(knowledgeUUIDs) > 0 {
		knowledges, err := s.knowRepo.FindByUUIDs(nil, knowledgeUUIDs)
		if err != nil {
			return nil, errors.Wrap(errors.ErrCodeInternalError, "查询知识失败", err)
		}
		for _, k := range knowledges {
			knowledgeTitleMap[k.UUID] = k.Title
		}
	}

	entries := make([]*ConfiguredEntryInfo, 0, len(records))
	for _, rec := range records {
		entryType := normalizeEntryType(rec.EntryType)
		entry := &ConfiguredEntryInfo{
			EntryType: entryType,
			Operator:  rec.Operator,
			CreatedAt: rec.CreatedAt,
		}
		if entryType == domain.EntryTypeDirectory {
			entry.DirectoryUUID = rec.DirectoryUUID
			entry.DirectoryName = dirNameMap[rec.DirectoryUUID]
		} else {
			entry.KnowledgeUUID = rec.KnowledgeUUID
			entry.KnowledgeVersion = rec.KnowledgeVersion
			entry.KnowledgeTitle = knowledgeTitleMap[rec.KnowledgeUUID]
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// buildKnowledgeInfoList 构建知识信息列表（展开目录后）
func (s *knowledgeApplyServiceImpl) buildKnowledgeInfoList(records []*domain.KnowledgeApplyRecord) ([]*AppliedKnowledgeInfo, error) {
	if len(records) == 0 {
		return []*AppliedKnowledgeInfo{}, nil
	}

	// 1. 先展开
	expanded, err := s.expandRecords(records)
	if err != nil {
		return nil, err
	}
	if len(expanded) == 0 {
		return []*AppliedKnowledgeInfo{}, nil
	}

	// 2. 批量获取知识信息
	knowledgeUUIDs := make([]string, 0, len(expanded))
	for _, e := range expanded {
		knowledgeUUIDs = append(knowledgeUUIDs, e.KnowledgeUUID)
	}
	knowledgeList, err := s.knowRepo.FindByUUIDs(nil, knowledgeUUIDs)
	if err != nil {
		return nil, err
	}
	knowledgeMap := make(map[string]*domain.Knowledge)
	for _, k := range knowledgeList {
		knowledgeMap[k.UUID] = k
	}

	// 3. 批量获取版本信息
	var versionQueries []repository.KnowledgeVersionQuery
	for _, e := range expanded {
		versionQueries = append(versionQueries, repository.KnowledgeVersionQuery{
			KnowledgeUUID: e.KnowledgeUUID,
			Version:       e.KnowledgeVersion,
		})
	}
	versions, _ := s.versionRepo.BatchFindByQueries(nil, versionQueries)
	versionMap := make(map[string]*domain.KnowledgeVersion)
	for _, v := range versions {
		key := v.KnowledgeUUID + ":" + v.Version
		versionMap[key] = v
	}

	// 4. 构建结果
	result := make([]*AppliedKnowledgeInfo, 0, len(expanded))
	for _, e := range expanded {
		knowledge := knowledgeMap[e.KnowledgeUUID]
		versionKey := e.KnowledgeUUID + ":" + e.KnowledgeVersion
		version := versionMap[versionKey]

		info := &AppliedKnowledgeInfo{
			KnowledgebaseUUID:   e.KnowledgebaseUUID,
			KnowledgeUUID:       e.KnowledgeUUID,
			KnowledgeVersion:    e.KnowledgeVersion,
			Operator:            e.Operator,
			PushStatus:          e.PushStatus,
			CreatedAt:           e.CreatedAt,
			SourceEntryType:     e.SourceEntryType,
			SourceDirectoryUUID: e.SourceDirectoryUUID,
			SourceDirectoryName: e.SourceDirectoryName,
		}

		if version != nil {
			info.KnowledgeTitle = version.Title
			info.KnowledgeContent = version.Content
		} else if knowledge != nil {
			info.KnowledgeTitle = knowledge.Title
			info.KnowledgeContent = knowledge.Content
		}

		if knowledge != nil {
			info.KnowledgeType = knowledge.KnowledgeType
			info.TemplateName = knowledge.TemplateName
		}

		result = append(result, info)
	}

	return result, nil
}
