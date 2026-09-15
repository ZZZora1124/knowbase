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
	"sync"

	"go.uber.org/zap"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/client"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/storage"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/utils"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

// 并发配置
const (
	MaxConcurrentDocFetch = 10 // 最大并发获取文档数
)

// docWithPath 携带层级路径的文档信息
type docWithPath struct {
	URL     string
	DocPath string // 层级路径，如 "父文档名/子文档名"，父文档自身为空
	DocGuid string
	DocName string
}

// docFetchTask 文档获取任务
type docFetchTask struct {
	URL     string
	DocPath string // 层级路径，用于传递到 RawKnowledge.Data["path"]
	Index   int    // 保持原始顺序
}

// docFetchResult 文档获取结果
type docFetchResult struct {
	Doc   types.RawKnowledge
	Index int
	Err   error
}

// KuDocLoader 知识库文档加载器
type KuDocLoader struct {
	*BaseLoader
	httpClient *client.HTTPClient
}

// NewKuDocLoader 创建知识库文档加载器
func NewKuDocLoader(cred *types.Credential) *KuDocLoader {
	return &KuDocLoader{
		BaseLoader: NewBaseLoader(cred),
		httpClient: client.NewHTTPClientWithCredential(cred),
	}
}

// Type 返回加载器类型
func (l *KuDocLoader) Type() string {
	return types.SourceTypeKuDoc
}

// Load 加载知识库文档数据
func (l *KuDocLoader) Load(ctx context.Context, source types.SourceConfig) ([]types.RawKnowledge, error) {
	allDocs := make([]types.RawKnowledge, 0)

	for _, filter := range source.Filters {
		docs, err := l.loadByFilter(ctx, filter)
		if err != nil {
			return nil, fmt.Errorf("failed to load ku docs for filter: %w", err)
		}
		allDocs = append(allDocs, docs...)
	}

	return allDocs, nil
}

// loadByFilter 根据filter加载知识库文档
func (l *KuDocLoader) loadByFilter(ctx context.Context, filter types.FilterConfig) ([]types.RawKnowledge, error) {
	// 优先使用URLs直接获取；urls 模式下 users 过滤不生效
	if len(filter.Urls) > 0 {
		return l.loadByUrls(ctx, filter.Urls)
	}

	// 必须指定知识库ID
	if filter.RepositoryGuid == "" {
		return nil, types.NewDetailedError(types.ErrorTypeConfig, "repositoryGuid is required").
			WithDetail("filter", "ku_doc").
			WithSuggestion("请在配置文件中指定 repositoryGuid 或 urls 字段")
	}

	// 根据是否指定父文档ID选择不同的查询方式，同时传入 users 过滤列表
	return l.loadByRepository(ctx, filter.RepositoryGuid, filter.ParentDocGuid, filter.Users)
}

// loadByUrls 根据URL列表直接加载文档
func (l *KuDocLoader) loadByUrls(ctx context.Context, urls []string) ([]types.RawKnowledge, error) {
	if len(urls) == 0 {
		return []types.RawKnowledge{}, nil
	}

	utils.Debug("Loading ku docs by URLs",
		zap.Int("count", len(urls)))

	// 使用并发加载
	docs, err := l.loadDocsConcurrently(ctx, urls)
	if err != nil {
		return nil, fmt.Errorf("failed to load docs concurrently: %w", err)
	}

	utils.Debug("Loaded ku docs by URLs",
		zap.Int("requested", len(urls)),
		zap.Int("succeeded", len(docs)))

	return docs, nil
}

