// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package server

import (
	"context"
	"fmt"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/log"
	"net/http"
	"os"

	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/config"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/storage"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/utils"
)

// KnowbaseServer 知识方舟服务
type KnowbaseServer struct {
	port      string
	cfg       *config.Config
	server    *http.Server
	bosClient *storage.BOSClient
}

// NewKnowbaseServer 创建知识方舟服务实例
func NewKnowbaseServer(port string, cfg *config.Config) (*KnowbaseServer, error) {
	// 初始化BOS存储
	bosAK := cfg.GetBOSAK()
	bosSK := cfg.GetBOSSK()
	if bosAK == "" || bosSK == "" {
		return nil, fmt.Errorf("BOS凭证为空，请设置环境变量: %s, %s", cfg.BOSConfig.AK, cfg.BOSConfig.SK)
	}

	bosClient, err := storage.NewBOSClient(
		cfg.BOSConfig.Endpoint,
		bosAK,
		bosSK,
		cfg.BOSConfig.Bucket,
	)
	if err != nil {
		return nil, fmt.Errorf("初始化BOS存储失败: %w", err)
	}
	log.Logger.Infof("BOS存储初始化成功, bucket: %s", cfg.BOSConfig.Bucket)

	// 设置JWT密钥（从环境变量读取，或使用默认值）
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret != "" {
		utils.SetJWTSecret(jwtSecret)
	}

	// 设置路由
	router := api.SetupRouter(bosClient)

	// 创建HTTP服务
	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	return &KnowbaseServer{
		port:      port,
		cfg:       cfg,
		server:    server,
		bosClient: bosClient,
	}, nil
}

// Run 启动服务
func (s *KnowbaseServer) Run() error {
	log.Logger.Infof("服务启动, 端口: %s", s.port)
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Logger.Errorf("服务启动失败, %v", err)
		return err
	}
	return nil
}

// ShutDown 优雅关闭服务
func (s *KnowbaseServer) ShutDown(ctx context.Context) error {
	log.Logger.Info("开始关闭服务...")
	if err := s.server.Shutdown(ctx); err != nil {
		log.Logger.Errorf("服务关闭失败, %v", err)
		return err
	}
	log.Logger.Info("服务已关闭")
	return nil
}

// GetBOSClient 获取BOS客户端
func (s *KnowbaseServer) GetBOSClient() *storage.BOSClient {
	return s.bosClient
}
