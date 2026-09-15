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

// CreateKnowledgebase 创建知识方舟
func (s *NotebookService) CreateKnowledgebase(ctx context.Context, req *CreateKnowledgebaseRequest) (*Knowledgebase, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, types.NewDetailedError(types.ErrorTypeValidation, "failed to marshal request").
			WithDetail("rawError", err.Error())
	}

	apiURL := s.BuildURL(types.PathNotebookCreate)
	body, err := WithRetry(ctx, "create knowledgebase", func() ([]byte, error) {
		return s.HTTPClient.Post(ctx, apiURL, data)
	})
	if err != nil {
		return nil, err
	}

	var result Knowledgebase
	if err := s.ParseResponse(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListKnowledgebases 查询知识方舟列表
func (s *NotebookService) ListKnowledgebases(ctx context.Context, keyword, visibility, owner string,
	page, pageSize int) (*KnowledgebaseListResponse, error) {

	params := url.Values{}
	if keyword != "" {
		params.Set("keyword", keyword)
	}
	if visibility != "" {
		params.Set("visibility", visibility)
	}
	if owner != "" {
		params.Set("owner", owner)
	}
	params.Set("page", fmt.Sprintf("%d", page))
	params.Set("pageSize", fmt.Sprintf("%d", pageSize))

	apiURL := s.BuildURL(types.PathNotebookList) + "?" + params.Encode()
	body, err := WithRetry(ctx, "list knowledgebases", func() ([]byte, error) {
		return s.HTTPClient.Get(ctx, apiURL)
	})
	if err != nil {
		return nil, err
	}

	var result KnowledgebaseListResponse
	if err := s.ParseResponse(body, &result); err != nil {
		return nil, err
	}
	if visibility != "public" {
		filtered := make([]Knowledgebase, 0, len(result.List))
		for _, kb := range result.List {
			if kb.Visibility != "public" {
				filtered = append(filtered, kb)
			}
		}
		result.List = filtered
		//result.Total = int64(len(filtered))
	}
	return &result, nil
}

// GetKnowledgebaseByName 根据名称获取知识方舟
func (s *NotebookService) GetKnowledgebaseByName(ctx context.Context, name string) (*Knowledgebase, error) {
	apiURL := s.BuildURL(types.PathNotebookGetByName, url.PathEscape(name))
	body, err := WithRetry(ctx, "get knowledgebase by name", func() ([]byte, error) {
		return s.HTTPClient.Get(ctx, apiURL)
	})
	if err != nil {
		return nil, err
	}

	var result Knowledgebase
	if err := s.ParseResponse(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetKnowledgebaseByUUID 根据UUID获取知识方舟
func (s *NotebookService) GetKnowledgebaseByUUID(ctx context.Context, uuid string) (*Knowledgebase, error) {
	apiURL := s.BuildURL(types.PathNotebookGetByUUID, uuid)
	body, err := WithRetry(ctx, "get knowledgebase by uuid", func() ([]byte, error) {
		return s.HTTPClient.Get(ctx, apiURL)
	})
	if err != nil {
		return nil, err
	}

	var result Knowledgebase
	if err := s.ParseResponse(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateKnowledgebase 更新知识方舟
func (s *NotebookService) UpdateKnowledgebase(ctx context.Context, uuid string,
	req *UpdateKnowledgebaseRequest) (*Knowledgebase, error) {

	data, err := json.Marshal(req)
	if err != nil {
		return nil, types.NewDetailedError(types.ErrorTypeValidation, "failed to marshal request").
			WithDetail("rawError", err.Error())
	}

	apiURL := s.BuildURL(types.PathNotebookUpdate, uuid)
	body, err := WithRetry(ctx, "update knowledgebase", func() ([]byte, error) {
		return s.HTTPClient.Put(ctx, apiURL, data)
	})
	if err != nil {
		return nil, err
	}

	var result Knowledgebase
	if err := s.ParseResponse(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
