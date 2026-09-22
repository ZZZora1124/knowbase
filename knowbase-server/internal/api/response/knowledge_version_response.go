package response

import "time"

type KnowledgeVersionItem struct {
	UUID      string    `json:"uuid"`
	Version   string    `json:"version"`
	Title     string    `json:"title"`
	Operator  string    `json:"operator"`
	Operation string    `json:"operation"`
	Remark    string    `json:"remark"`
	CreatedAt time.Time `json:"createdAt"`
}

type KnowledgeVersionListResponse struct {
	Versions []*KnowledgeVersionItem `json:"versions"`
}

type KnowledgeWithVersions struct {
	KnowledgeUUID string                  `json:"knowledgeUuid"`
	Versions      []*KnowledgeVersionItem `json:"versions"`
}

type BatchVersionsResponse struct {
	Data []*KnowledgeWithVersions `json:"data"`
}

type KnowledgeVersionDetailResponse struct {
	UUID          string    `json:"uuid"`
	KnowledgeUUID string    `json:"knowledgeUuid"`
	Version       string    `json:"version"`
	Title         string    `json:"title"`
	Content       string    `json:"content"`
	FilePath      string    `json:"filePath"`
	FileSize      int64     `json:"fileSize"`
	FileFormat    string    `json:"fileFormat"`
	Metadata      string    `json:"metadata"`
	Operator      string    `json:"operator"`
	Operation     string    `json:"operation"`
	Remark        string    `json:"remark"`
	CreatedAt     time.Time `json:"createdAt"`
}

type KnowledgeBasicInfo struct {
	UUID           string  `json:"uuid"`
	Title          string  `json:"title"`
	KnowledgeType  string  `json:"knowledgeType"`
	TemplateName   string  `json:"templateName"`
	CurrentVersion string  `json:"currentVersion"`
	DirectoryUUID  *string `json:"directoryUuid"`
	DirectoryName  string  `json:"directoryName"`
}

type TagItem struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

type KnowledgeWithVersionItem struct {
	Knowledge *KnowledgeBasicInfo             `json:"knowledge"`
	Version   *KnowledgeVersionDetailResponse `json:"version"`
	Tags      []*TagItem                      `json:"tags"`
}

type BatchKnowledgeWithVersionResponse struct {
	List []*KnowledgeWithVersionItem `json:"list"`
}
