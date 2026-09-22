package domain

import "time"

type KnowledgebaseOperationLog struct {
	ID                int64     `orm:"column(id);pk;auto" json:"-"`
	KnowledgebaseUUID string    `orm:"column(knowledgebase_uuid);size(36);index" json:"knowledgebaseUuid"`
	Operator          string    `orm:"column(operator);size(100);index" json:"operator"`
	Agent             string    `orm:"column(agent);size(32);null;index" json:"agent"`
	Operation         string    `orm:"column(operation);size(100);index" json:"operation"`
	ResourceType      string    `orm:"column(resource_type);size(50)" json:"resourceType"`
	ResourceUUID      string    `orm:"column(resource_uuid);size(100);null" json:"resourceUuid"`
	Detail            string    `orm:"column(detail);type(json);null" json:"detail"`
	CreatedAt         time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"createdAt"`
}

func (*KnowledgebaseOperationLog) TableName() string { return "knowledgebase_operation_log" }
