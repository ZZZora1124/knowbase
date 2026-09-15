// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package notebook

import (
	"context"
	"encoding/json"

	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

// CreateDirectory 创建目录
func (s *NotebookService) CreateDirectory(ctx context.Context, kbUUID string,
	req *CreateDirectoryRequest) (*Directory, error) {

	data, err := json.Marshal(req)
	if err != nil {
		return nil, types.NewDetailedError(types.ErrorTypeValidation, "failed to marshal request").
			WithDetail("rawError", err.Error())
	}

	apiURL := s.BuildURL(types.PathNotebookDirCreate, kbUUID)
	body, err := WithRetry(ctx, "create directory", func() ([]byte, error) {
		return s.HTTPClient.Post(ctx, apiURL, data)
	})
	if err != nil {
		return nil, err
	}

	var result Directory
	if err := s.ParseResponse(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListDirectories 查询目录列表
func (s *NotebookService) ListDirectories(ctx context.Context, kbUUID string) (*DirectoryListResponse, error) {
	apiURL := s.BuildURL(types.PathNotebookDirList, kbUUID)
	body, err := WithRetry(ctx, "list directories", func() ([]byte, error) {
		return s.HTTPClient.Get(ctx, apiURL)
	})
	if err != nil {
		return nil, err
	}

	var result DirectoryListResponse
	if err := s.ParseResponse(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetDirectory 获取目录详情
func (s *NotebookService) GetDirectory(ctx context.Context, kbUUID, dirUUID string) (*Directory, error) {
	apiURL := s.BuildURL(types.PathNotebookDirGet, kbUUID, dirUUID)
	body, err := WithRetry(ctx, "get directory", func() ([]byte, error) {
		return s.HTTPClient.Get(ctx, apiURL)
	})
	if err != nil {
		return nil, err
	}

	var result Directory
	if err := s.ParseResponse(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateDirectory 更新目录
func (s *NotebookService) UpdateDirectory(ctx context.Context, kbUUID, dirUUID string,
	req *UpdateDirectoryRequest) (*Directory, error) {

	data, err := json.Marshal(req)
	if err != nil {
		return nil, types.NewDetailedError(types.ErrorTypeValidation, "failed to marshal request").
			WithDetail("rawError", err.Error())
	}

	apiURL := s.BuildURL(types.PathNotebookDirUpdate, kbUUID, dirUUID)
	body, err := WithRetry(ctx, "update directory", func() ([]byte, error) {
		return s.HTTPClient.Put(ctx, apiURL, data)
	})
	if err != nil {
		return nil, err
	}

	var result Directory
	if err := s.ParseResponse(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteDirectory 删除目录（级联删除子目录及知识）
func (s *NotebookService) DeleteDirectory(ctx context.Context, kbUUID, dirUUID string) error {
	apiURL := s.BuildURL(types.PathNotebookDirDelete, kbUUID, dirUUID)
	body, err := WithRetry(ctx, "delete directory", func() ([]byte, error) {
		return s.HTTPClient.Delete(ctx, apiURL, nil)
	})
	if err != nil {
		return err
	}

	return s.ParseResponse(body, nil)
}

// GetDirectoryTree 获取目录树
func (s *NotebookService) GetDirectoryTree(ctx context.Context, kbUUID string) (*DirectoryTreeResponse, error) {
	apiURL := s.BuildURL(types.PathNotebookDirTree, kbUUID)
	body, err := WithRetry(ctx, "get directory tree", func() ([]byte, error) {
		return s.HTTPClient.Get(ctx, apiURL)
	})
	if err != nil {
		return nil, err
	}

	var result DirectoryTreeResponse
	if err := s.ParseResponse(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// MoveDirectory 移动目录
func (s *NotebookService) MoveDirectory(ctx context.Context, kbUUID, dirUUID string,
	req *MoveDirectoryRequest) (*Directory, error) {

	data, err := json.Marshal(req)
	if err != nil {
		return nil, types.NewDetailedError(types.ErrorTypeValidation, "failed to marshal request").
			WithDetail("rawError", err.Error())
	}

	apiURL := s.BuildURL(types.PathNotebookDirMove, kbUUID, dirUUID)
	body, err := WithRetry(ctx, "move directory", func() ([]byte, error) {
		return s.HTTPClient.Post(ctx, apiURL, data)
	})
	if err != nil {
		return nil, err
	}

	var result Directory
	if err := s.ParseResponse(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
