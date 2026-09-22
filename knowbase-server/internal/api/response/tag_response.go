package response

import "time"

type TagResponse struct {
	UUID              string    `json:"uuid"`
	KnowledgebaseUUID string    `json:"knowledgebaseUuid"`
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	CreatedBy         string    `json:"createdBy"`
	KnowledgeCount    int64     `json:"knowledgeCount"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type ListTagsResponse struct {
	Total int64          `json:"total"`
	Items []*TagResponse `json:"items"`
}