// loadByRepository 根据知识库ID加载文档
// users: 按用户名过滤，只拉取创建者或发布者在列表中的文档；为空时不过滤
func (l *KuDocLoader) loadByRepository(ctx context.Context, repositoryGuid string, parentDocGuid string, users []string) ([]types.RawKnowledge, error) {
	utils.Debug("Loading ku docs by repository",
		zap.String("repositoryGuid", repositoryGuid),
		zap.String("parentDocGuid", parentDocGuid))

	parentPath := ""
	allDocs := make([]types.RawKnowledge, 0)

	// 下载父文档内容（不参与 users 过滤，默认一定下载）
	if parentDocGuid != "" {
		docURL := l.resolveDocURL(ctx, repositoryGuid, parentDocGuid)
		if docURL != "" {
			parentDoc, err := l.getDocContentByURL(ctx, docURL, "")
			if err != nil {
				utils.PrintlnWarn("Failed to get parent doc %s: %v", parentDocGuid, err)
			} else {
				if name, ok := parentDoc.Data["name"]; ok {
					parentPath = utils.SanitizeFilename(fmt.Sprintf("%v", name))
				}
				// 设置父文档的存储路径为父目录
				parentDoc.Data["path"] = parentPath
				allDocs = append(allDocs, parentDoc)
			}
		}
	}

	// 获取文档列表(带层级路径，同时按 users 过滤)
	docPaths, err := l.getDocListWithPath(ctx, repositoryGuid, parentDocGuid, parentPath, users)
	if err != nil {
		return nil, fmt.Errorf("failed to get doc list: %w", err)
	}

	if len(docPaths) == 0 && len(allDocs) == 0 {
		utils.PrintlnWarn("No docs found in repository, repositoryGuid: %s", repositoryGuid)
		return []types.RawKnowledge{}, nil
	}

	utils.Debug("Found docs in repository",
		zap.String("repositoryGuid", repositoryGuid),
		zap.Int("count", len(docPaths)))

	// 使用并发加载（携带层级路径）
	docs, err := l.loadDocsConcurrentlyWithPath(ctx, docPaths)
	if err != nil {
		return nil, fmt.Errorf("failed to load docs concurrently: %w", err)
	}

	utils.Debug("Loaded ku docs by repository",
		zap.Int("total", len(docPaths)),
		zap.Int("succeeded", len(docs)))

	return append(allDocs, docs...), nil
}

// getDocList 获取知识库文档列表(支持分页)
func (l *KuDocLoader) getDocList(ctx context.Context, repositoryGuid string, parentDocGuid string) ([]KuDocInfo, error) {
	const pageSize = 100
	allDocs := make([]KuDocInfo, 0)

	// 第一次请求获取总数
	firstPage, total, err := l.getDocListPage(ctx, repositoryGuid, parentDocGuid, pageSize, 1)
	if err != nil {
		return nil, err
	}
	allDocs = append(allDocs, firstPage...)

	// 如果还有更多页,继续请求
	if total > pageSize {
		totalPages := (total + pageSize - 1) / pageSize
		utils.Debug("Need to fetch more pages",
			zap.Int("total", total),
			zap.Int("totalPages", totalPages))

		for pageNum := 2; pageNum <= totalPages; pageNum++ {
			pageDocs, _, err := l.getDocListPage(ctx, repositoryGuid, parentDocGuid, pageSize, pageNum)
			if err != nil {
				utils.PrintlnError("Failed to fetch page, pageNum: %d, err: %v", pageNum, err.Error())
				continue
			}
			allDocs = append(allDocs, pageDocs...)
		}
	}

	// 过滤只保留type=1的普通文档,并递归获取有子文档的文档
	filteredDocs := make([]KuDocInfo, 0, len(allDocs))
	for _, doc := range allDocs {
		if doc.Type == 1 {
			filteredDocs = append(filteredDocs, doc)

			// 如果有子文档,递归获取
			if doc.ChildCount > 0 {
				utils.Debug("Doc has children, fetching recursively",
					zap.String("docGuid", doc.DocGuid),
					zap.String("name", doc.Name),
					zap.Int("childCount", doc.ChildCount))

				childDocs, err := l.getDocList(ctx, repositoryGuid, doc.DocGuid)
				if err != nil {
					utils.PrintlnError("Failed to get child docs, parentDocGuid: %s, err: %v\n", doc.DocGuid, err.Error())
					continue
				}
				filteredDocs = append(filteredDocs, childDocs...)
			}
		}
	}

	utils.Debug("Filtered docs by type",
		zap.Int("total", len(allDocs)),
		zap.Int("filtered", len(filteredDocs)))

	return filteredDocs, nil
}

