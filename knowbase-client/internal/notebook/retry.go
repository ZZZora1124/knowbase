// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package notebook

import (
	"context"
	"fmt"
	"math"
	"time"

	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/utils"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

const (
	defaultMaxRetries = 3
	defaultBaseDelay  = 1 * time.Second
)

// RetryableFunc 可重试的函数类型
type RetryableFunc func() ([]byte, error)

// WithRetry 带指数退避的重试执行
func WithRetry(ctx context.Context, operation string, fn RetryableFunc) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= defaultMaxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(math.Pow(2, float64(attempt-1))) * defaultBaseDelay
			utils.PrintlnWarn("Retrying %s (attempt %d/%d) after %v...", operation, attempt, defaultMaxRetries, delay)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		result, err := fn()
		if err == nil {
			return result, nil
		}

		lastErr = err
		if detailedErr, ok := err.(*types.DetailedError); ok {
			if detailedErr.ErrorType == types.ErrorTypeAuth ||
				detailedErr.ErrorType == types.ErrorTypeValidation ||
				detailedErr.ErrorType == types.ErrorTypeConfig {
				return nil, err
			}
		}
	}
	return nil, fmt.Errorf("%s failed after %d retries: %w", operation, defaultMaxRetries, lastErr)
}
