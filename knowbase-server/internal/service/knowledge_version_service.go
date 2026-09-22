package service

import (
	"context"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/request"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/database"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/repository"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
)

type KnowledgeVersionService interface {
	ListVersions(context.Context, string, string, bool) ([]*domain.KnowledgeVersion, error)
	BatchListVersions(context.Context, []string, string, bool) (map[string][]*domain.KnowledgeVersion, error)
	GetVersion(context.Context, string, string, string, bool) (*domain.KnowledgeVersion, error)
	RollbackToVersion(context.Context, *request.RollbackVersionRequest, string, bool) (*domain.Knowledge, error)
	GetKnowledgeBatchWithVersion(context.Context, *request.BatchGetKnowledgeWithVersionRequest, string, bool) ([]*KnowledgeWithVersion, error)
}

// KnowledgeWithVersion 将当前归属信息与指定版本快照分开返回。
type KnowledgeWithVersion struct {
	Knowledge     *domain.Knowledge
	Version       *domain.KnowledgeVersion
	Tags          []*domain.KnowledgebaseTag
	DirectoryName string
}

type knowledgeVersionServiceImpl struct {
	kbRepo      repository.KnowledgebaseRepository
	dirRepo     repository.KnowledgebaseDirectoryRepository
	knowRepo    repository.KnowledgeRepository
	versionRepo repository.KnowledgeVersionRepository
	tagRelRepo  repository.KnowledgeTagRelationRepository
	permission  PermissionService
	txManager   *database.TransactionManager
}

func NewKnowledgeVersionService(
	kbRepo repository.KnowledgebaseRepository,
	dirRepo repository.KnowledgebaseDirectoryRepository,
	knowRepo repository.KnowledgeRepository,
	versionRepo repository.KnowledgeVersionRepository,
	tagRelRepo repository.KnowledgeTagRelationRepository,
	permission PermissionService,
) KnowledgeVersionService {
	return &knowledgeVersionServiceImpl{
		kbRepo: kbRepo, dirRepo: dirRepo, knowRepo: knowRepo,
		versionRepo: versionRepo, tagRelRepo: tagRelRepo, permission: permission,
		txManager: database.NewTransactionManager(),
	}
}

// readableKnowledge 对每一条知识检查所属知识本的访问权限。
func (s *knowledgeVersionServiceImpl) readableKnowledge(ctx context.Context, tx *database.TxContext,
	id, operator string, isAdmin bool) (*domain.Knowledge, *domain.Knowledgebase, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if id == "" {
		return nil, nil, errors.ErrInvalidParam
	}
	knowledge, err := s.knowRepo.FindByUUID(tx, id)
	if err != nil {
		return nil, nil, err
	}
	if knowledge == nil {
		return nil, nil, errors.ErrKnowledgeNotFound
	}
	kb, err := s.kbRepo.FindByUUID(tx, knowledge.KnowledgebaseUUID)
	if err != nil {
		return nil, nil, err
	}
	if kb == nil {
		return nil, nil, errors.ErrKnowledgebaseNotFound
	}
	if !s.permission.CheckViewPermission(kb, kb.UUID, operator, isAdmin) {
		return nil, nil, errors.ErrNoPermission
	}
	return knowledge, kb, nil
}

func (s *knowledgeVersionServiceImpl) ListVersions(ctx context.Context, id, operator string, isAdmin bool) ([]*domain.KnowledgeVersion, error) {
	if _, _, err := s.readableKnowledge(ctx, nil, id, operator, isAdmin); err != nil {
		return nil, err
	}
	return s.versionRepo.ListByKnowledgeUUID(nil, id)
}

func (s *knowledgeVersionServiceImpl) BatchListVersions(ctx context.Context, ids []string, operator string, isAdmin bool) (map[string][]*domain.KnowledgeVersion, error) {
	if len(ids) == 0 || len(ids) > 100 {
		return nil, errors.ErrInvalidParam
	}
	for _, id := range ids {
		if _, _, err := s.readableKnowledge(ctx, nil, id, operator, isAdmin); err != nil {
			return nil, err
		}
	}
	return s.versionRepo.BatchListByKnowledgeUUIDs(nil, ids)
}

func (s *knowledgeVersionServiceImpl) GetVersion(ctx context.Context, id, version, operator string, isAdmin bool) (*domain.KnowledgeVersion, error) {
	if err := ValidateVersion(version); err != nil {
		return nil, err
	}
	if _, _, err := s.readableKnowledge(ctx, nil, id, operator, isAdmin); err != nil {
		return nil, err
	}
	return s.findVersion(nil, id, version)
}

func (s *knowledgeVersionServiceImpl) findVersion(tx *database.TxContext, id, version string) (*domain.KnowledgeVersion, error) {
	snapshot, err := s.versionRepo.FindByKnowledgeUUIDAndVersion(tx, id, version)
	if err != nil {
		return nil, err
	}
	if snapshot == nil {
		return nil, errors.New(errors.ErrCodeNotFound, "知识版本不存在")
	}
	return snapshot, nil
}