// getDocListWithPath 获取知识库文档列表，保留层级路径信息
// parentPath 表示当前层级文档应存放的目录路径
// 有子文档的文档和子文档一起放在以该文档名命名的子目录下，叶子文档直接放在当前目录
func (l *KuDocLoader) getDocListWithPath(ctx context.Context, repositoryGuid string, parentDocGuid string, parentPath string, users []string) ([]docWithPath, error) {
	// 只获取当前层级的文档（不递归）
	currentDocs, err := l.getDocListCurrentLevel(ctx, repositoryGuid, parentDocGuid)
	if err != nil {
		return nil, err
	}

	// 按用户名过滤（仅在 users 不为空时生效）
	currentDocs = filterDocsByUsers(currentDocs, users)

	result := make([]docWithPath, 0, len(currentDocs))
	for _, doc := range currentDocs {
		if doc.ChildCount > 0 {
			// 有子文档的文档，放到以自己名命名的子目录下，子文档也在同一目录
			sanitizedName := utils.SanitizeFilename(doc.Name)
			docPath := sanitizedName
			if parentPath != "" {
				docPath = filepath.Join(parentPath, sanitizedName)
			}
			result = append(result, docWithPath{
				URL:     doc.URL,
				DocPath: docPath,
				DocGuid: doc.DocGuid,
				DocName: doc.Name,
			})
			childDocs, err := l.getDocListWithPath(ctx, repositoryGuid, doc.DocGuid, docPath, users)
			if err != nil {
				utils.PrintlnError("Failed to get child docs for path building, parentDocGuid: %s, err: %v", doc.DocGuid, err.Error())
				continue
			}
			result = append(result, childDocs...)
		} else {
			// 叶子文档，直接放在当前目录下
			result = append(result, docWithPath{
				URL:     doc.URL,
				DocPath: parentPath,
				DocGuid: doc.DocGuid,
				DocName: doc.Name,
			})
		}
	}

	return result, nil
}

// resolveDocURL 获取指定文档的URL，通过查询仓库文档列表中的URL来构造
// URL格式: https://ku.baidu-int.com/knowledge/{spaceGuid}/{teamGuid}/{repoGuid}/{docGuid}
func (l *KuDocLoader) resolveDocURL(ctx context.Context, repositoryGuid string, docGuid string) string {
	// 优先从该文档的子文档列表中提取URL前缀
	docs, err := l.getDocListCurrentLevel(ctx, repositoryGuid, docGuid)
	if err == nil && len(docs) > 0 {
		for _, doc := range docs {
			if url := l.buildDocURLFromPrefix(doc.URL, repositoryGuid, docGuid); url != "" {
				return url
			}
		}
	}

	// 兜底：从仓库根级文档列表中提取URL前缀（解决叶子节点没有子文档的问题）
	rootDocs, err := l.getDocListCurrentLevel(ctx, repositoryGuid, "")
	if err == nil && len(rootDocs) > 0 {
		for _, doc := range rootDocs {
			if url := l.buildDocURLFromPrefix(doc.URL, repositoryGuid, docGuid); url != "" {
				return url
			}
		}
	}

	return ""
}

// buildDocURLFromPrefix 从已有的文档URL中提取前缀，拼接目标docGuid
func (l *KuDocLoader) buildDocURLFromPrefix(existingURL string, repositoryGuid string, docGuid string) string {
	if existingURL == "" {
		return ""
	}
	idx := strings.LastIndex(existingURL, repositoryGuid)
	if idx >= 0 {
		prefix := existingURL[:idx+len(repositoryGuid)]
		return prefix + "/" + docGuid
	}
	return ""
}

