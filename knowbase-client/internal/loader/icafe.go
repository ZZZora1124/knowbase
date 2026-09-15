// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package loader

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/client"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/converter"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/storage"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/utils"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

// getICafeAPIBaseURL 获取iCafe API基础URL
func getICafeAPIBaseURL() string {
	return fmt.Sprintf("%s%s", types.GetAuthHost(), types.PathIcafeCard)
}

// ICafeLoader iCafe卡片加载器
type ICafeLoader struct {
	*BaseLoader
	httpClient *client.HTTPClient
	converter  *converter.Converter
}

// NewICafeLoader 创建iCafe加载器
func NewICafeLoader(cred *types.Credential) *ICafeLoader {
	return &ICafeLoader{
		BaseLoader: NewBaseLoader(cred),
		httpClient: client.NewHTTPClientWithCredential(cred),
		converter:  converter.NewConverter(),
	}
}

// Type 返回加载器类型
func (l *ICafeLoader) Type() string {
	return types.SourceTypeICafeCard
}

// Load 加载iCafe卡片数据
func (l *ICafeLoader) Load(ctx context.Context, source types.SourceConfig) ([]types.RawKnowledge, error) {
	allCards := make([]types.RawKnowledge, 0)

	for _, filter := range source.Filters {
		cards, err := l.loadByFilter(ctx, filter)
		if err != nil {
			return nil, fmt.Errorf("failed to load cards for filter: %w", err)
		}
		allCards = append(allCards, cards...)
	}

	return allCards, nil
}

// loadByFilter 根据filter加载卡片
func (l *ICafeLoader) loadByFilter(ctx context.Context, filter types.FilterConfig) ([]types.RawKnowledge, error) {
	// 如果指定了issueIds，直接获取
	if len(filter.IssueIDs) > 0 {
		return l.loadByIssueIDs(ctx, filter.Space, filter.IssueIDs)
	}

	// 否则按条件查询
	return l.loadByQuery(ctx, filter)
}

// loadByIssueIDs 根据卡片ID列表加载
func (l *ICafeLoader) loadByIssueIDs(ctx context.Context, space string, issueIDs []string) ([]types.RawKnowledge, error) {
	cards := make([]types.RawKnowledge, 0, len(issueIDs))

	for _, issueID := range issueIDs {
		space, sequence := getSpaceAndSequence(issueID)
		// 构建URL
		apiURL := fmt.Sprintf("%s/spaces/%s/cards/%s", getICafeAPIBaseURL(), space, sequence)

		// 发送请求
		respBody, err := l.httpClient.Get(ctx, apiURL)
		if err != nil {
			// 如果已经是 DetailedError，直接返回
			if _, ok := err.(*types.DetailedError); ok {
				return nil, err
			}
			// 否则包装为 DetailedError
			return nil, types.NewDetailedError(types.ErrorTypeNetwork, fmt.Sprintf("failed to get card %s", issueID)).
				WithDetail("url", apiURL).
				WithDetail("issueID", issueID).
				WithDetail("rawError", err.Error()).
				WithSuggestion(utils.GetNetworkSuggestion(err))
		}

		// 解析响应
		var resp ICafeResponse
		if err := json.Unmarshal(respBody, &resp); err != nil {
			return nil, types.NewDetailedError(types.ErrorTypeValidation, fmt.Sprintf("failed to parse response for card %s", issueID)).
				WithDetail("url", apiURL).
				WithDetail("issueID", issueID).
				WithDetail("rawError", err.Error()).
				WithDetail("responseBody", utils.TruncateString(string(respBody), 500)).
				WithSuggestion("API 响应格式错误，请稍后重试或联系管理员")
		}

		if resp.Code != 200 {
			return nil, types.NewDetailedError(types.ErrorTypeAPI, fmt.Sprintf("API returned error for card %s", issueID)).
				WithDetail("url", apiURL).
				WithDetail("issueID", issueID).
				WithDetail("statusCode", fmt.Sprintf("%d", resp.Code)).
				WithDetail("message", resp.Message).
				WithSuggestion(utils.GetAPISuggestion(resp.Code))
		}

		// 转换为RawKnowledge
		for _, card := range resp.Cards {
			cardRes := processCardDetail(card)
			if cardRes == nil {
				continue
			}
			cards = append(cards, types.RawKnowledge{
				SourceType: types.SourceTypeICafeCard,
				Data:       cardRes,
			})
		}
	}

	return cards, nil
}

