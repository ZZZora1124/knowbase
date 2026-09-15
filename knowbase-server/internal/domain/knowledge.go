// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package domain

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// Knowledge 知识模型
type Knowledge struct {
	ID                int64      `orm:"column(id);pk;auto" json:"-"`
	UUID              string     `orm:"column(uuid);size(36);unique" json:"uuid"`
	KnowledgebaseUUID string     `orm:"column(knowledgebase_uuid);size(36);index" json:"knowledgebaseUuid"`
	DirectoryUUID     *string    `orm:"column(directory_uuid);size(36);null;index" json:"directoryUuid"`
	CreatorUsername   string     `orm:"column(creator_username);size(100);index" json:"creatorUsername"`
	UpdaterUsername   string     `orm:"column(updater_username);size(100);index" json:"updaterUsername,omitempty"`
	Title             string     `orm:"column(title);size(500)" json:"title"`
	KnowledgeType     string     `orm:"column(knowledge_type);size(20);index" json:"knowledgeType"`
	TemplateName      string     `orm:"column(template_name);size(20);default('')" json:"templateName"`
	KnowledgeSource   string     `orm:"column(knowledge_source);size(20);default('custom');index" json:"knowledgeSource"`
	Content           string     `orm:"column(content);type(text);null" json:"content,omitempty"`
	FilePath          string     `orm:"column(file_path);size(1024);null" json:"filePath,omitempty"`
	FileSize          int64      `orm:"column(file_size);default(0)" json:"fileSize,omitempty"`
	FileFormat        string     `orm:"column(file_format);size(50);null" json:"fileFormat,omitempty"`
	SourceMethod      string     `orm:"column(source_method);size(100);default('manual')" json:"sourceMethod"`
	Metadata          string     `orm:"column(metadata);type(json);null" json:"metadata,omitempty"`
	CurrentVersion    string     `orm:"column(current_version);size(50);default(v1)" json:"currentVersion"`
	CreatedAt         time.Time  `orm:"column(created_at);auto_now_add;type(datetime)" json:"createdAt"`
	UpdatedAt         time.Time  `orm:"column(updated_at);auto_now;type(datetime)" json:"updatedAt"`
	DeletedAt         *time.Time `orm:"column(deleted_at);null" json:"-"`
}

// TableName 指定表名
func (m *Knowledge) TableName() string {
	return "knowledge"
}

// IsCustomKnowledge 是否为自定义知识
func (m *Knowledge) IsCustomKnowledge() bool {
	return m.KnowledgeSource == "custom"
}

// IsConfigType 是否为配置类型
func (m *Knowledge) IsConfigType() bool {
	return m.KnowledgeType == "config"
}

// IsFileType 是否为文件类型
func (m *Knowledge) IsFileType() bool {
	return m.KnowledgeType == "text_file" ||
		m.KnowledgeType == "audio_file" ||
		m.KnowledgeType == "image_file" ||
		m.KnowledgeType == "video_file"
}

// JSONMap 用于存储JSON元数据
type JSONMap map[string]interface{}

// Value 实现 driver.Valuer 接口
func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan 实现 sql.Scanner 接口
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	result := make(JSONMap)
	if err := json.Unmarshal(bytes, &result); err != nil {
		return err
	}
	*j = result
	return nil
}
