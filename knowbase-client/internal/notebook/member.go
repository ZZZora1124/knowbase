// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package notebook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

// AddMember 添加成员
func (s *NotebookService) AddMember(ctx context.Context, kbUUID string, req *AddMemberRequest) (*Member, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, types.NewDetailedError(types.ErrorTypeValidation, "failed to marshal request").
			WithDetail("rawError", err.Error())
	}

	apiURL := s.BuildURL(types.PathNotebookMemberAdd, kbUUID)
	body, err := WithRetry(ctx, "add member", func() ([]byte, error) {
		return s.HTTPClient.Post(ctx, apiURL, data)
	})
	if err != nil {
		return nil, err
	}

	var result Member
	if err := s.ParseResponse(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListMembers 查询成员列表
func (s *NotebookService) ListMembers(ctx context.Context, kbUUID string,
	page, pageSize int) (*MemberListResponse, error) {

	params := url.Values{}
	params.Set("page", fmt.Sprintf("%d", page))
	params.Set("pageSize", fmt.Sprintf("%d", pageSize))

	apiURL := s.BuildURL(types.PathNotebookMemberList, kbUUID) + "?" + params.Encode()
	body, err := WithRetry(ctx, "list members", func() ([]byte, error) {
		return s.HTTPClient.Get(ctx, apiURL)
	})
	if err != nil {
		return nil, err
	}

	var result MemberListResponse
	if err := s.ParseResponse(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateMemberRole 更新成员角色
func (s *NotebookService) UpdateMemberRole(ctx context.Context, kbUUID, memberUUID string,
	req *UpdateMemberRoleRequest) error {

	data, err := json.Marshal(req)
	if err != nil {
		return types.NewDetailedError(types.ErrorTypeValidation, "failed to marshal request").
			WithDetail("rawError", err.Error())
	}

	apiURL := s.BuildURL(types.PathNotebookMemberUpdateRole, kbUUID, memberUUID)
	body, err := WithRetry(ctx, "update member role", func() ([]byte, error) {
		return s.HTTPClient.Put(ctx, apiURL, data)
	})
	if err != nil {
		return err
	}

	return s.ParseResponse(body, nil)
}