// loadByQuery 根据查询条件加载
func (l *ICafeLoader) loadByQuery(ctx context.Context, filter types.FilterConfig) ([]types.RawKnowledge, error) {
	// 校验时间范围
	if err := validateTimeRange(filter.StartTimeDate, filter.EndTimeDate); err != nil {
		return nil, types.NewDetailedError(types.ErrorTypeValidation, "time range validation failed").
			WithDetail("startTime", filter.StartTimeDate).
			WithDetail("endTime", filter.EndTimeDate).
			WithDetail("rawError", err.Error()).
			WithSuggestion("请检查配置文件中的 startTime 和 endTime 格式是否为 YYYY-MM-DD，且 startTime 必须早于 endTime")
	}

	// 构建IQL查询语句
	iql := l.buildIQL(filter)
	apiURL := fmt.Sprintf("%s/spaces/%s/cards", getICafeAPIBaseURL(), filter.Space)

	fmt.Printf("Loading iCafe cards with IQL, space: %v, iql: %v\n", filter.Space, iql)

	// 分页查询所有卡片
	allCards := make([]types.RawKnowledge, 0)
	currentPage := 1
	var totalPages int

	for {
		// 构建URL参数
		params := url.Values{}
		params.Add("iql", iql)
		params.Add("showDetail", "true")
		params.Add("isDesc", "true")
		params.Add("page", fmt.Sprintf("%d", currentPage))
		fullURL := fmt.Sprintf("%s?%s", apiURL, params.Encode())

		// 发送请求
		respBody, err := l.httpClient.Get(ctx, fullURL)
		if err != nil {
			// 如果已经是 DetailedError，直接返回
			if _, ok := err.(*types.DetailedError); ok {
				return nil, err
			}
			// 否则包装为 DetailedError
			return nil, types.NewDetailedError(types.ErrorTypeNetwork, "failed to query cards").
				WithDetail("url", fullURL).
				WithDetail("space", filter.Space).
				WithDetail("iql", iql).
				WithDetail("page", fmt.Sprintf("%d", currentPage)).
				WithDetail("rawError", err.Error()).
				WithSuggestion(utils.GetNetworkSuggestion(err))
		}

		// 解析响应
		var resp ICafeResponse
		if err = json.Unmarshal(respBody, &resp); err != nil {
			return nil, types.NewDetailedError(types.ErrorTypeValidation, "failed to parse response").
				WithDetail("url", fullURL).
				WithDetail("page", fmt.Sprintf("%d", currentPage)).
				WithDetail("rawError", err.Error()).
				WithDetail("responseBody", utils.TruncateString(string(respBody), 500)).
				WithSuggestion("API 响应格式错误，请稍后重试或联系管理员")
		}

		if resp.Code != 200 {
			return nil, types.NewDetailedError(types.ErrorTypeAPI, "API returned error").
				WithDetail("url", fullURL).
				WithDetail("space", filter.Space).
				WithDetail("iql", iql).
				WithDetail("page", fmt.Sprintf("%d", currentPage)).
				WithDetail("statusCode", fmt.Sprintf("%d", resp.Code)).
				WithDetail("message", resp.Message).
				WithSuggestion(utils.GetAPISuggestion(resp.Code))
		}

		if currentPage == 1 {
			totalPages = resp.PageSize
			fmt.Printf("Load iCafe Cards, total: %d cards, %d page(s)\n", resp.Total, totalPages)
		}

		// 转换当前页的卡片为RawKnowledge
		for _, card := range resp.Cards {
			cardRes := processCardDetail(card)
			if cardRes == nil {
				continue
			}
			allCards = append(allCards, types.RawKnowledge{
				SourceType: types.SourceTypeICafeCard,
				Data:       cardRes,
			})
		}

		// 显示进度
		if totalPages > 1 {
			fmt.Printf("Load iCafe Cards, fetching page %d/%d (%d cards)\n", currentPage, totalPages, len(resp.Cards))
		}

		// 如果已经获取所有页或当前页没有数据，退出循环
		if currentPage >= totalPages || len(resp.Cards) == 0 {
			break
		}

		currentPage++
	}

	fmt.Printf("Loaded iCafe cards, count: %d\n", len(allCards))

	return allCards, nil
}

// validateDateFormat 校验日期格式是否为 YYYY-MM-DD
func validateDateFormat(date string) error {
	if date == "" {
		return nil // 空字符串认为是合法的(未设置)
	}

	// 使用time.Parse解析日期
	layout := "2006-01-02"
	_, err := time.Parse(layout, date)
	if err != nil {
		return fmt.Errorf("invalid date format: %s, expected YYYY-MM-DD", date)
	}

	return nil
}