// RollbackToVersion 以历史快照生成新版本；不删除历史，也不改变当前目录和标签。
func (s *knowledgeVersionServiceImpl) RollbackToVersion(ctx context.Context, req *request.RollbackVersionRequest,
	operator string, isAdmin bool) (*domain.Knowledge, error) {
	if req == nil || req.KnowledgeUUID == "" || utf8.RuneCountInString(req.Remark) > 500 {
		return nil, errors.ErrInvalidParam
	}
	if err := ValidateVersion(req.Version); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var result *domain.Knowledge
	err := s.txManager.ExecuteInTransactionWithContext(ctx, func(tx *database.TxContext) error {
		knowledge, kb, err := s.readableKnowledge(ctx, tx, req.KnowledgeUUID, operator, isAdmin)
		if err != nil {
			return err
		}
		if err := s.permission.CheckEditPermissionWithResource(kb, kb.UUID, knowledge.CreatorUsername, operator, isAdmin); err != nil {
			return err
		}
		target, err := s.findVersion(tx, knowledge.UUID, req.Version)
		if err != nil {
			return err
		}
		if target.Title != knowledge.Title {
			exists, err := s.knowRepo.ExistsByDirectoryAndTitle(tx, kb.UUID, knowledge.DirectoryUUID, target.Title, knowledge.UUID)
			if err != nil {
				return err
			}
			if exists {
				return errors.ErrKnowledgeTitleDuplicate
			}
		}
		next := GenerateNextVersion(knowledge.CurrentVersion)
		if err := ValidateVersion(next); err != nil {
			return err
		}
		exists, err := s.versionRepo.ExistsByKnowledgeUUIDAndVersion(tx, knowledge.UUID, next)
		if err != nil {
			return err
		}
		if exists {
			return errors.New(errors.ErrCodeAlreadyExists, "生成的版本号已存在")
		}
		now := time.Now()
		knowledge.Title, knowledge.Content = target.Title, target.Content
		knowledge.FilePath, knowledge.FileSize, knowledge.FileFormat = target.FilePath, target.FileSize, target.FileFormat
		knowledge.Metadata, knowledge.CurrentVersion = target.Metadata, next
		knowledge.UpdaterUsername, knowledge.UpdatedAt = operator, now
		if err := s.knowRepo.Update(tx, knowledge); err != nil {
			return err
		}
		snapshot := *target
		snapshot.ID, snapshot.UUID = 0, uuid.New().String()
		snapshot.Version, snapshot.Operator = next, operator
		snapshot.Operation, snapshot.Remark, snapshot.CreatedAt = domain.VersionOperationRollback, req.Remark, now
		if err := s.versionRepo.Create(tx, &snapshot); err != nil {
			return err // 与主记录更新在同一事务中，失败则一起回滚。
		}
		result = knowledge
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetKnowledgeBatchWithVersion 按请求顺序返回；任一项无权限或不存在时整批失败。
// 未指定版本时使用主记录的 current_version，避免按创建时间猜测当前版本。
func (s *knowledgeVersionServiceImpl) GetKnowledgeBatchWithVersion(ctx context.Context,
	req *request.BatchGetKnowledgeWithVersionRequest, operator string, isAdmin bool) ([]*KnowledgeWithVersion, error) {
	if req == nil || len(req.Items) == 0 || len(req.Items) > 100 {
		return nil, errors.ErrInvalidParam
	}
	knowledgeByID := make(map[string]*domain.Knowledge)
	ids := make([]string, 0, len(req.Items))
	dirIDs := make([]string, 0)
	queries := make([]repository.KnowledgeVersionQuery, 0, len(req.Items))
	for _, item := range req.Items {
		knowledge := knowledgeByID[item.KnowledgeUUID]
		if knowledge == nil {
			var err error
			knowledge, _, err = s.readableKnowledge(ctx, nil, item.KnowledgeUUID, operator, isAdmin)
			if err != nil {
				return nil, err
			}
			knowledgeByID[item.KnowledgeUUID] = knowledge
			ids = append(ids, knowledge.UUID)
			if knowledge.DirectoryUUID != nil {
				dirIDs = append(dirIDs, *knowledge.DirectoryUUID)
			}
		}
		version := item.Version
		if version == "" {
			version = knowledge.CurrentVersion
		}
		if err := ValidateVersion(version); err != nil {
			return nil, err
		}
		queries = append(queries, repository.KnowledgeVersionQuery{KnowledgeUUID: knowledge.UUID, Version: version})
	}
	versions, err := s.versionRepo.BatchFindByQueries(nil, queries)
	if err != nil {
		return nil, err
	}
	byQuery := make(map[repository.KnowledgeVersionQuery]*domain.KnowledgeVersion, len(versions))
	for _, version := range versions {
		byQuery[repository.KnowledgeVersionQuery{KnowledgeUUID: version.KnowledgeUUID, Version: version.Version}] = version
	}
	// 先确认所有快照存在，再获取附加信息。
	for _, query := range queries {
		if byQuery[query] == nil {
			return nil, errors.New(errors.ErrCodeNotFound, "知识版本不存在")
		}
	}
	tags, err := s.tagRelRepo.FindTagsByKnowledgeUUIDs(nil, ids)
	if err != nil {
		return nil, err
	}
	directoryNames := make(map[string]string)
	if len(dirIDs) > 0 {
		directories, err := s.dirRepo.FindByUUIDs(nil, dirIDs)
		if err != nil {
			return nil, err
		}
		for _, directory := range directories {
			directoryNames[directory.UUID] = directory.Name
		}
	}
	result := make([]*KnowledgeWithVersion, 0, len(queries))
	for _, query := range queries {
		knowledge := knowledgeByID[query.KnowledgeUUID]
		item := &KnowledgeWithVersion{Knowledge: knowledge, Version: byQuery[query], Tags: tags[knowledge.UUID]}
		if knowledge.DirectoryUUID != nil {
			item.DirectoryName = directoryNames[*knowledge.DirectoryUUID]
		}
		result = append(result, item)
	}
	return result, nil
}
