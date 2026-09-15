// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package loader

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/client"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/storage"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/utils"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

// KnowledgeBookLoader 知识方舟加载器
type KnowledgeBookLoader struct {
	*BaseLoader
	httpClient *client.HTTPClient
}

// NewKnowledgeBookLoader 创建知识方舟加载器
func NewKnowledgeBookLoader(cred *types.Credential) *KnowledgeBookLoader {
	return &KnowledgeBookLoader{
		BaseLoader: NewBaseLoader(cred),
		httpClient: client.NewHTTPClientWithCredential(cred),
	}
}

// Type 返回加载器类型
func (l *KnowledgeBookLoader) Type() string {
	return types.SourceTypeKnowledgeBook
}

// Load 加载知识方舟数据
func (l *KnowledgeBookLoader) Load(ctx context.Context, source types.SourceConfig) ([]types.RawKnowledge, error) {
	allKnowledge := make([]types.RawKnowledge, 0)

	for _, filter := range source.Filters {
		knowledge, err := l.loadByFilter(ctx, filter)
		if err != nil {
			return nil, fmt.Errorf("failed to load knowledge book for filter: %w", err)
		}
		allKnowledge = append(allKnowledge, knowledge...)

		// 如果指定了 knowledgebookUuid，需要查找子知识方舟
		if filter.KnowledgebookUuid != "" {
			subKnowledge, err := l.loadSubKnowledgeBooks(ctx, filter.KnowledgebookUuid, filter.KnowledgeType)
			if err != nil {
				utils.PrintlnError("Failed to load sub knowledge books: %v", err)
			} else {
				allKnowledge = append(allKnowledge, subKnowledge...)
			}
		}
	}

	return allKnowledge, nil
}

// loadByFilter 根据filter加载知识
func (l *KnowledgeBookLoader) loadByFilter(ctx context.Context, filter types.FilterConfig) ([]types.RawKnowledge, error) {
	// 校验：knowledgeUuids 和 knowledgebookUuid 二选一
	hasKnowledgeUuids := len(filter.KnowledgeUuids) > 0
	hasKnowledgebookUuid := filter.KnowledgebookUuid != ""

	if !hasKnowledgeUuids && !hasKnowledgebookUuid {
		return nil, types.NewDetailedError(types.ErrorTypeConfig, "knowledgebookUuid or knowledgeUuids is required").
			WithDetail("filter", "knowledge_book").
			WithSuggestion("请在配置文件中指定 knowledgebookUuid 或 knowledgeUuids 字段")
	}

	// 优先使用 knowledgeUuids（直接指定更精确）
	if hasKnowledgeUuids {
		return l.fetchKnowledgeByUuids(ctx, filter.KnowledgeUuids)
	}

	// 获取知识方舟名称（用于文件命名）
	knowledgeBaseName, err := l.getKnowledgeBaseName(ctx, filter.KnowledgebookUuid)
	if err != nil {
		utils.PrintlnError("Failed to get knowledge base name: %v", err)
		knowledgeBaseName = filter.KnowledgebookUuid // 使用UUID作为后备方案
	}

	// 使用 knowledgebookUuid 走原有逻辑
	var rawKnowledge []types.RawKnowledge
	if len(filter.TagNames) > 0 || len(filter.DirectoryNames) > 0 {
		// 使用接口2：根据标签、目录或知识类型查询
		rawKnowledge, err = l.fetchKnowledgeByTags(ctx, filter.KnowledgebookUuid, filter.TagNames, filter.DirectoryNames, filter.KnowledgeType)
	} else {
		// 使用接口1：查询所有知识
		rawKnowledge, err = l.fetchAllKnowledge(ctx, filter.KnowledgebookUuid, filter.KnowledgeType)
	}

	if err != nil {
		return nil, err
	}

	// 为每个知识项添加知识方舟名称、知识方舟UUID和知识类型信息
	for i := range rawKnowledge {
		if rawKnowledge[i].Data == nil {
			rawKnowledge[i].Data = make(map[string]interface{})
		}
		rawKnowledge[i].Data["knowledgeBaseName"] = knowledgeBaseName
		rawKnowledge[i].Data["knowledgebookUuid"] = filter.KnowledgebookUuid
		if filter.KnowledgeType != "" {
			rawKnowledge[i].Data["filterKnowledgeType"] = filter.KnowledgeType
		}
	}

	return rawKnowledge, nil
}