func (l *KuDocLoader) getDocListCurrentLevel(ctx context.Context, repositoryGuid string, parentDocGuid string) ([]KuDocInfo, error) {
	const pageSize = 100
	allDocs := make([]KuDocInfo, 0)

	// 第一次请求获取总数
	firstPage, total, err := l.getDocListPage(ctx, repositoryGuid, parentDocGuid, pageSize, 1)
	if err != nil {
		return nil, err
	}
	allDocs = append(allDocs, firstPage...)

	// 如果还有更多页,继续请求
	if total > pageSize {
		totalPages := (total + pageSize - 1) / pageSize
		for pageNum := 2; pageNum <= totalPages; pageNum++ {
			pageDocs, _, err := l.getDocListPage(ctx, repositoryGuid, parentDocGuid, pageSize, pageNum)
			if err != nil {
				utils.PrintlnError("Failed to fetch page, pageNum: %d, err: %v", pageNum, err.Error())
				continue
			}
			allDocs = append(allDocs, pageDocs...)
		}
	}

	// 过滤只保留type=1的普通文档
	filteredDocs := make([]KuDocInfo, 0, len(allDocs))
	for _, doc := range allDocs {
		if doc.Type == 1 {
			filteredDocs = append(filteredDocs, doc)
		}
	}

	return filteredDocs, nil
}

// getDocListPage 获取单页文档列表
func (l *KuDocLoader) getDocListPage(ctx context.Context, repositoryGuid string, parentDocGuid string, pageSize int, pageNum int) ([]KuDocInfo, int, error) {
	apiURL := fmt.Sprintf("%s%s", types.GetKuHost(), types.PathKuDocPageByRepo)

	// 构建请求体
	reqBody := map[string]interface{}{
		"repoId":   repositoryGuid,
		"pageSize": pageSize,
		"pageNum":  pageNum,
	}
	if parentDocGuid != "" {
		reqBody["parentDocGuid"] = parentDocGuid
	}

	reqData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, 0, types.NewDetailedError(types.ErrorTypeValidation, "failed to marshal request").
			WithDetail("rawError", err.Error()).
			WithSuggestion("请求参数格式错误")
	}

	utils.Debug("Getting ku doc list page",
		zap.String("repositoryGuid", repositoryGuid),
		zap.String("parentDocGuid", parentDocGuid),
		zap.Int("pageSize", pageSize),
		zap.Int("pageNum", pageNum))

	// 发送请求
	respBody, err := l.httpClient.Post(ctx, apiURL, reqData)
	if err != nil {
		// 如果已经是 DetailedError，直接返回
		if _, ok := err.(*types.DetailedError); ok {
			return nil, 0, err
		}
		return nil, 0, types.NewDetailedError(types.ErrorTypeNetwork, "failed to get doc list").
			WithDetail("url", apiURL).
			WithDetail("repositoryGuid", repositoryGuid).
			WithDetail("rawError", err.Error()).
			WithSuggestion(utils.GetNetworkSuggestion(err))
	}

	// 解析响应
	var resp KuDocPageResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, 0, types.NewDetailedError(types.ErrorTypeValidation, "failed to parse doc list response").
			WithDetail("url", apiURL).
			WithDetail("rawError", err.Error()).
			WithDetail("responseBody", utils.TruncateString(string(respBody), 500)).
			WithSuggestion("API 响应格式错误，请稍后重试或联系管理员")
	}

	if resp.Code != 0 {
		return nil, 0, types.NewDetailedError(types.ErrorTypeAPI, "API returned error").
			WithDetail("url", apiURL).
			WithDetail("repositoryGuid", repositoryGuid).
			WithDetail("statusCode", fmt.Sprintf("%d", resp.Code)).
			WithDetail("message", resp.Msg).
			WithSuggestion(utils.GetAPISuggestion(resp.Code))
	}

	return resp.Data.Data, resp.Data.Count, nil
}