// validateTimeRange 校验时间范围
func validateTimeRange(startTime, endTime string) error {
	// 校验格式
	if err := validateDateFormat(startTime); err != nil {
		return fmt.Errorf("startTime validation failed: %w", err)
	}
	if err := validateDateFormat(endTime); err != nil {
		return fmt.Errorf("endTime validation failed: %w", err)
	}

	// 如果两个都设置了,校验逻辑关系
	if startTime != "" && endTime != "" {
		layout := "2006-01-02"
		start, _ := time.Parse(layout, startTime)
		end, _ := time.Parse(layout, endTime)

		if start.After(end) || start.Equal(end) {
			return fmt.Errorf("startTime (%s) must be before endTime (%s)", startTime, endTime)
		}
	}

	return nil
}

// buildIQL 构建IQL查询语句
func (l *ICafeLoader) buildIQL(filter types.FilterConfig) string {
	conditions := make([]string, 0)

	// 类型条件
	if len(filter.Types) > 0 {
		types := strings.Join(filter.Types, ",")
		conditions = append(conditions, fmt.Sprintf("类型 in (%s)", types))
	}

	// 负责人条件
	if len(filter.Owner) > 0 {
		owners := strings.Join(filter.Owner, ",")
		conditions = append(conditions, fmt.Sprintf("负责人 in (%s)", owners))
	}

	// 状态条件
	if len(filter.Status) > 0 {
		status := strings.Join(filter.Status, ",")
		conditions = append(conditions, fmt.Sprintf("流程状态 in (%s)", status))
	}

	// 关键字条件
	if filter.Keyword != "" {
		conditions = append(conditions, fmt.Sprintf("关键字 ~ %s", filter.Keyword))
	}

	// 时间范围条件 - 优先使用startTime和endTime
	if filter.StartTimeDate != "" || filter.EndTimeDate != "" {
		// 有startTime或endTime时,忽略timeRange
		if filter.StartTimeDate != "" {
			conditions = append(conditions, fmt.Sprintf("创建时间 > %s", filter.StartTimeDate))
		}
		if filter.EndTimeDate != "" {
			conditions = append(conditions, fmt.Sprintf("创建时间 < %s", filter.EndTimeDate))
		}
	} else if filter.TimeRange != "" {
		// 只有在没有设置startTime和endTime时才使用timeRange
		conditions = append(conditions, fmt.Sprintf("创建时间 = %s", filter.TimeRange))
	}

	return strings.Join(conditions, " AND ")
}

// Convert 转换格式
func (l *ICafeLoader) Convert(sourceType string, raw []types.RawKnowledge, output types.OutputConfig) ([]types.KnowledgeItem, error) {
	return l.converter.Convert(sourceType, raw, output)
}

// Store 存储到本地，返回存储的知识项详细信息列表
func (l *ICafeLoader) Store(ctx context.Context, items []types.KnowledgeItem, storage *storage.LocalStorage, source types.SourceConfig) ([]types.StoredKnowledgeInfo, error) {
	// 确保子目录存在
	subDir, err := storage.EnsureSubDir(types.SourceTypeICafeCard)
	if err != nil {
		return nil, types.NewDetailedError(types.ErrorTypeStorage, "failed to create sub directory").
			WithDetail("subDir", types.SourceTypeICafeCard).
			WithDetail("rawError", err.Error()).
			WithSuggestion("请检查存储路径的权限和磁盘空间")
	}

	// 保存每个卡片并收集信息
	storedInfos := make([]types.StoredKnowledgeInfo, 0, len(items))
	for _, item := range items {
		// 清理标题作为文件名
		filename := utils.SanitizeFilename(item.Title)
		if filename == "" {
			filename = item.ID
		}

		// 添加扩展名
		ext := types.FileSuffixMarkdown
		if source.Output.Format == types.OutputFormatJSON {
			ext = types.FileSuffixJSON
		} else if source.Output.Format == types.OutputFormatXML {
			ext = types.FileSuffixXML
		}

		// 构建文件路径
		filePath := filepath.Join(types.SourceTypeICafeCard, filename+ext)

		// 写入文件
		if err := storage.WriteFile(filePath, []byte(item.Content), false); err != nil {
			// 记录错误但继续处理其他项
			utils.PrintlnError("Failed to write card file %s: %v", filePath, err.Error())
			continue
		}
		// 安全获取 Metadata 字段
		var createTime, updateTime, status string
		if item.Metadata != nil {
			if v, ok := item.Metadata["createdTime"].(string); ok {
				createTime = v
			}
			if v, ok := item.Metadata["lastModifiedTime"].(string); ok {
				updateTime = v
			}
			if v, ok := item.Metadata["status"].(string); ok {
				status = v
			}
		}
		item.CreateTime = createTime
		item.UpdateTime = updateTime

		// 收集存储信息
		fullPath := storage.GetFullPath(filePath)
		// description := extractDescription(item.Content, 200) // 提取前200字符作为描述
		storedInfos = append(storedInfos, types.StoredKnowledgeInfo{
			Description: fmt.Sprintf("创建时间: %v, 更新时间: %v, 当前状态: %v", createTime, updateTime, status),
			Name:        item.Title,
			FullPath:    fullPath,
			RelPath:     filePath,
		})

		utils.Debug("Saved iCafe card", zap.String("file", filePath))
	}

	fmt.Printf("Stored iCafe cards, directory: %s, count: %d\n", subDir, len(storedInfos))

	return storedInfos, nil
}

