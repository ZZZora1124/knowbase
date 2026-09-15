// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package notebook

import (
	"context"
	"encoding/json"

	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

// CreateTag 创建标签
func (s *NotebookService) CreateTag(ctx context.Context, kbUUID string, req *CreateTagRequest) (*Tag, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, types.NewDetailedError(types.ErrorTypeValidation, "failed to marshal request").
			WithDetail("rawError", err.Error())
	}

	apiURL := s.BuildURL(types.PathNotebookTagCreate, kbUUID)
	body, err := WithRetry(ctx, "create tag", func() ([]byte, error) {
		return s.HTTPClient.Post(ctx, apiURL, data)
	})
	if err != nil {
		return nil, err
	}

	var result Tag
	if err := s.ParseResponse(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListTags 查询标签列表
func (s *NotebookService) ListTags(ctx context.Context, kbUUID string) (*TagListResponse, error) {
	apiURL := s.BuildURL(types.PathNotebookTagList, kbUUID)
	body, err := WithRetry(ctx, "list tags", func() ([]byte, error) {
		return s.HTTPClient.Get(ctx, apiURL)
	})
	if err != nil {
		return nil, err
	}

	var result TagListResponse
	if err := s.ParseResponse(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateTag 更新标签
func (s *NotebookService) UpdateTag(ctx context.Context, kbUUID, tagUUID string,
	req *UpdateTagRequest) (*Tag, error) {

	data, err := json.Marshal(req)
	if err != nil {
		return nil, types.NewDetailedError(types.ErrorTypeValidation, "failed to marshal request").
			WithDetail("rawError", err.Error())
	}

	apiURL := s.BuildURL(types.PathNotebookTagUpdate, kbUUID, tagUUID)
	body, err := WithRetry(ctx, "update tag", func() ([]byte, error) {
		return s.HTTPClient.Put(ctx, apiURL, data)
	})
	if err != nil {
		return nil, err
	}

	var result Tag
	if err := s.ParseResponse(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
