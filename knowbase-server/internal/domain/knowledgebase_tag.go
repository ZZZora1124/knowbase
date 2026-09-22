package domain

import "time"

type KnowledgebaseTag struct {
	ID                int64      `orm:"column(id);pk;auto" json:"-"`
	UUID              string     `orm:"column(uuid);size(36);unique" json:"uuid"`
	KnowledgebaseUUID string     `orm:"column(knowledgebase_uuid);size(36);index" json:"knowledgebaseUuid"`
	Name              string     `orm:"column(name);size(100)" json:"name"`
	Description       string     `orm:"column(description);size(500);null" json:"description"`
	CreatedBy         string     `orm:"column(created_by);size(100)" json:"createdBy"`
	CreatedAt         time.Time  `orm:"column(created_at);auto_now_add;type(datetime)" json:"createdAt"`
	UpdatedAt         time.Time  `orm:"column(updated_at);auto_now;type(datetime)" json:"updatedAt"`
	DeletedAt         *time.Time `orm:"column(deleted_at);null" json:"-"`
}

func (*KnowledgebaseTag) TableName() string { return "knowledgebase_tag" }

func (*KnowledgebaseTag) TableUnique() [][]string {
	return [][]string{{"KnowledgebaseUUID", "Name"}}
}