// getDocContentByURL 根据URL获取文档内容
func (l *KuDocLoader) getDocContentByURL(ctx context.Context, docURL string, docPath string) (types.RawKnowledge, error) {
	// 获取文档的JSON内容
	apiURL := fmt.Sprintf("%s%s", types.GetKuHost(), types.PathKuDocQueryContent)

	reqBody := map[string]interface{}{
		"url": docURL,
	}
	reqData, err := json.Marshal(reqBody)
	if err != nil {
		return types.RawKnowledge{}, types.NewDetailedError(types.ErrorTypeValidation, "failed to marshal request").
			WithDetail("docURL", docURL).
			WithDetail("rawError", err.Error())
	}

	utils.Debug("Getting ku doc content",
		zap.String("url", docURL))

	// 发送请求
	respBody, err := l.httpClient.Post(ctx, apiURL, reqData)
	if err != nil {
		// 如果已经是 DetailedError，直接返回
		if _, ok := err.(*types.DetailedError); ok {
			return types.RawKnowledge{}, err
		}
		return types.RawKnowledge{}, types.NewDetailedError(types.ErrorTypeNetwork, "failed to get doc content").
			WithDetail("url", apiURL).
			WithDetail("docURL", docURL).
			WithDetail("rawError", err.Error()).
			WithSuggestion(utils.GetNetworkSuggestion(err))
	}

	// 解析响应
	var resp KuDocContentResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return types.RawKnowledge{}, types.NewDetailedError(types.ErrorTypeValidation, "failed to parse doc content response").
			WithDetail("url", apiURL).
			WithDetail("docURL", docURL).
			WithDetail("rawError", err.Error()).
			WithDetail("responseBody", utils.TruncateString(string(respBody), 500)).
			WithSuggestion("API 响应格式错误，请稍后重试或联系管理员")
	}

	if resp.Code != 200 {
		return types.RawKnowledge{}, types.NewDetailedError(types.ErrorTypeAPI, "API returned error").
			WithDetail("url", apiURL).
			WithDetail("docURL", docURL).
			WithDetail("statusCode", fmt.Sprintf("%d", resp.Code)).
			WithDetail("message", resp.Msg).
			WithSuggestion(utils.GetAPISuggestion(resp.Code))
	}

	// 从URL中提取文档信息
	docGuid, docName := l.extractDocInfoFromURL(docURL)

	// 将JSON内容转换为Markdown
	var markdown string
	if len(resp.Data.Text) > 0 {
		markdown = resp.Data.Text
	} else if resp.Data.Content != nil {
		markdown, err = l.convertJSONToMarkdown(ctx, resp.Data.Content)
		if err != nil {
			return types.RawKnowledge{}, err
		}
	} else {
		return types.RawKnowledge{}, fmt.Errorf("no content found in doc")
	}
	// 从markdown内容中提取标题，提取规则找到第一个\n之前的内容
	docTitle := extractDocTitleFromContent(markdown)
	if docTitle != "" {
		docName = docTitle
	}

	// 封装为RawKnowledge
	data := map[string]interface{}{
		"docGuid": docGuid,
		"name":    docName,
		"content": markdown,
		"url":     docURL,
	}
	if docPath != "" {
		data["path"] = docPath
	}
	return types.RawKnowledge{
		SourceType: types.SourceTypeKuDoc,
		Data:       data,
	}, nil
}

