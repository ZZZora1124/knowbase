package request

type BatchListVersionsRequest struct {
	KnowledgeUUIDs []string `json:"knowledgeUuids" binding:"required,min=1,max=100,dive,required"`
}

type RollbackVersionRequest struct {
	KnowledgeUUID string `json:"-"` // 由路由参数赋值，禁止请求体覆盖。
	Version       string `json:"version" binding:"required,max=50"`
	Remark        string `json:"remark" binding:"max=500"`
}

type KnowledgeVersionQueryItem struct {
	KnowledgeUUID string `json:"knowledgeUuid" binding:"required"`
	Version       string `json:"version" binding:"max=50"` // 空值表示当前版本。
}

type BatchGetKnowledgeWithVersionRequest struct {
	Items []KnowledgeVersionQueryItem `json:"items" binding:"required,min=1,max=100,dive"`
}
