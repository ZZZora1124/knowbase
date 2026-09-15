// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package brain

import (
	"context"
	"fmt"
	"os"
	"time"

	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/auth"
	br "icode.baidu.com/baidu/devops-ai/knowbase-client/internal/brain"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/utils"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

// initBrainEnv 初始化 brain 环境，返回 BrainClient、带超时的 context 和 cancel 函数
func initBrainEnv() (*br.BrainClient, context.Context, context.CancelFunc) {
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

	brainUUID := brainKbUUID
	if brainUUID == "" {
		brainUUID = os.Getenv(types.EnvKnowbaseBrainUUID)
	}
	if brainUUID == "" {
		utils.PrintlnError("brain UUID is required (via --kb-uuid or env %s)", types.EnvKnowbaseBrainUUID)
		os.Exit(types.ExitRunSubCMDError)
	}

	c := br.NewBrainClientWithCredential(cred, brainUUID)
	ctx, cancel := newContextWithTimeout(brainTimeout)
	return c, ctx, cancel
}

// newContextWithTimeout creates a context with the given timeout in seconds.
// If timeout <= 0, returns a plain context.Background() with a no-op cancel.
func newContextWithTimeout(timeout int) (context.Context, context.CancelFunc) {
	if timeout > 0 {
		return context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	}
	return context.Background(), func() {}
}

func printResultRows(results []br.SearchResult) {
	for _, r := range results {
		runes := []rune(r.CompiledTruth)
		chunk := r.CompiledTruth
		if len(runes) > 100 {
			chunk = string(runes[:100])
		}
		fmt.Printf("[%.4f] %s -- %s\n", r.Score, r.Slug, chunk)
	}
}
