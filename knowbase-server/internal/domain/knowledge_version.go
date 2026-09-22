package domain

import "time"

const (
	VersionOperationCreate   = "create"
	VersionOperationUpdate   = "update"
	VersionOperationRollback = "rollback"
)

// KnowledgeVersion 保存内容快照；目录、标签等关系仍以当前知识为准。
type KnowledgeVersion struct {
	ID            int64     `orm:"column(id);pk;auto" json:"-"`
	UUID          string    `orm:"column(uuid);size(36);unique" json:"uuid"`
	KnowledgeUUID string    `orm:"column(knowledge_uuid);size(36);index" json:"knowledgeUuid"`
	Version       string    `orm:"column(version);size(50)" json:"version"`
	Title         string    `orm:"column(title);size(500)" json:"title"`
	Content       string    `orm:"column(content);type(text);null" json:"content"`
	FilePath      string    `orm:"column(file_path);size(1024);null" json:"filePath"`
	FileSize      int64     `orm:"column(file_size);default(0)" json:"fileSize"`
	FileFormat    string    `orm:"column(file_format);size(50);null" json:"fileFormat"`
	Metadata      string    `orm:"column(metadata);type(json);null" json:"metadata"`
	Operator      string    `orm:"column(operator);size(100)" json:"operator"`
	Operation     string    `orm:"column(operation);size(20)" json:"operation"`
	Remark        string    `orm:"column(remark);size(500);null" json:"remark"`
	CreatedAt     time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"createdAt"`
}

func (*KnowledgeVersion) TableName() string { return "knowledge_version" }

func (*KnowledgeVersion) TableUnique() [][]string {
	return [][]string{{"KnowledgeUUID", "Version"}}
}
