// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package database

import (
	"fmt"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/go-sql-driver/mysql" // MySQL驱动
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/log"
)

func init() {
	orm.RegisterModel(
		new(domain.InfoFlowMessage),
		new(domain.InfoFlowMessageDeletionLog),
		new(domain.InfoFlowOneOnOneMessage),
		new(domain.Knowledge),
		new(domain.KnowledgeDownloadMetric),
		new(domain.Knowledgebase),
		new(domain.KnowledgebaseDirectory),
		new(domain.KnowledgebaseMember),
		new(domain.KnowledgebaseOperationLog),
		new(domain.KnowledgebaseTag),
		new(domain.KnowledgeTagRelation),
		new(domain.OpenAPIClient),
		new(domain.KnowledgeVersion),
		new(domain.KnowledgeApplyConfig),
		new(domain.KnowledgeApplyRecord),
		new(domain.KnowledgeConsolidationMetric),
	)
}

// InitMySQL 初始化MySQL数据库连接
func InitMySQL(dsn string) error {
	if err := orm.RegisterDriver("mysql", orm.DRMySQL); err != nil {
		return fmt.Errorf("注册MySQL驱动失败: %w", err)
	}

	if err := orm.RegisterDataBase("default", "mysql", dsn); err != nil {
		return fmt.Errorf("注册MySQL数据库失败: %w", err)
	}

	db, err := orm.GetDB("default")
	if err != nil {
		return fmt.Errorf("获取数据库连接失败: %w", err)
	}

	if err := db.Ping(); err != nil {
		return fmt.Errorf("数据库连接测试失败: %w", err)
	}

	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(100)

	// 配置ORM日志输出到日志文件
	orm.Debug = true
	if writer := log.GetWriter(); writer != nil {
		orm.DebugLog = orm.NewLog(writer)
	}

	return nil
}

// GetOrm 获取ORM对象
func GetOrm() orm.Ormer {
	return orm.NewOrm()
}
