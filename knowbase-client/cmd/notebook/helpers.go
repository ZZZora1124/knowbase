// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package notebook

import (
	"context"
	"os"
	"text/tabwriter"

	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/auth"
	nb "icode.baidu.com/baidu/devops-ai/knowbase-client/internal/notebook"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/utils"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

// initNotebookEnv 初始化 notebook 环境
func initNotebookEnv() (*nb.NotebookService, context.Context) {
	// 处理环境变量
	envVal := os.Getenv(types.EnvKnowbaseEnv)
	if envVal == "" {
		envVal = "prod"
	}
	if err := types.InitEnv(envVal); err != nil {
		utils.PrintlnError("Error: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	// 获取认证凭证
	cred, err := auth.GetCredential()
	if err != nil {
		utils.PrintlnError("%v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	svc := nb.NewNotebookServiceWithCredential(cred)
	ctx := context.Background()
	return svc, ctx
}

// outputResult 根据 --output 标志选择输出格式
func outputResult(data interface{}, tableFunc func()) {
	if outputFormat == "json" {
		utils.OutputJSON(data)
	} else {
		tableFunc()
	}
}

// newTabWriter 创建制表符对齐的写入器
func newTabWriter() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
}