// loadSubKnowledgeBooks 加载子知识方舟的指定类型知识
func (l *KnowledgeBookLoader) loadSubKnowledgeBooks(ctx context.Context, parentKnowledgebookUuid string, knowledgeType string) ([]types.RawKnowledge, error) {
	utils.Debug("Loading sub knowledge books",
		zap.String("parentKnowledgebookUuid", parentKnowledgebookUuid),
		zap.String("knowledgeType", knowledgeType))

	// 查询类型为 config 且来源为 knowledge_book 的知识（子知识方舟）
	subKnowledgeBooks, err := l.fetchSubKnowledgeBooksConfig(ctx, parentKnowledgebookUuid)
	if err != nil {
		return nil, err
	}

	if len(subKnowledgeBooks) == 0 {
		utils.Debug("No sub knowledge books found")
		return nil, nil
	}

	utils.Debug("Found sub knowledge books", zap.Int("count", len(subKnowledgeBooks)))

	// 为每个子知识方舟加载指定类型的知识
	allSubKnowledge := make([]types.RawKnowledge, 0)
	for _, subKB := range subKnowledgeBooks {
		// 从 metadata 中获取子知识方舟的 UUID
		subKBUUID, ok := subKB.Metadata["knowledgebaseUuid"].(string)
		if !ok || subKBUUID == "" {
			utils.PrintlnError("Sub knowledge book missing knowledgebaseUuid in metadata")
			continue
		}

		// 获取子知识方舟名称
		subKBName, err := l.getKnowledgeBaseName(ctx, subKBUUID)
		if err != nil {
			utils.PrintlnError("Failed to get sub knowledge base name for %s: %v", subKBUUID, err)
			subKBName = subKBUUID
		}

		utils.Debug("Loading knowledge from sub knowledge book",
			zap.String("subKBUUID", subKBUUID),
			zap.String("subKBName", subKBName),
			zap.String("knowledgeType", knowledgeType))

		// 加载子知识方舟中指定类型的知识
		subKnowledge, err := l.fetchAllKnowledge(ctx, subKBUUID, knowledgeType)
		if err != nil {
			utils.PrintlnError("Failed to load knowledge from sub knowledge book %s: %v", subKBUUID, err)
			continue
		}

		// 为每个子知识方舟的知识项添加名称、UUID和类型信息
		for i := range subKnowledge {
			if subKnowledge[i].Data == nil {
				subKnowledge[i].Data = make(map[string]interface{})
			}
			subKnowledge[i].Data["knowledgeBaseName"] = subKBName
			subKnowledge[i].Data["knowledgebookUuid"] = subKBUUID
			subKnowledge[i].Data["filterKnowledgeType"] = knowledgeType
			subKnowledge[i].Data["isSubKnowledgeBook"] = true
		}

		allSubKnowledge = append(allSubKnowledge, subKnowledge...)
	}

	utils.Debug("Loaded sub knowledge books total", zap.Int("count", len(allSubKnowledge)))
	return allSubKnowledge, nil
}

// fetchSubKnowledgeBooksConfig 查询子知识方舟配置
func (l *KnowledgeBookLoader) fetchSubKnowledgeBooksConfig(ctx context.Context, knowledgebookUuid string) ([]KnowledgeItem, error) {
	allKnowledge := make([]KnowledgeItem, 0)
	page := 1
	pageSize := 20

	for {
		apiURL := fmt.Sprintf("%s%s?page=%d&pageSize=%d&knowledgeType=config&knowledgeSource=knowledge_book",
			types.GetKnowbaseHost(),
			fmt.Sprintf(types.PathKnowledgeBookList, knowledgebookUuid),
			page,
			pageSize)

		respBody, err := l.httpClient.Get(ctx, apiURL)
		if err != nil {
			return nil, err
		}

		var resp KnowledgeBookResponse
		if err := json.Unmarshal(respBody, &resp); err != nil {
			return nil, err
		}

		if resp.Code != 0 {
			return nil, fmt.Errorf("API returned error: %s", resp.Message)
		}

		if len(resp.Data.List) > 0 {
			allKnowledge = append(allKnowledge, resp.Data.List...)
		}

		if page*pageSize >= resp.Data.Total {
			break
		}
		page++
	}

	return allKnowledge, nil
}

