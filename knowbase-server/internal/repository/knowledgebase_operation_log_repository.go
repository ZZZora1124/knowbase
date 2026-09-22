package repository

import (
	"github.com/beego/beego/v2/client/orm"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/database"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
)

type OperationLogFilter struct {
	KnowledgebaseUUID string
	Operator          string
	Agent             string
	ResourceType      string
	ResourceUUID      string
	Page              int
	PageSize          int
}

type KnowledgebaseOperationLogRepository interface {
	Create(ctx *database.TxContext, entry *domain.KnowledgebaseOperationLog) error
	List(filter *OperationLogFilter) ([]*domain.KnowledgebaseOperationLog, int64, error)
}

type knowledgebaseOperationLogRepositoryImpl struct {
	ormer orm.Ormer
}

func NewKnowledgebaseOperationLogRepository(ormer orm.Ormer) KnowledgebaseOperationLogRepository {
	return &knowledgebaseOperationLogRepositoryImpl{ormer: ormer}
}

func (r *knowledgebaseOperationLogRepositoryImpl) Create(ctx *database.TxContext, entry *domain.KnowledgebaseOperationLog) error {
	var executor orm.QueryExecutor = r.ormer
	if ctx != nil && ctx.Ormer() != nil {
		executor = ctx.Ormer()
	}
	_, err := executor.Insert(entry)
	return err
}

func (r *knowledgebaseOperationLogRepositoryImpl) List(filter *OperationLogFilter) ([]*domain.KnowledgebaseOperationLog, int64, error) {
	if filter == nil {
		return nil, 0, errors.ErrInvalidParam
	}
	query := r.ormer.QueryTable(new(domain.KnowledgebaseOperationLog))
	for column, value := range map[string]string{
		"knowledgebase_uuid": filter.KnowledgebaseUUID,
		"operator":           filter.Operator,
		"agent":              filter.Agent,
		"resource_type":      filter.ResourceType,
		"resource_uuid":      filter.ResourceUUID,
	} {
		if value != "" {
			query = query.Filter(column, value)
		}
	}
	total, err := query.Count()
	if err != nil {
		return nil, 0, err
	}
	page, size := filter.Page, filter.PageSize
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		return nil, 0, errors.New(errors.ErrCodeInvalidParam, "每页最多查询100条操作日志")
	}
	items := make([]*domain.KnowledgebaseOperationLog, 0)
	_, err = query.OrderBy("-created_at", "-id").Limit(size, (page-1)*size).All(&items)
	return items, total, err
}
