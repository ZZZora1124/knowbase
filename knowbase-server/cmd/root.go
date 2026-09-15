// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/cache"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/infoflow"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/uuap"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/repository"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/scheduler"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/service"

	"github.com/spf13/cobra"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/config"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/database"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/server"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/log"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/types/constants"
)

// NewRootCommand 创建根命令
func NewRootCommand() *cobra.Command {
	var configPath string

	var command = cobra.Command{
		Use:   "knowbase-server",
		Short: "knowbase-server is the service to manage the knowledge",
		RunE: func(cmd *cobra.Command, args []string) error {
			// 初始化系统配置
			configPath = os.Getenv(constants.EnvConfigPath)
			if len(configPath) < 1 {
				return fmt.Errorf("未设置环境变量: %s", constants.EnvConfigPath)
			}

			cfg, err := config.LoadConfig(configPath)
			if err != nil {
				return fmt.Errorf("加载配置文件失败: %w", err)
			}

			// 初始化日志
			err = log.InitLogger(
				cfg.Log.Level,
				cfg.Log.FilePath,
				cfg.Log.MaxSize,
				cfg.Log.MaxBackups,
				cfg.Log.MaxAge,
				cfg.Log.Compress,
			)
			if err != nil {
				return fmt.Errorf("初始化日志失败: %w", err)
			}

			log.Logger.Info("启动知识方舟服务...")

			// 初始化缓存
			log.Logger.Info("init cache driver start")
			err = cache.NewCache(cfg.CacheCfg)
			if err != nil {
				return fmt.Errorf("init cache driver error: %w", err)
			}
			log.Logger.Info("init cache driver done")

			// 初始化数据库
			dbConnection := cfg.GetDBConnection()
			if dbConnection == "" {
				return fmt.Errorf("数据库连接字符串为空，请设置环境变量: %s", cfg.DBCfg.Connection)
			}

			err = database.InitMySQL(dbConnection)
			if err != nil {
				return fmt.Errorf("初始化数据库失败: %w", err)
			}
			log.Logger.Info("数据库初始化成功")

			// 初始化UUAP客户端
			log.Logger.Info("初始化UUAP客户端...")
			_, err = uuap.GetClient()
			if err != nil {
				return fmt.Errorf("初始化UUAP客户端失败: %w", err)
			}

			// 初始化Web服务
			port := os.Getenv(constants.EnvPort)
			if len(port) < 1 {
				return fmt.Errorf("未设置环境变量: %s", constants.EnvPort)
			}

			ks, err := server.NewKnowbaseServer(port, cfg)
			if err != nil || ks == nil {
				return fmt.Errorf("初始化knowbase服务失败: %w", err)
			}

			// 初始化定时清理调度器
			var cleanupScheduler *scheduler.InfoFlowCleanupScheduler
			if cfg.InfoFlowConfig != nil && cfg.InfoFlowConfig.Cleanup != nil {
				ormer := database.GetOrm()
				infoflowRepo := repository.NewInfoFlowRepository(ormer)
				infoflowService := service.NewInfoFlowService(infoflowRepo)

				cleanupScheduler = scheduler.NewInfoFlowCleanupScheduler(infoflowService, cfg.InfoFlowConfig.Cleanup)
				if err := cleanupScheduler.Start(); err != nil {
					log.Logger.Errorf("启动如流消息清理调度器失败: %v", err)
				}
			}

			// 初始化群成员缓存刷新调度器
			var groupCacheRefreshScheduler *scheduler.InfoFlowGroupCacheRefreshScheduler
			if cfg.InfoFlowConfig != nil && cfg.InfoFlowConfig.GroupCacheRefresh != nil {
				ormer := database.GetOrm()
				memberRepo := repository.NewKnowledgebaseMemberRepository(ormer)
				infoFlowClient := infoflow.GetClient()

				groupCacheRefreshScheduler = scheduler.NewInfoFlowGroupCacheRefreshScheduler(
					memberRepo,
					infoFlowClient,
					cfg.InfoFlowConfig.GroupCacheRefresh,
				)
				if err := groupCacheRefreshScheduler.Start(); err != nil {
					log.Logger.Errorf("启动群成员缓存刷新调度器失败: %v", err)
				}
			}

			// Initializing the server in a goroutine so that
			// it won't block the graceful shutdown handling below
			go func() {
				log.Logger.Info("knowbase web service start")
				ks.Run()
			}()

			// Wait for interrupt signal to gracefully shutdown the server with
			// a timeout of 5 seconds.
			quit := make(chan os.Signal, 1)
			// kill (no param) default send syscall.SIGTERM
			// kill -2 is syscall.SIGINT
			// kill -9 is syscall.SIGKILL but can't be catch, so don't need add it
			signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
			<-quit
			log.Logger.Info("Shutting down server...")

			// 停止调度器
			if cleanupScheduler != nil {
				cleanupScheduler.Stop()
			}
			if groupCacheRefreshScheduler != nil {
				groupCacheRefreshScheduler.Stop()
			}

			// The context is used to inform the server it has 5 seconds to finish
			// the request it is currently handling
			newCtx, newCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer newCancel()

			if err := ks.ShutDown(newCtx); err != nil {
				log.Logger.Fatalf("Server forced to shutdown, err: %v", err.Error())
			}

			log.Logger.Info("Server exiting")
			return nil
		},
	}

	return &command
}