// convertJSONToMarkdown 将JSON内容转换为Markdown
func (l *KuDocLoader) convertJSONToMarkdown(ctx context.Context, content interface{}) (string, error) {
	apiURL := fmt.Sprintf("%s%s", types.KuJsonToMdHost, types.PathKuJsonToMd)

	// 构建请求体
	reqBody := map[string]interface{}{
		"data": content,
	}
	reqData, err := json.Marshal(reqBody)
	if err != nil {
		return "", types.NewDetailedError(types.ErrorTypeValidation, "failed to marshal request").
			WithDetail("rawError", err.Error())
	}

	utils.Debug("Converting JSON to Markdown")

	// 发送请求
	respBody, err := l.httpClient.Post(ctx, apiURL, reqData)
	if err != nil {
		// 如果已经是 DetailedError，直接返回
		if _, ok := err.(*types.DetailedError); ok {
			return "", err
		}
		return "", types.NewDetailedError(types.ErrorTypeNetwork, "failed to convert to markdown").
			WithDetail("url", apiURL).
			WithDetail("rawError", err.Error()).
			WithSuggestion(utils.GetNetworkSuggestion(err))
	}

	// 解析响应
	var resp KuJsonToMdResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", types.NewDetailedError(types.ErrorTypeValidation, "failed to parse conversion response").
			WithDetail("url", apiURL).
			WithDetail("rawError", err.Error()).
			WithDetail("responseBody", utils.TruncateString(string(respBody), 500)).
			WithSuggestion("转换服务响应格式错误，请稍后重试或联系管理员")
	}

	if resp.Code != 200 {
		return "", types.NewDetailedError(types.ErrorTypeAPI, "conversion API returned error").
			WithDetail("url", apiURL).
			WithDetail("statusCode", fmt.Sprintf("%d", resp.Code)).
			WithSuggestion(utils.GetAPISuggestion(resp.Code))
	}

	return resp.Data, nil
}

// loadDocsConcurrently 并发加载文档内容（不带层级路径，用于 loadByUrls）
func (l *KuDocLoader) loadDocsConcurrently(ctx context.Context, urls []string) ([]types.RawKnowledge, error) {
	if len(urls) == 0 {
		return []types.RawKnowledge{}, nil
	}

	// 将 URLs 转换为不带路径的 docWithPath
	docPaths := make([]docWithPath, 0, len(urls))
	for _, url := range urls {
		docPaths = append(docPaths, docWithPath{
			URL:     url,
			DocPath: "", // 无层级信息
			DocGuid: "",
			DocName: "",
		})
	}

	return l.loadDocsConcurrentlyWithPath(ctx, docPaths)
}

// loadDocsConcurrentlyWithPath 并发加载文档内容（携带层级路径）
func (l *KuDocLoader) loadDocsConcurrentlyWithPath(ctx context.Context, docPaths []docWithPath) ([]types.RawKnowledge, error) {
	if len(docPaths) == 0 {
		return []types.RawKnowledge{}, nil
	}

	utils.Debug("Starting concurrent doc fetch",
		zap.Int("total", len(docPaths)),
		zap.Int("concurrency", MaxConcurrentDocFetch))

	// 创建任务队列和结果队列
	tasks := make(chan docFetchTask, len(docPaths))
	results := make(chan docFetchResult, len(docPaths))

	// 启动工作协程池
	var wg sync.WaitGroup
	for i := 0; i < MaxConcurrentDocFetch; i++ {
		wg.Add(1)
		go l.docFetchWorker(ctx, tasks, results, &wg)
	}

	// 发送任务
	for i, dp := range docPaths {
		tasks <- docFetchTask{
			URL:     dp.URL,
			DocPath: dp.DocPath,
			Index:   i,
		}
	}
	close(tasks)

	// 等待所有worker完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集结果
	return l.collectResults(results, len(docPaths))
}

// docFetchWorker 文档获取工作协程
func (l *KuDocLoader) docFetchWorker(ctx context.Context, tasks <-chan docFetchTask, results chan<- docFetchResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for task := range tasks {
		// 检查context是否已取消
		select {
		case <-ctx.Done():
			results <- docFetchResult{
				Index: task.Index,
				Err:   ctx.Err(),
			}
			return
		default:
		}

		// 获取文档内容(包含JSON转MD)，传递层级路径
		doc, err := l.getDocContentByURL(ctx, task.URL, task.DocPath)
		results <- docFetchResult{
			Doc:   doc,
			Index: task.Index,
			Err:   err,
		}
	}
}

