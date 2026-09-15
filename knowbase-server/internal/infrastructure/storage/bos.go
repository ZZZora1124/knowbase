// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package storage

import (
	"fmt"
	"github.com/baidubce/bce-sdk-go/bce"
	"github.com/baidubce/bce-sdk-go/services/bos"
	"github.com/baidubce/bce-sdk-go/services/bos/api"
)

// BOSClient BOS存储客户端
type BOSClient struct {
	client *bos.Client
	bucket string
}

// NewBOSClient 创建BOS客户端
func NewBOSClient(endpoint, ak, sk, bucket string) (*BOSClient, error) {
	client, err := bos.NewClient(ak, sk, endpoint)
	if err != nil {
		return nil, fmt.Errorf("创建BOS客户端失败: %w", err)
	}

	return &BOSClient{
		client: client,
		bucket: bucket,
	}, nil
}

// PutObject 上传对象到BOS
func (c *BOSClient) PutObject(objectKey string, fileData []byte, contentType string) (string, error) {
	// 创建文件流
	bodyStream, err := bce.NewBodyFromBytes(fileData)
	if err != nil {
		return "", fmt.Errorf("failed to create body from bytes: %v", err)
	}

	// 上传文件
	args := new(api.PutObjectArgs)
	args.ContentDisposition = "inline"
	_, err = c.client.PutObject(c.bucket, objectKey, bodyStream, args)
	if err != nil {
		return "", fmt.Errorf("上传对象失败: %w", err)
	}
	// 生成文件访问URL
	//fileURL := fmt.Sprintf("https://%s/%s", bosClient.Config.Endpoint, path.Join(bosConfig.Bucket, fileName))
	fileURL := c.client.BasicGeneratePresignedUrl(c.bucket, objectKey, -1)
	return fileURL, nil
}

// GetObject 从BOS获取对象
func (c *BOSClient) GetObject(objectKey string) (*api.GetObjectResult, error) {
	result, err := c.client.GetObject(c.bucket, objectKey, nil)
	if err != nil {
		return nil, fmt.Errorf("获取对象失败: %w", err)
	}
	return result, nil
}

// DeleteObject 从BOS删除对象
func (c *BOSClient) DeleteObject(objectKey string) error {
	err := c.client.DeleteObject(c.bucket, objectKey)
	if err != nil {
		return fmt.Errorf("删除对象失败: %w", err)
	}
	return nil
}

// InitiateMultipartUpload 初始化分片上传
func (c *BOSClient) InitiateMultipartUpload(objectKey, contentType string) (*api.InitiateMultipartUploadResult, error) {
	result, err := c.client.InitiateMultipartUpload(c.bucket, objectKey, contentType, nil)
	if err != nil {
		return nil, fmt.Errorf("初始化分片上传失败: %w", err)
	}
	return result, nil
}

// UploadPart 上传分片
func (c *BOSClient) UploadPart(objectKey, uploadID string, partNumber int, fileData []byte) (string, error) {
	// 创建文件流
	bodyStream, err := bce.NewBodyFromBytes(fileData)
	if err != nil {
		return "", fmt.Errorf("failed to create body from bytes: %v", err)
	}
	etag, err := c.client.UploadPart(c.bucket, objectKey, uploadID, partNumber, bodyStream, nil)
	if err != nil {
		return "", fmt.Errorf("上传分片失败: %w", err)
	}
	return etag, nil
}

// CompleteMultipartUpload 完成分片上传
func (c *BOSClient) CompleteMultipartUpload(objectKey, uploadID string, parts []api.UploadInfoType) error {
	_, err := c.client.CompleteMultipartUploadFromStruct(c.bucket, objectKey, uploadID, &api.CompleteMultipartUploadArgs{
		Parts: parts,
	})
	if err != nil {
		return fmt.Errorf("完成分片上传失败: %w", err)
	}
	return nil
}

// AbortMultipartUpload 取消分片上传
func (c *BOSClient) AbortMultipartUpload(objectKey, uploadID string) error {
	err := c.client.AbortMultipartUpload(c.bucket, objectKey, uploadID)
	if err != nil {
		return fmt.Errorf("取消分片上传失败: %w", err)
	}
	return nil
}
