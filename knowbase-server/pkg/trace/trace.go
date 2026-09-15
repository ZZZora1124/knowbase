// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package trace

import (
	"context"

	"github.com/google/uuid"
)

type ctxKey struct{}

// NewTraceID 生成新的TraceID
func NewTraceID() string {
	return uuid.New().String()
}

// WithTraceID 将TraceID存入context
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, ctxKey{}, traceID)
}

// FromContext 从context提取TraceID
func FromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(ctxKey{}).(string); ok {
		return v
	}
	return ""
}
