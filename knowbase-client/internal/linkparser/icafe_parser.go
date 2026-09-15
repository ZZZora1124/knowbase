// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package linkparser

import (
	"fmt"
	"regexp"
	"strings"
)

// ICafeParser iCafe卡片链接解析器
type ICafeParser struct{}

// Parse 解析iCafe卡片链接
// 格式1: https://console.cloud.baidu-int.com/devops/icafe/issue/<issueId>/show
// 格式2: https://console.cloud.baidu-int.com/devops/icafe/space/<space>/planbox/<planId>/issue
// 示例1: https://console.cloud.baidu-int.com/devops/icafe/issue/cloud-iPipe-9223/show
// 示例2: https://console.cloud.baidu-int.com/devops/icafe/space/dev-partner/planbox/1097518/issue
func (p *ICafeParser) Parse(link string) (map[string]string, error) {
	params := make(map[string]string)

	// 尝试匹配格式1: 单个卡片
	pattern1 := `https://console\.cloud\.baidu-int\.com/devops/icafe/issue/([^/]+)/show`
	re1 := regexp.MustCompile(pattern1)
	matches1 := re1.FindStringSubmatch(link)

	if len(matches1) == 2 {
		params["issueId"] = strings.TrimSpace(matches1[1])
		params["type"] = "single" // 标记为单个卡片
		return params, nil
	}

	// 尝试匹配格式2: planbox
	pattern2 := `https://console\.cloud\.baidu-int\.com/devops/icafe/space/([^/]+)/planbox/(\d+)/issue`
	re2 := regexp.MustCompile(pattern2)
	matches2 := re2.FindStringSubmatch(link)

	if len(matches2) == 3 {
		params["space"] = strings.TrimSpace(matches2[1])
		params["planId"] = strings.TrimSpace(matches2[2])
		params["type"] = "planbox" // 标记为planbox
		return params, nil
	}

	return nil, fmt.Errorf("invalid iCafe card link format, expected: https://console.cloud.baidu-int.com/devops/icafe/issue/<issueId>/show or .../space/<space>/planbox/<planId>/issue")
}

func (p *ICafeParser) Validate(params map[string]string) error {
	linkType := params["type"]
	if linkType == "" {
		return fmt.Errorf("type parameter is required")
	}

	switch linkType {
	case "single":
		return validateRequiredParam(params, "issueId")
	case "planbox":
		if err := validateRequiredParam(params, "space"); err != nil {
			return err
		}
		return validateRequiredParam(params, "planId")
	default:
		return fmt.Errorf("unknown iCafe link type: %s", linkType)
	}
}
