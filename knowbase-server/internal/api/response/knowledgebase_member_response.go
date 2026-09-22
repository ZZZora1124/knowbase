package response

import "time"

type MemberResponse struct {
	UUID              string    `json:"uuid"`
	KnowledgebaseUUID string    `json:"knowledgebaseUuid"`
	MemberType        string    `json:"memberType"`
	MemberValue       string    `json:"memberValue"`
	Role              string    `json:"role"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type ListMembersResponse struct {
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
	Items    []*MemberResponse `json:"items"`
}

type ExpandedMemberResponse struct {
	Username          string    `json:"username"`
	Role              string    `json:"role"`
	MemberSource      string    `json:"memberSource"`
	SourceName        string    `json:"sourceName"`
	KnowledgebaseUUID string    `json:"knowledgebaseUuid"`
	JoinedAt          time.Time `json:"joinedAt"`
}

type ListExpandedMembersResponse struct {
	Total    int64                     `json:"total"`
	Page     int                       `json:"page"`
	PageSize int                       `json:"pageSize"`
	Items    []*ExpandedMemberResponse `json:"items"`
}

type AdminResponse struct {
	MemberType  string `json:"memberType"`
	MemberValue string `json:"memberValue"`
	Role        string `json:"role"`
}

type ListAdminsResponse struct {
	Total int64            `json:"total"`
	Items []*AdminResponse `json:"items"`
}