// extractDescription 从内容中提取描述信息
func extractDescription(content string, maxLen int) string {
	// 移除多余的空白字符
	content = strings.TrimSpace(content)

	// 如果内容为空，返回空字符串
	if content == "" {
		return ""
	}

	// 截取指定长度
	if len(content) > maxLen {
		// 尝试在单词边界截断
		content = content[:maxLen]
		if lastSpace := strings.LastIndex(content, " "); lastSpace > maxLen/2 {
			content = content[:lastSpace]
		}
		content += "..."
	}

	return content
}

func getSpaceAndSequence(issueID string) (string, string) {
	// 查找最后一个'-'的位置
	lastIndex := strings.LastIndex(issueID, "-")

	// 如果没有找到'-'
	if lastIndex == -1 {
		return "", ""
	}

	// 分割字符串
	before := issueID[:lastIndex]
	after := issueID[lastIndex+1:]

	return before, after
}

func processCardDetail(card map[string]interface{}) map[string]interface{} {
	if card == nil {
		return nil
	}
	detailOri, ok := card["detail"]
	if ok {
		detail, ok := detailOri.(string)
		if ok {
			card["detail"] = utils.RemoveHTMLTags(detail)
		}
	}
	return card
}

// ICafeResponse iCafe API响应
type ICafeResponse struct {
	Code        int                      `json:"code"`
	Message     string                   `json:"message"`
	Result      string                   `json:"result"`
	Cards       []map[string]interface{} `json:"cards"`
	CurrentPage int                      `json:"currentPage"`
	PageSize    int                      `json:"pageSize"`
	Total       int                      `json:"total"`
}

// FetchPlanboxCards 从planbox获取完整的卡片数据（支持分页）
func (l *ICafeLoader) FetchPlanboxCards(ctx context.Context, space, planID string) ([]types.RawKnowledge, error) {
	allRawCards := make([]types.RawKnowledge, 0)
	currentPage := 1
	var totalPages int

	for {
		// 构建请求URL
		apiURL := fmt.Sprintf("%s%s/spaces/%s/cards",
			types.GetAuthHost(), types.PathIcafeCard, space)

		params := url.Values{}
		params.Add("q", fmt.Sprintf("[%d][in][%s]", 45788, planID))
		params.Add("showDetail", "true")
		params.Add("isDesc", "true")
		params.Add("page", fmt.Sprintf("%d", currentPage))
		fullURL := fmt.Sprintf("%s?%s", apiURL, params.Encode())

		// 发送请求
		respBody, err := l.httpClient.Get(ctx, fullURL)
		if err != nil {
			return nil, fmt.Errorf("failed to send request (page %d): %w", currentPage, err)
		}

		// 解析响应
		var apiResp ICafeResponse
		if err := json.Unmarshal(respBody, &apiResp); err != nil {
			return nil, fmt.Errorf("failed to decode response (page %d): %w", currentPage, err)
		}

		if apiResp.Code != 200 {
			return nil, fmt.Errorf("API returned error code %d: %s", apiResp.Code, apiResp.Message)
		}

		// 计算总页数（第一次循环时）
		if currentPage == 1 {
			totalPages = apiResp.PageSize
			fmt.Printf("Load iCafe Cards, total: %d cards, %d page(s)\n", apiResp.Total, totalPages)
		}

		// 处理当前页的卡片
		for _, card := range apiResp.Cards {
			cleanedCard := processCardDetail(card)
			allRawCards = append(allRawCards, types.RawKnowledge{
				Data: cleanedCard,
			})
		}

		// 显示进度
		if totalPages > 1 {
			fmt.Printf("Load iCafe Cards, fetching page %d/%d (%d cards)\n", currentPage, totalPages, len(apiResp.Cards))
		}

		// 如果已经获取所有页或当前页没有数据，退出循环
		if currentPage >= totalPages || len(apiResp.Cards) == 0 {
			break
		}

		currentPage++
	}

	if len(allRawCards) == 0 {
		return nil, fmt.Errorf("no cards found in planbox %s", planID)
	}

	return allRawCards, nil
}
