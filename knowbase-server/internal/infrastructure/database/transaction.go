// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package database

import (
	"context"

	"github.com/beego/beego/v2/client/orm"
)

// TxContext 事务上下文
// 用于在事务中传递 TxOrmer 和 context.Context（携带TraceID等信息）
// nil TxContext 表示使用默认 ormer（非事务操作）
type TxContext struct {
	ctx context.Context
	tx  orm.TxOrmer
}

// NewTxContext 创建事务上下文
func NewTxContext(tx orm.TxOrmer) *TxContext {
	return &TxContext{
		ctx: context.Background(),
		tx:  tx,
	}
}

// NewTxContextWithContext 创建带context的事务上下文
func NewTxContextWithContext(ctx context.Context, tx orm.TxOrmer) *TxContext {
	return &TxContext{
		ctx: ctx,
		tx:  tx,
	}
}

// Context 获取context.Context（用于日志记录等场景）
func (c *TxContext) Context() context.Context {
	if c == nil || c.ctx == nil {
		return context.Background()
	}
	return c.ctx
}

// Ormer 获取 TxOrmer，如果事务存在则返回事务 ormer，否则返回 nil
// 注意：返回类型为 orm.TxOrmer，它实现了 QueryExecutor 和 TxCommitter 接口
func (c *TxContext) Ormer() orm.TxOrmer {
	if c == nil {
		return nil
	}
	return c.tx
}

// TransactionManager 事务管理器
// 封装事务的开启、提交、回滚
// 注意：此结构体不持有 Ormer 实例，每次事务都会创建新的 Ormer，确保并发安全
type TransactionManager struct{}

// NewTransactionManager 创建事务管理器
func NewTransactionManager() *TransactionManager {
	return &TransactionManager{}
}

// ExecuteInTransaction 在事务中执行函数
// 如果函数返回 error，则回滚事务；否则提交事务
// 每次调用都会创建新的 Ormer 实例，确保 goroutine 安全
//
// 示例:
//
//	err := txManager.ExecuteInTransaction(func(ctx *TxContext) error {
//	    if err := repo.Create(ctx, entity); err != nil {
//	        return err
//	    }
//	    return nil
//	})
func (m *TransactionManager) ExecuteInTransaction(fn func(ctx *TxContext) error) error {
	return m.ExecuteInTransactionWithContext(context.Background(), fn)
}

// ExecuteInTransactionWithContext 在事务中执行函数（带context，用于日志追踪）
func (m *TransactionManager) ExecuteInTransactionWithContext(ctx context.Context, fn func(ctx *TxContext) error) error {
	// 每次执行事务时创建新的 Ormer，避免 goroutine 共享
	ormer := GetOrm()
	tx, err := ormer.Begin()
	if err != nil {
		return err
	}

	txCtx := NewTxContextWithContext(ctx, tx)

	if err := fn(txCtx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// Begin 开始事务，返回事务上下文
// 手动管理事务时使用，需要显式调用 Commit 或 Rollback
// 每次调用都会创建新的 Ormer 实例，确保 goroutine 安全
//
// 示例:
//
//	ctx, err := txManager.Begin()
//	if err != nil {
//	    return err
//	}
//	defer txManager.Rollback(ctx) // 确保异常时回滚
//
//	// 执行业务逻辑
//	if err := repo.Create(ctx, entity); err != nil {
//	    return err
//	}
//
//	// 提交事务
//	return txManager.Commit(ctx)
func (m *TransactionManager) Begin() (*TxContext, error) {
	return m.BeginWithContext(context.Background())
}

// BeginWithContext 开始事务（带context，用于日志追踪）
func (m *TransactionManager) BeginWithContext(ctx context.Context) (*TxContext, error) {
	// 每次开始事务时创建新的 Ormer，避免 goroutine 共享
	ormer := GetOrm()
	tx, err := ormer.Begin()
	if err != nil {
		return nil, err
	}
	return NewTxContextWithContext(ctx, tx), nil
}

// Commit 提交事务
func (m *TransactionManager) Commit(ctx *TxContext) error {
	if ctx == nil || ctx.tx == nil {
		return nil
	}
	return ctx.tx.Commit()
}

// Rollback 回滚事务
func (m *TransactionManager) Rollback(ctx *TxContext) error {
	if ctx == nil || ctx.tx == nil {
		return nil
	}
	return ctx.tx.Rollback()
}