// collectResults 收集结果并保持顺序
func (l *KuDocLoader) collectResults(results <-chan docFetchResult, expectedCount int) ([]types.RawKnowledge, error) {
	// 使用map临时存储,保证顺序
	resultMap := make(map[int]types.RawKnowledge)
	errorCount := 0

	for result := range results {
		if result.Err != nil {
			utils.PrintlnError("Failed to fetch doc at index %d: %v\n", result.Index, result.Err.Error())
			errorCount++
			continue
		}
		resultMap[result.Index] = result.Doc
	}

	// 按索引顺序组装结果
	docs := make([]types.RawKnowledge, 0, len(resultMap))
	for i := 0; i < expectedCount; i++ {
		if doc, ok := resultMap[i]; ok {
			docs = append(docs, doc)
		}
	}

	utils.Debug("Concurrent doc fetch completed",
		zap.Int("total", expectedCount),
		zap.Int("succeeded", len(docs)),
		zap.Int("failed", errorCount))

	return docs, nil
}

// extractDocInfoFromURL 从URL中提取文档GUID和名称
// URL格式: https://ku.baidu-int.com/knowledge/{spaceGuid}/{teamGuid}/{repoGuid}/{docGuid}
func (l *KuDocLoader) extractDocInfoFromURL(docURL string) (docGuid string, docName string) {
	// 提取docGuid (URL最后一段)
	parts := strings.Split(strings.TrimSuffix(docURL, "/"), "/")
	if len(parts) > 0 {
		docGuid = parts[len(parts)-1]
	}

	// docName暂时使用docGuid, 实际名称从API返回中获取
	docName = docGuid

	return docGuid, docName
}

// Convert 转换格式
func (l *KuDocLoader) Convert(sourceType string, raw []types.RawKnowledge, output types.OutputConfig) ([]types.KnowledgeItem, error) {
	// ku_doc通常不需要转换,保留原始格式
	if !output.NeedConvert {
		// 直接将RawKnowledge转换为KnowledgeItem
		items := make([]types.KnowledgeItem, 0, len(raw))
		for _, r := range raw {
			metadata := map[string]interface{}{
				"url":     r.Data["url"],
				"docGuid": r.Data["docGuid"],
				"name":    r.Data["name"],
				"content": r.Data["content"],
			}
			// 携带层级路径
			if path, ok := r.Data["path"]; ok {
				metadata["path"] = path
			}
			item := types.KnowledgeItem{
				ID:       fmt.Sprintf("%v", r.Data["docGuid"]),
				Title:    fmt.Sprintf("%v", r.Data["name"]),
				Content:  fmt.Sprintf("%v", r.Data["content"]),
				Metadata: metadata,
			}
			items = append(items, item)
		}
		return items, nil
	}

	// 如果需要转换,返回不支持错误
	return nil, fmt.Errorf("conversion not supported for ku_doc loader")
}

// Store 存储到本地，返回存储的知识项详细信息列表
func (l *KuDocLoader) Store(ctx context.Context, items []types.KnowledgeItem, storage *storage.LocalStorage, source types.SourceConfig) ([]types.StoredKnowledgeInfo, error) {
	// 确保子目录存在
	subDir, err := storage.EnsureSubDir(types.SourceTypeKuDoc)
	if err != nil {
		return nil, types.NewDetailedError(types.ErrorTypeStorage, "failed to create sub directory").
			WithDetail("subDir", types.SourceTypeKuDoc).
			WithDetail("rawError", err.Error()).
			WithSuggestion("请检查存储路径的权限和磁盘空间")
	}

	// 保存每个文档并收集信息
	storedInfos := make([]types.StoredKnowledgeInfo, 0, len(items))
	for _, item := range items {
		if item.Metadata == nil {
			continue
		}
		item.Title = fmt.Sprintf("%v", item.Metadata["name"])
		item.Content = fmt.Sprintf("%v", item.Metadata["content"])
		item.ID = fmt.Sprintf("%v", item.Metadata["docGuid"])
		// 清理标题作为文件名
		filename := utils.SanitizeFilename(item.Title)
		if filename == "" {
			filename = item.ID
		}

		// 文件扩展名固定为.md
		ext := types.FileSuffixMarkdown

		// 构建文件路径（支持层级目录）
		var filePath string
		if docPath, ok := item.Metadata["path"]; ok && fmt.Sprintf("%v", docPath) != "" {
			filePath = filepath.Join(types.SourceTypeKuDoc, fmt.Sprintf("%v", docPath), filename+ext)
		} else {
			filePath = filepath.Join(types.SourceTypeKuDoc, filename+ext)
		}

		// 写入文件
		if err := storage.WriteFile(filePath, []byte(item.Content), false); err != nil {
			// 记录错误但继续处理其他文档
			utils.PrintlnError("Failed to write ku doc file %s: %v\n", filePath, err.Error())
			continue
		}

		// 收集存储信息
		fullPath := storage.GetFullPath(filePath)
		docName := item.Title
		docURL := ""
		if url, ok := item.Metadata["url"]; ok {
			docURL = fmt.Sprintf("%v", url)
		}

		description := fmt.Sprintf("知识库文档: %s", docURL)

		storedInfos = append(storedInfos, types.StoredKnowledgeInfo{
			Name:        docName,
			Description: description,
			FullPath:    fullPath,
			RelPath:     filePath,
		})

		utils.Debug("Saved ku doc", zap.String("file", filePath))
	}

	fmt.Printf("Stored ku docs, directory: %s, count: %d\n", subDir, len(storedInfos))

	return storedInfos, nil
}