// getKnowledgeBaseName 获取知识方舟名称
func (l *KnowledgeBookLoader) getKnowledgeBaseName(ctx context.Context, knowledgebookUuid string) (string, error) {
	apiURL := fmt.Sprintf("%s%s", types.GetKnowbaseHost(), fmt.Sprintf(types.PathNotebookGetByUUID, knowledgebookUuid))

	respBody, err := l.httpClient.Get(ctx, apiURL)
	if err != nil {
		return "", err
	}

	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Name string `json:"name"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", err
	}

	if resp.Code != 0 {
		return "", fmt.Errorf("API returned error: %s", resp.Message)
	}

	return resp.Data.Name, nil
}

// fetchAllKnowledge 获取知识方舟下的所有知识（支持分页）
func (l *KnowledgeBookLoader) fetchAllKnowledge(ctx context.Context, knowledgebookUuid string, knowledgeType string) ([]types.RawKnowledge, error) {
	allKnowledge := make([]types.RawKnowledge, 0)
	page := 1
	pageSize := 20

	utils.Debug("Fetching all knowledge from knowledge book",
		zap.String("knowledgebookUuid", knowledgebookUuid),
		zap.String("knowledgeType", knowledgeType))

	for {
		// 构建URL，如果指定了 knowledgeType，添加到查询参数
		apiURL := fmt.Sprintf("%s%s?page=%d&pageSize=%d",
			types.GetKnowbaseHost(),
			fmt.Sprintf(types.PathKnowledgeBookList, knowledgebookUuid),
			page,
			pageSize)
		if knowledgeType != "" {
			apiURL = fmt.Sprintf("%s&knowledgeType=%s", apiURL, knowledgeType)
		}

		utils.Debug("Getting knowledge book list",
			zap.String("url", apiURL),
			zap.Int("page", page))

		// 发送请求
		respBody, err := l.httpClient.Get(ctx, apiURL)
		if err != nil {
			if _, ok := err.(*types.DetailedError); ok {
				return nil, err
			}
			return nil, types.NewDetailedError(types.ErrorTypeNetwork, "failed to get knowledge book list").
				WithDetail("url", apiURL).
				WithDetail("knowledgebookUuid", knowledgebookUuid).
				WithDetail("rawError", err.Error()).
				WithSuggestion(utils.GetNetworkSuggestion(err))
		}

		// 解析响应
		var resp KnowledgeBookResponse
		if err := json.Unmarshal(respBody, &resp); err != nil {
			return nil, types.NewDetailedError(types.ErrorTypeValidation, "failed to parse knowledge book response").
				WithDetail("url", apiURL).
				WithDetail("knowledgebookUuid", knowledgebookUuid).
				WithDetail("rawError", err.Error()).
				WithDetail("responseBody", utils.TruncateString(string(respBody), 500)).
				WithSuggestion("API 响应格式错误，请稍后重试或联系管理员")
		}

		if resp.Code != 0 {
			return nil, types.NewDetailedError(types.ErrorTypeAPI, "API returned error").
				WithDetail("url", apiURL).
				WithDetail("knowledgebookUuid", knowledgebookUuid).
				WithDetail("statusCode", fmt.Sprintf("%d", resp.Code)).
				WithDetail("message", resp.Message).
				WithSuggestion(utils.GetAPISuggestion(resp.Code))
		}

		// 转换数据：过滤掉 config 类型的知识，其他直接使用
		for _, k := range resp.Data.List {
			// 排除 config 类型的知识
			if k.KnowledgeType == "config" {
				continue
			}
			allKnowledge = append(allKnowledge, l.convertToRawKnowledge(k))
		}

		utils.Debug("Loaded knowledge book page",
			zap.Int("page", page),
			zap.Int("total", resp.Data.Total),
			zap.Int("pageItems", len(resp.Data.List)))

		// 检查是否还有更多数据
		if page*pageSize >= resp.Data.Total {
			break
		}
		page++
	}

	utils.Debug("Loaded all knowledge from knowledge book",
		zap.String("knowledgebookUuid", knowledgebookUuid),
		zap.Int("count", len(allKnowledge)))

	return allKnowledge, nil
}

// fetchKnowledgeByTags 根据标签或目录查询知识（支持分页）
func (l *KnowledgeBookLoader) fetchKnowledgeByTags(ctx context.Context, knowledgebookUuid string,
	tagNames []string, directoryNames []string, knowledgeType string) ([]types.RawKnowledge, error) {
	allKnowledge := make([]types.RawKnowledge, 0)
	page := 1
	pageSize := 20

	utils.Debug("Fetching knowledge by tags/directories",
		zap.String("knowledgebookUuid", knowledgebookUuid),
		zap.Strings("tagNames", tagNames),
		zap.Strings("directoryNames", directoryNames),
		zap.String("knowledgeType", knowledgeType))

	for {
		// 构建URL
		apiURL := fmt.Sprintf("%s%s",
			types.GetKnowbaseHost(),
			fmt.Sprintf(types.PathKnowledgeBookByTags, knowledgebookUuid))

		// 构建请求body
		reqBody := map[string]interface{}{
			"page":     page,
			"pageSize": pageSize,
		}
		if len(tagNames) > 0 {
			reqBody["tagNames"] = tagNames
		}
		if len(directoryNames) > 0 {
			reqBody["directoryNames"] = directoryNames
		}
		if knowledgeType != "" {
			reqBody["knowledgeType"] = knowledgeType
		}

		reqBytes, err := json.Marshal(reqBody)
		if err != nil {
			return nil, types.NewDetailedError(types.ErrorTypeValidation, "failed to marshal request body").
				WithDetail("rawError", err.Error())
		}

		utils.Debug("Posting knowledge book by tags",
			zap.String("url", apiURL),
			zap.Int("page", page))

		// 发送POST请求
		respBody, err := l.httpClient.Post(ctx, apiURL, reqBytes)
		if err != nil {
			if _, ok := err.(*types.DetailedError); ok {
				return nil, err
			}
			return nil, types.NewDetailedError(types.ErrorTypeNetwork, "failed to get knowledge book by tags").
				WithDetail("url", apiURL).
				WithDetail("knowledgebookUuid", knowledgebookUuid).
				WithDetail("rawError", err.Error()).
				WithSuggestion(utils.GetNetworkSuggestion(err))
		}

		// 解析响应
		var resp KnowledgeBookResponse
		if err := json.Unmarshal(respBody, &resp); err != nil {
			return nil, types.NewDetailedError(types.ErrorTypeValidation, "failed to parse knowledge book response").
				WithDetail("url", apiURL).
				WithDetail("knowledgebookUuid", knowledgebookUuid).
				WithDetail("rawError", err.Error()).
				WithDetail("responseBody", utils.TruncateString(string(respBody), 500)).
				WithSuggestion("API 响应格式错误，请稍后重试或联系管理员")
		}

		if resp.Code != 0 {
			return nil, types.NewDetailedError(types.ErrorTypeAPI, "API returned error").
				WithDetail("url", apiURL).
				WithDetail("knowledgebookUuid", knowledgebookUuid).
				WithDetail("statusCode", fmt.Sprintf("%d", resp.Code)).
				WithDetail("message", resp.Message).
				WithSuggestion(utils.GetAPISuggestion(resp.Code))
		}

		// 转换数据：过滤掉 config 类型的知识，其他直接使用
		for _, k := range resp.Data.List {
			// 排除 config 类型的知识
			if k.KnowledgeType == "config" {
				continue
			}
			allKnowledge = append(allKnowledge, l.convertToRawKnowledge(k))
		}

		utils.Debug("Loaded knowledge book page by tags",
			zap.Int("page", page),
			zap.Int("total", resp.Data.Total),
			zap.Int("pageItems", len(resp.Data.List)))

		// 检查是否还有更多数据
		if page*pageSize >= resp.Data.Total {
			break
		}
		page++
	}

	utils.Debug("Loaded knowledge by tags/directories",
		zap.String("knowledgebookUuid", knowledgebookUuid),
		zap.Int("count", len(allKnowledge)))

	return allKnowledge, nil
}

// fetchKnowledgeByUuids 根据知识UUID列表批量获取知识
func (l *KnowledgeBookLoader) fetchKnowledgeByUuids(ctx context.Context, knowledgeUuids []string) ([]types.RawKnowledge, error) {
	utils.Debug("Fetching knowledge by UUIDs",
		zap.Strings("knowledgeUuids", knowledgeUuids))

	// 构建URL
	apiURL := fmt.Sprintf("%s%s", types.GetKnowbaseHost(), types.PathKnowledgeBatch)

	// 构建请求body
	reqBody := map[string]interface{}{
		"uuids": knowledgeUuids,
	}
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, types.NewDetailedError(types.ErrorTypeValidation, "failed to marshal request body").
			WithDetail("rawError", err.Error())
	}

	utils.Debug("Posting knowledge batch request",
		zap.String("url", apiURL),
		zap.Int("uuidCount", len(knowledgeUuids)))

	// 发送POST请求
	respBody, err := l.httpClient.Post(ctx, apiURL, reqBytes)
	if err != nil {
		if _, ok := err.(*types.DetailedError); ok {
			return nil, err
		}
		return nil, types.NewDetailedError(types.ErrorTypeNetwork, "failed to get knowledge by uuids").
			WithDetail("url", apiURL).
			WithDetail("rawError", err.Error()).
			WithSuggestion(utils.GetNetworkSuggestion(err))
	}

	// 解析响应
	var resp KnowledgeBatchResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, types.NewDetailedError(types.ErrorTypeValidation, "failed to parse knowledge batch response").
			WithDetail("url", apiURL).
			WithDetail("rawError", err.Error()).
			WithDetail("responseBody", utils.TruncateString(string(respBody), 500)).
			WithSuggestion("API 响应格式错误，请稍后重试或联系管理员")
	}

	if resp.Code != 0 {
		return nil, types.NewDetailedError(types.ErrorTypeAPI, "API returned error").
			WithDetail("url", apiURL).
			WithDetail("statusCode", fmt.Sprintf("%d", resp.Code)).
			WithDetail("message", resp.Message).
			WithSuggestion(utils.GetAPISuggestion(resp.Code))
	}

	// 转换数据：过滤掉 config 类型的知识，其他直接使用
	allKnowledge := make([]types.RawKnowledge, 0, len(resp.Data.List))
	// 缓存 knowledgebaseUuid -> name，避免重复调用 API
	kbNameCache := make(map[string]string)
	for _, k := range resp.Data.List {
		// 排除 config 类型的知识
		if k.KnowledgeType == "config" {
			continue
		}
		raw := l.convertToRawKnowledge(k)
		// 获取知识方舟名称并注入
		if k.KnowledgebaseUuid != "" {
			if name, ok := kbNameCache[k.KnowledgebaseUuid]; ok {
				raw.Data["knowledgeBaseName"] = name
			} else {
				name, err := l.getKnowledgeBaseName(ctx, k.KnowledgebaseUuid)
				if err != nil {
					utils.PrintlnError("Failed to get knowledge base name for %s: %v", k.KnowledgebaseUuid, err)
					name = k.KnowledgebaseUuid // 降级为 UUID
				}
				kbNameCache[k.KnowledgebaseUuid] = name
				raw.Data["knowledgeBaseName"] = name
			}
		}
		allKnowledge = append(allKnowledge, raw)
	}

	utils.Debug("Loaded knowledge by UUIDs",
		zap.Int("requestedCount", len(knowledgeUuids)),
		zap.Int("returnedCount", len(resp.Data.List)))

	return allKnowledge, nil
}

// convertToRawKnowledge 将API响应的知识项转换为RawKnowledge
func (l *KnowledgeBookLoader) convertToRawKnowledge(k KnowledgeItem) types.RawKnowledge {
	return types.RawKnowledge{
		SourceType: types.SourceTypeKnowledgeBook,
		Data: map[string]interface{}{
			"knowledgebookUuid": k.KnowledgebaseUuid,
			"title":             k.Title,
			"content":           k.Content,
			"knowledgeType":     k.KnowledgeType,
			"knowledgeSource":   k.KnowledgeSource,
			"metadata":          k.Metadata,
			"createdAt":         k.CreatedAt,
			"updatedAt":         k.UpdatedAt,
			"uuid":              k.UUID,
			"dirPath":           k.DirPath,
		},
	}
}

// Convert 转换格式
func (l *KnowledgeBookLoader) Convert(sourceType string, raw []types.RawKnowledge, output types.OutputConfig) ([]types.KnowledgeItem, error) {
	// knowledge_book不需要转换,直接将RawKnowledge转换为KnowledgeItem
	items := make([]types.KnowledgeItem, 0, len(raw))
	for _, r := range raw {
		item := types.KnowledgeItem{
			ID:      fmt.Sprintf("%v_%v", r.Data["title"], r.Data["createdAt"]),
			Title:   fmt.Sprintf("%v", r.Data["title"]),
			Content: fmt.Sprintf("%v", r.Data["content"]),
			Metadata: map[string]interface{}{
				"title":               r.Data["title"],
				"content":             r.Data["content"],
				"knowledgeType":       r.Data["knowledgeType"],
				"knowledgeSource":     r.Data["knowledgeSource"],
				"metadata":            r.Data["metadata"],
				"createdAt":           r.Data["createdAt"],
				"updatedAt":           r.Data["updatedAt"],
				"knowledgeBaseName":   r.Data["knowledgeBaseName"],
				"filterKnowledgeType": r.Data["filterKnowledgeType"],
				"isSubKnowledgeBook":  r.Data["isSubKnowledgeBook"],
				"uuid":                r.Data["uuid"],
				"knowledgebookUuid":   r.Data["knowledgebookUuid"],
				"dirPath":             r.Data["dirPath"],
			},
		}
		items = append(items, item)
	}
	return items, nil
}

// Store 存储到本地，返回存储的知识项详细信息列表
func (l *KnowledgeBookLoader) Store(ctx context.Context, items []types.KnowledgeItem, storage *storage.LocalStorage, source types.SourceConfig) ([]types.StoredKnowledgeInfo, error) {
	// 确保子目录存在
	subDir, err := storage.EnsureSubDir(filepath.Join(types.SourceTypeKnowledgeBook, "custom"))
	if err != nil {
		return nil, types.NewDetailedError(types.ErrorTypeStorage, "failed to create sub directory").
			WithDetail("subDir", filepath.Join(types.SourceTypeKnowledgeBook, "custom")).
			WithDetail("rawError", err.Error()).
			WithSuggestion("请检查存储路径的权限和磁盘空间")
	}

	// 保存每个知识项并收集信息
	storedInfos := make([]types.StoredKnowledgeInfo, 0, len(items))
	for _, item := range items {
		item.Title = item.Metadata["title"].(string)
		item.Content = item.Metadata["content"].(string)
		item.CreateTime = item.Metadata["createdAt"].(string)
		item.UpdateTime = item.Metadata["updatedAt"].(string)
		knowledgeType := fmt.Sprintf("%v", item.Metadata["knowledgeType"])

		// 处理不同类型的知识
		if knowledgeType == "content" || knowledgeType == "readme" {
			// 文本内容类型：保存为.md文件
			if err := l.storeContentKnowledge(item, storage, &storedInfos); err != nil {
				utils.PrintlnError("Failed to store content knowledge %s: %v", item.Title, err.Error())
				continue
			}
		} else {
			// 文件类型：下载文件
			if err := l.storeFileKnowledge(item, storage, &storedInfos); err != nil {
				utils.PrintlnError("Failed to store file knowledge %s: %v", item.Title, err.Error())
				continue
			}
		}
	}

	fmt.Printf("Stored knowledge book items, base directory: %s, count: %d\n", subDir, len(storedInfos))

	return storedInfos, nil
}

// buildKnowledgeStorePath 构建知识存储路径
// 路径规则：knowledge_book/custom/{kbName}/{dirPath}/{filename}
// kbName 或 dirPath 为空时对应层级省略
func buildKnowledgeStorePath(knowledgeBaseName, dirPath, filename string) string {
	base := filepath.Join(types.SourceTypeKnowledgeBook, "custom")

	if knowledgeBaseName != "" {
		base = filepath.Join(base, utils.SanitizeFilename(knowledgeBaseName))
	}

	if dirPath != "" {
		// 对每个路径分段单独处理，防止路径穿越
		for _, seg := range strings.Split(filepath.ToSlash(dirPath), "/") {
			// 显式过滤 ".." 和 "." 防止路径穿越
			if seg == ".." || seg == "." || seg == "" {
				continue
			}
			sanitized := utils.SanitizeFilename(seg)
			if sanitized != "" {
				base = filepath.Join(base, sanitized)
			}
		}
	}

	return filepath.Join(base, filename)
}

// storeContentKnowledge 存储文本内容类型的知识
func (l *KnowledgeBookLoader) storeContentKnowledge(item types.KnowledgeItem, storage *storage.LocalStorage,
	storedInfos *[]types.StoredKnowledgeInfo) error {
	var filename, filePath string

	// 获取知识项的实际类型（从API返回的）
	itemKnowledgeType, hasItemType := item.Metadata["knowledgeType"].(string)
	// 获取配置中指定的过滤类型
	filterKnowledgeType, hasFilterType := item.Metadata["filterKnowledgeType"].(string)
	knowledgeBaseName, hasKBName := item.Metadata["knowledgeBaseName"].(string)

	// 判断是否使用特殊命名规则：
	// 1. 配置中指定了 filterKnowledgeType = readme，且有 knowledgeBaseName
	// 2. 或者知识项的实际类型是 "readme"，且有 knowledgeBaseName
	useSpecialNaming := (hasFilterType && filterKnowledgeType == "readme" && hasKBName && knowledgeBaseName != "") ||
		(hasItemType && itemKnowledgeType == "readme" && hasKBName && knowledgeBaseName != "")

	if useSpecialNaming {
		typeName := itemKnowledgeType
		if typeName == "" {
			typeName = filterKnowledgeType
		}
		// 使用格式: 知识方舟名称-knowledgeType.md，保持原有行为不受 DirPath 影响
		sanitizedKBName := utils.SanitizeFilename(knowledgeBaseName)
		filename = fmt.Sprintf("%s-%s", sanitizedKBName, typeName)
		filePath = filepath.Join(types.SourceTypeKnowledgeBook, "custom", filename+types.FileSuffixMarkdown)
	} else {
		// 使用标题作为文件名
		filename = utils.SanitizeFilename(item.Title)
		if filename == "" {
			filename = item.ID
		}
		dirPath, _ := item.Metadata["dirPath"].(string)
		filePath = buildKnowledgeStorePath(knowledgeBaseName, dirPath, filename+types.FileSuffixMarkdown)
	}

	// 写入文件
	content := fmt.Sprintf("%v", item.Metadata["content"])
	if err := storage.WriteFile(filePath, []byte(content), false); err != nil {
		return err
	}

	// 收集存储信息
	fullPath := storage.GetFullPath(filePath)
	kbUUID, _ := item.Metadata["knowledgebookUuid"].(string)
	knowledgeUUID, _ := item.Metadata["uuid"].(string)
	desc := fmt.Sprintf("知识uuid:%s, 知识方舟 uuid:%s", knowledgeUUID, kbUUID)
	if hasKBName && knowledgeBaseName != "" {
		desc += fmt.Sprintf(", 知识方舟名称:%s", knowledgeBaseName)
	}
	desc += fmt.Sprintf(", 创建时间:%s, 最近修改时间:%s", item.CreateTime, item.UpdateTime)
	*storedInfos = append(*storedInfos, types.StoredKnowledgeInfo{
		Name:        item.Title,
		Description: desc,
		FullPath:    fullPath,
		RelPath:     filePath,
	})

	utils.Debug("Saved content knowledge", zap.String("file", filePath))
	return nil
}

// storeFileKnowledge 存储文件类型的知识
func (l *KnowledgeBookLoader) storeFileKnowledge(item types.KnowledgeItem, storage *storage.LocalStorage,
	storedInfos *[]types.StoredKnowledgeInfo) error {
	// 获取metadata
	metadata, ok := item.Metadata["metadata"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid metadata format")
	}

	// 获取文件URL
	fileURL, ok := metadata["fileUrl"].(string)
	if !ok || fileURL == "" {
		return fmt.Errorf("fileUrl not found in metadata")
	}

	// 获取原始文件名
	originalName, ok := metadata["originalName"].(string)
	if !ok || originalName == "" {
		originalName = item.Title
	}

	// 清理文件名
	filename := utils.SanitizeFilename(originalName)
	if filename == "" {
		filename = item.ID
	}

	// 构建文件路径
	kbName, _ := item.Metadata["knowledgeBaseName"].(string)
	dirPath, _ := item.Metadata["dirPath"].(string)
	filePath := buildKnowledgeStorePath(kbName, dirPath, filename)

	// 下载文件
	fileContent, err := l.httpClient.Get(context.Background(), fileURL)
	if err != nil {
		return types.NewDetailedError(types.ErrorTypeNetwork, "failed to download file").
			WithDetail("url", fileURL).
			WithDetail("filename", filename).
			WithDetail("rawError", err.Error()).
			WithSuggestion(utils.GetNetworkSuggestion(err))
	}

	// 写入文件
	if err := storage.WriteFile(filePath, fileContent, false); err != nil {
		return err
	}

	// 收集存储信息
	fullPath := storage.GetFullPath(filePath)
	hasKBName := kbName != ""
	kbUUID, _ := item.Metadata["knowledgebookUuid"].(string)
	knowledgeUUID, _ := item.Metadata["uuid"].(string)
	desc := fmt.Sprintf("知识uuid:%s, 知识方舟 uuid:%s", knowledgeUUID, kbUUID)
	if hasKBName && kbName != "" {
		desc += fmt.Sprintf(", 知识方舟名称:%s", kbName)
	}
	desc += fmt.Sprintf(", 创建时间:%s, 最近修改时间:%s", item.CreateTime, item.UpdateTime)
	*storedInfos = append(*storedInfos, types.StoredKnowledgeInfo{
		Name:        originalName,
		Description: desc,
		FullPath:    fullPath,
		RelPath:     filePath,
	})

	utils.Debug("Saved file knowledge", zap.String("file", filePath))
	return nil
}

// KnowledgeBookResponse 知识方舟API响应
type KnowledgeBookResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		List     []KnowledgeItem `json:"list"`
		Total    int             `json:"total"`
		Page     int             `json:"page"`
		PageSize int             `json:"pageSize"`
	} `json:"data"`
}

// KnowledgeBatchResponse 批量获取知识API响应
type KnowledgeBatchResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		List []KnowledgeItem `json:"list"`
	} `json:"data"`
}

// KnowledgeItem 知识项
type KnowledgeItem struct {
	UUID              string                 `json:"uuid"`
	KnowledgebaseUuid string                 `json:"knowledgebaseUuid"`
	DirPath           string                 `json:"dirPath"`
	CreatorUsername   string                 `json:"creatorUsername"`
	Title             string                 `json:"title"`
	Content           string                 `json:"content"`
	Metadata          map[string]interface{} `json:"metadata"`
	KnowledgeType     string                 `json:"knowledgeType"`
	KnowledgeSource   string                 `json:"knowledgeSource"`
	CreatedAt         string                 `json:"createdAt"`
	UpdatedAt         string                 `json:"updatedAt"`
}
