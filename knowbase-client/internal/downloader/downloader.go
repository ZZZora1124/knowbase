// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package downloader

import (
	"context"
	"fmt"
	"path"
	"time"

	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/client"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/linkparser"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/loader"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/storage"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

type Downloader struct {
	credential   *types.Credential
	storage      *storage.LocalStorage
	metricClient *client.MetricClient
}

// NewDownloader 创建下载器（兼容旧接口）
func NewDownloader(authToken, username string, storage *storage.LocalStorage) *Downloader {
	cred := &types.Credential{
		Mode:     types.AuthModeComate,
		Token:    authToken,
		Username: username,
	}
	return &Downloader{
		credential:   cred,
		storage:      storage,
		metricClient: client.NewMetricClientWithCredential(cred),
	}
}

// NewDownloaderWithCredential 创建支持双模式认证的下载器
func NewDownloaderWithCredential(cred *types.Credential, storage *storage.LocalStorage) *Downloader {
	return &Downloader{
		credential:   cred,
		storage:      storage,
		metricClient: client.NewMetricClientWithCredential(cred),
	}
}

func (d *Downloader) Download(ctx context.Context, linkType linkparser.LinkType, params map[string]string) error {
	startTime := time.Now()
	sourceType := d.linkTypeToSourceType(linkType)

	var count int
	var err error

	// 执行下载
	switch linkType {
	case linkparser.LinkTypeICodeRepo:
		count, err = d.downloadICodeRepo(ctx, params)
	case linkparser.LinkTypeICodeCR:
		count, err = d.downloadICodeCRDiff(ctx, params)
	case linkparser.LinkTypeIApi:
		count, err = d.downloadIApiDoc(ctx, params)
	case linkparser.LinkTypeICafe:
		count, err = d.downloadICafeCard(ctx, params)
	case linkparser.LinkTypeKuDoc:
		count, err = d.downloadKuDoc(ctx, params)
	case linkparser.LinkTypeIDataset:
		count, err = d.downloadIDataset(ctx, params)
	default:
		err = fmt.Errorf("unsupported link type: %s", linkType)
	}

	// 构造上报数据
	result := types.LoadResult{
		SourceType: sourceType,
		Count:      count,
		Duration:   time.Since(startTime),
		Success:    err == nil,
	}
	if err != nil {
		result.ErrorMessage = truncateError(err.Error(), 500)
	}

	// 同步上报指标（忽略所有错误）
	_ = d.metricClient.ReportDownloadMetric(context.Background(), result)

	return err
}

func (d *Downloader) downloadICodeCRDiff(ctx context.Context, params map[string]string) (int, error) {
	crURL := params["url"]
	changeNumber := params["changeNumber"]

	fmt.Printf("Downloading iCode CR diff...\n")
	fmt.Printf("  - Change Number: %s\n", changeNumber)
	fmt.Printf("  - URL: %s\n", crURL)

	crLoader := loader.NewICodeCRLoader(d.credential)

	source := types.SourceConfig{
		Type:    types.SourceTypeICodeCRDiff,
		Enabled: true,
		Filters: []types.FilterConfig{
			{
				Urls: []string{crURL},
			},
		},
		Output: types.OutputConfig{
			Format:      types.OutputFormatMarkdown,
			ExportType:  types.ExportTypeMd,
			NeedConvert: false,
		},
	}

	count, err := d.loadConvertStore(ctx, crLoader, source)
	if err != nil {
		return 0, err
	}

	fmt.Printf("✓ Successfully downloaded iCode CR diff (%d files) to: %s/icode_cr_diff/\n",
		count, d.storage.GetBasePath())

	return count, nil
}

func (d *Downloader) downloadICodeRepo(ctx context.Context, params map[string]string) (int, error) {
	repo := params["repo"]
	branch := params["branch"]

	fmt.Printf("Downloading iCode repository...\n")
	fmt.Printf("  - Repository: %s\n", repo)
	fmt.Printf("  - Branch: %s\n", branch)

	repoLoader := loader.NewICodeRepoLoader(d.credential)

	source := types.SourceConfig{
		Type:    types.SourceTypeICodeRepo,
		Enabled: true,
		Filters: []types.FilterConfig{
			{
				Repo:   repo,
				Branch: branch,
				Depth:  "10",
			},
		},
		Output: types.OutputConfig{
			Format:      types.OutputFormatMarkdown,
			NeedConvert: false,
		},
	}

	storeInfo, err := repoLoader.Store(ctx, nil, d.storage, source)
	if err != nil {
		return 0, fmt.Errorf("failed to store iCode repository: %v", err)
	}

	count := len(storeInfo)
	fmt.Printf("✓ Successfully downloaded %d iCode repository to: %s/icode_repo/\n",
		count, d.storage.GetBasePath())
	return count, nil
}

func (d *Downloader) downloadIApiDoc(ctx context.Context, params map[string]string) (int, error) {
	projectID := params["projectId"]
	apiID := params["apiId"]

	fmt.Printf("Downloading iAPI document...\n")
	fmt.Printf("  - Project ID: %s\n", projectID)
	if apiID != "" {
		fmt.Printf("  - API ID: %s\n", apiID)
	} else {
		fmt.Printf("  - Downloading all APIs in the project\n")
	}

	apiLoader := loader.NewIApiDocLoader(d.credential)

	filter := types.FilterConfig{
		ProjectID: projectID,
	}
	if apiID != "" {
		filter.ApiIDs = []string{apiID}
	}

	source := types.SourceConfig{
		Type:    types.SourceTypeIApiDoc,
		Enabled: true,
		Filters: []types.FilterConfig{filter},
		Output: types.OutputConfig{
			Format:      types.OutputFormatMarkdown,
			ExportType:  types.ExportTypeMd,
			NeedConvert: false,
		},
	}

	count, err := d.loadConvertStore(ctx, apiLoader, source)
	if err != nil {
		return 0, err
	}

	fmt.Printf("✓ Successfully downloaded iAPI document (%d files) to: %s/iapi_doc/\n",
		count, d.storage.GetBasePath())

	return count, nil
}

func (d *Downloader) downloadICafeCard(ctx context.Context, params map[string]string) (int, error) {
	linkType := params["type"]

	fmt.Printf("Downloading iCafe card...\n")

	cafeLoader := loader.NewICafeLoader(d.credential)

	var rawData []types.RawKnowledge
	var err error

	if linkType == "single" {
		// 单个卡片，使用原有的Load方法
		issueID := params["issueId"]
		fmt.Printf("  - Issue ID: %s\n", issueID)

		source := types.SourceConfig{
			Type:    types.SourceTypeICafeCard,
			Enabled: true,
			Filters: []types.FilterConfig{
				{
					IssueIDs: []string{issueID},
				},
			},
		}

		rawData, err = cafeLoader.Load(ctx, source)
		if err != nil {
			return 0, fmt.Errorf("failed to load iCafe card: %w", err)
		}

	} else if linkType == "planbox" {
		// planbox，直接获取完整卡片数据
		space := params["space"]
		planID := params["planId"]
		fmt.Printf("  - Space: %s\n", space)
		fmt.Printf("  - Planbox ID: %s\n", planID)
		fmt.Printf("  - Fetching cards from planbox...\n")

		rawData, err = cafeLoader.FetchPlanboxCards(ctx, space, planID)
		if err != nil {
			return 0, fmt.Errorf("failed to fetch planbox cards: %w", err)
		}

		fmt.Printf("  - Found %d card(s) in planbox\n", len(rawData))

	} else {
		return 0, fmt.Errorf("unknown iCafe link type: %s", linkType)
	}

	// 转换和存储
	outputConfig := types.OutputConfig{
		Format: types.OutputFormatMarkdown,
		Fields: []types.FieldConfig{
			{
				KeyPath:     "title",
				Description: "标题",
			},
			{
				KeyPath:     "detail",
				Description: "内容",
			},
		},
	}

	items, err := cafeLoader.Convert(types.SourceTypeICafeCard, rawData, outputConfig)
	if err != nil {
		return 0, fmt.Errorf("failed to convert iCafe cards: %w", err)
	}

	source := types.SourceConfig{
		Type: types.SourceTypeICafeCard,
	}

	storedCount, err := cafeLoader.Store(ctx, items, d.storage, source)
	if err != nil {
		return 0, fmt.Errorf("failed to store iCafe cards: %w", err)
	}

	count := len(storedCount)
	fmt.Printf("✓ Successfully downloaded iCafe card(s) (%d files) to: %s/icafe_card/\n",
		count, d.storage.GetBasePath())

	return count, nil
}

func (d *Downloader) downloadKuDoc(ctx context.Context, params map[string]string) (int, error) {
	repositoryGuid := params["repositoryGuid"]
	docGuid := params["docGuid"]
	fullDocUrl := params["fullDocUrl"]

	fmt.Printf("Downloading KU document...\n")
	fmt.Printf("  - Repository: %s\n", repositoryGuid)
	fmt.Printf("  - Document: %s\n", docGuid)

	kuLoader := loader.NewKuDocLoader(d.credential)

	fmt.Printf("  - Downloading current document...\n")
	currentDocRaw, err := d.downloadCurrentKuDoc(ctx, kuLoader, fullDocUrl)
	if err != nil {
		return 0, fmt.Errorf("failed to download current document: %w", err)
	}
	fmt.Printf("  ✓ Current document downloaded\n")

	fmt.Printf("  - Checking for sub-documents...\n")
	childDocsRaw, err := d.downloadChildKuDocs(ctx, kuLoader, repositoryGuid, docGuid)
	if err != nil {
		fmt.Printf("  ⚠ Warning: failed to download child documents: %v\n", err)
		childDocsRaw = []types.RawKnowledge{}
	}

	allDocs := append(currentDocRaw, childDocsRaw...)
	if len(allDocs) == 0 {
		return 0, fmt.Errorf("no documents downloaded")
	}

	fmt.Printf("  - Total documents: %d (1 current + %d children)\n",
		len(allDocs), len(childDocsRaw))

	source := types.SourceConfig{
		Type:    types.SourceTypeKuDoc,
		Enabled: true,
		Output: types.OutputConfig{
			Format: types.OutputFormatMarkdown,
		},
	}

	items, err := kuLoader.Convert(source.Type, allDocs, source.Output)
	if err != nil {
		return 0, fmt.Errorf("failed to convert documents: %w", err)
	}

	storedItems, err := kuLoader.Store(ctx, items, d.storage, source)
	if err != nil {
		return 0, fmt.Errorf("failed to store documents: %w", err)
	}

	count := len(storedItems)
	fmt.Printf("✓ Successfully downloaded %d KU document(s) to: %s/ku_doc/\n",
		count, d.storage.GetBasePath())

	return count, nil
}

func (d *Downloader) downloadCurrentKuDoc(ctx context.Context,
	kuLoader *loader.KuDocLoader, fullDocUrl string) ([]types.RawKnowledge, error) {

	source := types.SourceConfig{
		Type:    types.SourceTypeKuDoc,
		Enabled: true,
		Filters: []types.FilterConfig{
			{
				Urls: []string{fullDocUrl},
			},
		},
	}

	docs, err := kuLoader.Load(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("failed to load current document: %w", err)
	}

	if len(docs) == 0 {
		return nil, fmt.Errorf("current document not found")
	}

	return docs, nil
}

func (d *Downloader) downloadChildKuDocs(ctx context.Context,
	kuLoader *loader.KuDocLoader, repositoryGuid, parentDocGuid string) ([]types.RawKnowledge, error) {

	source := types.SourceConfig{
		Type:    types.SourceTypeKuDoc,
		Enabled: true,
		Filters: []types.FilterConfig{
			{
				RepositoryGuid: repositoryGuid,
				ParentDocGuid:  parentDocGuid,
			},
		},
	}

	docs, err := kuLoader.Load(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("failed to load child documents: %w", err)
	}

	if len(docs) == 0 {
		fmt.Printf("  - No child documents found\n")
	} else {
		fmt.Printf("  ✓ Found %d child document(s) (recursively)\n", len(docs))
	}

	return docs, nil
}

func (d *Downloader) loadConvertStore(ctx context.Context, loader loader.KnowledgeLoader, source types.SourceConfig) (int, error) {
	rawData, err := loader.Load(ctx, source)
	if err != nil {
		return 0, fmt.Errorf("failed to load: %w", err)
	}

	items, err := loader.Convert(source.Type, rawData, source.Output)
	if err != nil {
		return 0, fmt.Errorf("failed to convert: %w", err)
	}

	count, err := loader.Store(ctx, items, d.storage, source)
	if err != nil {
		return 0, fmt.Errorf("failed to store: %w", err)
	}

	return len(count), nil
}

// linkTypeToSourceType 将linkType转换为sourceType
func (d *Downloader) linkTypeToSourceType(linkType linkparser.LinkType) string {
	switch linkType {
	case linkparser.LinkTypeICodeRepo:
		return types.SourceTypeICodeRepo
	case linkparser.LinkTypeICodeCR:
		return types.SourceTypeICodeCRDiff
	case linkparser.LinkTypeIApi:
		return types.SourceTypeIApiDoc
	case linkparser.LinkTypeICafe:
		return types.SourceTypeICafeCard
	case linkparser.LinkTypeKuDoc:
		return types.SourceTypeKuDoc
	case linkparser.LinkTypeIDataset:
		return types.SourceTypeIDataset
	default:
		return "unknown"
	}
}

func (d *Downloader) downloadIDataset(ctx context.Context, params map[string]string) (int, error) {
	repo := params["repo"]
	branch := params["branch"]
	pathParam := params["path"]
	fileType := params["type"] // "tree" or "blob"

	fmt.Printf("Downloading iDataset...\n")
	fmt.Printf("  - Repository: %s\n", repo)
	fmt.Printf("  - Branch: %s\n", branch)
	if pathParam != "" {
		fmt.Printf("  - Path: %s\n", pathParam)
	}
	if fileType == "blob" {
		fmt.Printf("  - Mode: single file\n")
	}

	datasetLoader := loader.NewIDatasetLoader(d.credential)

	filter := types.FilterConfig{
		Repo:   repo,
		Branch: branch,
	}

	// 设置目录过滤
	if pathParam != "" {
		if fileType == "blob" {
			// blob模式：使用父目录作为 DirectoryNames
			parentDir := path.Dir(pathParam)
			if parentDir != "." && parentDir != "" {
				filter.DirectoryNames = []string{parentDir}
			}
		} else {
			// tree模式：直接使用路径作为 DirectoryNames
			filter.DirectoryNames = []string{pathParam}
		}
	}

	source := types.SourceConfig{
		Type:    types.SourceTypeIDataset,
		Enabled: true,
		Filters: []types.FilterConfig{filter},
		Output: types.OutputConfig{
			Format:      types.OutputFormatMarkdown,
			NeedConvert: false,
		},
	}

	// 加载文件列表
	rawData, err := datasetLoader.Load(ctx, source)
	if err != nil {
		return 0, fmt.Errorf("failed to load iDataset files: %w", err)
	}

	// blob模式下过滤只保留指定文件
	if fileType == "blob" && pathParam != "" {
		filtered := make([]types.RawKnowledge, 0)
		for _, raw := range rawData {
			if rawPath, ok := raw.Data["path"].(string); ok && rawPath == pathParam {
				filtered = append(filtered, raw)
			}
		}
		if len(filtered) == 0 {
			return 0, fmt.Errorf("file not found: %s", pathParam)
		}
		rawData = filtered
	}

	// 转换为 KnowledgeItem（Store方法需要）
	items := make([]types.KnowledgeItem, 0, len(rawData))
	for _, raw := range rawData {
		items = append(items, types.KnowledgeItem{
			Metadata: raw.Data,
		})
	}

	// 存储文件
	storedInfos, err := datasetLoader.Store(ctx, items, d.storage, source)
	if err != nil {
		return 0, fmt.Errorf("failed to store iDataset files: %w", err)
	}

	count := len(storedInfos)
	fmt.Printf("✓ Successfully downloaded iDataset (%d files) to: %s/%s/\n",
		count, d.storage.GetBasePath(), types.SourceTypeIDataset)

	return count, nil
}

// truncateError 截断过长的错误信息
func truncateError(msg string, maxLen int) string {
	if len(msg) <= maxLen {
		return msg
	}
	return msg[:maxLen]
}