// extractDocTitleFromContent
func extractDocTitleFromContent(content string) (docName string) {
	if len(content) < 1 {
		return ""
	}

	// 获取content中第一个\n前的文本
	strs := strings.Split(content, "\n")
	if len(strs) < 1 {
		return ""
	}
	return strs[0]
}

// KuDocPageResponse 知识库文档列表响应
type KuDocPageResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Count int         `json:"count"`
		Data  []KuDocInfo `json:"data"`
		Total int         `json:"total"`
	} `json:"result"`
}

// KuDocInfo 知识库文档信息
type KuDocInfo struct {
	RepositoryGuid    string            `json:"repositoryGuid"`
	DocGuid           string            `json:"docGuid"`
	Name              string            `json:"name"`
	Type              int               `json:"type"` // 1=普通文档, 2=知识本目录, 3=知识本文档, 4=普通表, 6=PPT, 7=数据表, 8=数据表子节点
	ChildCount        int               `json:"childCount"`
	Created           string            `json:"created"`
	PublishTime       string            `json:"publishTime"`
	URL               string            `json:"url"`
	CreatorUserInfo   CreatorUserInfo   `json:"creatorUserInfo"`
	PublisherUserInfo PublisherUserInfo `json:"publisherUserInfo"`
}

type CreatorUserInfo struct {
	UserName string `json:"username"`
}

type PublisherUserInfo struct {
	UserName string `json:"username"`
}

// filterDocsByUsers 按创建者或发布者用户名过滤文档列表。
// 当 users 为空时直接返回原列表（不过滤）。
func filterDocsByUsers(docs []KuDocInfo, users []string) []KuDocInfo {
	if len(users) == 0 {
		return docs
	}

	userSet := make(map[string]struct{}, len(users))
	for _, u := range users {
		userSet[u] = struct{}{}
	}

	filtered := make([]KuDocInfo, 0, len(docs))
	for _, doc := range docs {
		_, isCreator := userSet[doc.CreatorUserInfo.UserName]
		_, isPublisher := userSet[doc.PublisherUserInfo.UserName]
		if isCreator || isPublisher {
			filtered = append(filtered, doc)
		}
	}
	return filtered
}

// KuDocContentResponse 知识库文档内容响应
type KuDocContentResponse struct {
	Code int    `json:"returnCode"`
	Msg  string `json:"msg"`
	Data struct {
		DocGuid string      `json:"docGuid"`
		Content interface{} `json:"content"` // JSON结构化内容
		Text    string      `json:"text"`    // 纯文本内容
	} `json:"result"`
}

// KuJsonToMdResponse JSON转Markdown响应
type KuJsonToMdResponse struct {
	Code int    `json:"code"`
	Data string `json:"data"` // Markdown文本
}
