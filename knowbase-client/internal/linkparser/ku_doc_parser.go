// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package linkparser

import (
	"fmt"
	"regexp"
	"strings"
)

// KuDocParser KU文档链接解析器
type KuDocParser struct{}

// Parse 解析KU文档链接
// 格式: https://ku.baidu-int.com/knowledge/xxx/xxx/<repositoryGuid>/<parentDocGuid>
// 示例: https://ku.baidu-int.com/knowledge/HFVrC7hq1Q/_SKPgSwp2G/B8wSneaLSC/OUuqDTFJhs8dCw
func (p *KuDocParser) Parse(link string) (map[string]string, error) {
	// 正则匹配模式
	// 路径前缀可能有多段，repositoryGuid和docGuid在最后两段
	pattern := `https://ku\.baidu-int\.com/knowledge/([^/]+/[^/]+)/([^/]+)/([^/?]+)`
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(link)

	if len(matches) != 4 {
		return nil, fmt.Errorf("invalid KU document link format, expected: https://ku.baidu-int.com/knowledge/xxx/xxx/<repositoryGuid>/<docGuid>")
	}

	pathPrefix := strings.TrimSpace(matches[1])       // HFVrC7hq1Q/_SKPgSwp2G
	repositoryGuid := strings.TrimSpace(matches[2])   // B8wSneaLSC
	docGuid := strings.TrimSpace(matches[3])          // OUuqDTFJhs8dCw

	// 构造完整的文档URL（用于loadByUrls下载当前文档）
	fullDocUrl := fmt.Sprintf("https://ku.baidu-int.com/knowledge/%s/%s/%s",
		pathPrefix, repositoryGuid, docGuid)

	return map[string]string{
		"repositoryGuid": repositoryGuid,
		"docGuid":        docGuid,        // 当前文档ID，也是子文档的parentDocGuid
		"fullDocUrl":     fullDocUrl,     // 完整的文档URL
	}, nil
}

func (p *KuDocParser) Validate(params map[string]string) error {
	requiredParams := []string{"repositoryGuid", "docGuid", "fullDocUrl"}
	for _, param := range requiredParams {
		if err := validateRequiredParam(params, param); err != nil {
			return err
		}
	}
	return nil
}
