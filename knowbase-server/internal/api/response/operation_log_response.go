package response

import "time"

type OperationLogResponse struct {
	KnowledgebaseUUID string                 `json:"knowledgebaseUuid"`
	Operator          string                 `json:"operator"`
	Agent             string                 `json:"agent"`
	Operation         string                 `json:"operation"`
	ResourceType      string                 `json:"resourceType"`
	ResourceUUID      string                 `json:"resourceUuid"`
	Detail            map[string]interface{} `json:"detail"`
	CreatedAt         time.Time              `json:"createdAt"`
}

type ListOperationLogsResponse struct {
	Items    []*OperationLogResponse `json:"items"`
	Total    int64                   `json:"total"`
	Page     int                     `json:"page"`
	PageSize int                     `json:"pageSize"`
}
