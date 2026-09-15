// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package api

import (
	"github.com/gin-gonic/gin"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/handler"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/middleware"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/storage"
)

// SetupRouter 设置路由
func SetupRouter(bosClient *storage.BOSClient) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()

	// 全局中间件（注意顺序：Trace必须在Logger之前，以便Logger能获取TraceID）
	r.Use(middleware.Trace())    // 1. 生成/提取TraceID
	r.Use(middleware.Recovery()) // 2. 异常恢复
	r.Use(middleware.Logger())   // 3. 请求日志（含TraceID）
	r.Use(middleware.CORS())     // 4. 跨域处理

	// 健康检查接口（无需认证）
	r.GET("/health", handler.HealthCheck)

	// 初始化 Handlers
	infoFlowHandler := handler.NewInfoFlowHandler()
	metricHandler := handler.NewMetricHandler()
	knowledgebaseHandler := handler.NewKnowledgebaseHandler()
	memberHandler := handler.NewKnowledgebaseMemberHandler()
	tagHandler := handler.NewTagHandler()
	directoryHandler := handler.NewDirectoryHandler()
	knowledgeHandler := handler.NewKnowledgeHandler(bosClient)
	openapiClientHandler := handler.NewOpenAPIClientHandler()
	versionHandler := handler.NewKnowledgeVersionHandler()
	applyHandler := handler.NewKnowledgeApplyHandler()
	operationLogHandler := handler.NewOperationLogHandler()
	brainHandler := handler.NewBrainHandler()

	// API v1路由组
	v1 := r.Group("/api/v1")
	{
		// 不需要认证的接口
		// 例如：登录、注册等
		// v1.POST("/login", handler.Login)

		// 需要Cloud认证的接口组
		cloudAuth := v1.Group("")
		cloudAuth.Use(middleware.AuthCloud())
		{
			// 知识本管理
			knowledgebases := cloudAuth.Group("/knowledgebases")
			{
				knowledgebases.POST("", knowledgebaseHandler.CreateKnowledgebase)
				knowledgebases.GET("/my", knowledgebaseHandler.ListMyKnowledgebases) // 新增：查询用户有权限的知识本
				knowledgebases.GET("", knowledgebaseHandler.ListKnowledgebases)
				knowledgebases.GET("/name/:name", knowledgebaseHandler.GetKnowledgebaseByName)
				knowledgebases.GET("/:kb_uuid", knowledgebaseHandler.GetKnowledgebase)
				knowledgebases.PUT("/:kb_uuid", knowledgebaseHandler.UpdateKnowledgebase)
				knowledgebases.DELETE("/:kb_uuid", knowledgebaseHandler.DeleteKnowledgebase)

				// 成员
				knowledgebases.POST("/:kb_uuid/members", memberHandler.AddMember)
				knowledgebases.GET("/:kb_uuid/members", memberHandler.ListMembers)
				knowledgebases.GET("/:kb_uuid/members/expanded", memberHandler.ListExpandedMembers)
				knowledgebases.GET("/:kb_uuid/admins", memberHandler.ListAdmins)
				knowledgebases.PUT("/:kb_uuid/members/:uuid", memberHandler.UpdateMemberRole)
				knowledgebases.DELETE("/:kb_uuid/members/:uuid", memberHandler.RemoveMember)

				// 标签
				knowledgebases.POST("/:kb_uuid/tags", tagHandler.CreateTag)
				knowledgebases.GET("/:kb_uuid/tags", tagHandler.ListTags)
				knowledgebases.PUT("/:kb_uuid/tags/:uuid", tagHandler.UpdateTag)
				knowledgebases.DELETE("/:kb_uuid/tags/:uuid", tagHandler.DeleteTag)

				// 目录
				knowledgebases.POST("/:kb_uuid/directories", directoryHandler.CreateDirectory)
				knowledgebases.GET("/:kb_uuid/directories", directoryHandler.ListDirectories)
				knowledgebases.GET("/:kb_uuid/directories/tree", directoryHandler.GetDirectoryTree)
				knowledgebases.GET("/:kb_uuid/directories/:uuid", directoryHandler.GetDirectory)
				knowledgebases.PUT("/:kb_uuid/directories/:uuid", directoryHandler.UpdateDirectory)
				knowledgebases.DELETE("/:kb_uuid/directories/:uuid", directoryHandler.DeleteDirectory)
				knowledgebases.POST("/:kb_uuid/directories/:uuid/move", directoryHandler.MoveDirectory)

				// 知识
				knowledgebases.POST("/:kb_uuid/knowledge", knowledgeHandler.CreateKnowledge)
				knowledgebases.GET("/:kb_uuid/knowledge", knowledgeHandler.ListKnowledge)
				knowledgebases.GET("/:kb_uuid/knowledge/by-tags", knowledgeHandler.ListKnowledgeByTags)

				// 操作历史
				knowledgebases.GET("/operation-logs", operationLogHandler.ListOperationLogs)
			}

			// 知识操作（不依赖知识本UUID）
			knowledge := cloudAuth.Group("/knowledge")
			{
				knowledge.POST("/batch", knowledgeHandler.GetKnowledgeBatch)
				knowledge.GET("/:uuid", knowledgeHandler.GetKnowledge)
				knowledge.PUT("/:uuid", knowledgeHandler.UpdateKnowledge)
				knowledge.POST("/:uuid/file", knowledgeHandler.UpdateKnowledgeFile)
				knowledge.DELETE("/:uuid", knowledgeHandler.DeleteKnowledge)
				knowledge.POST("/:uuid/move", knowledgeHandler.MoveKnowledge)
				knowledge.POST("/:uuid/tags", knowledgeHandler.BindTags)
				knowledge.DELETE("/:uuid/tags", knowledgeHandler.UnbindTags)
				knowledge.GET("/:uuid/tags", knowledgeHandler.GetKnowledgeTags)

				// 版本管理
				knowledge.GET("/:uuid/versions", versionHandler.ListVersions)
				knowledge.GET("/:uuid/versions/:version", versionHandler.GetVersion)
				knowledge.POST("/:uuid/rollback", versionHandler.RollbackToVersion)

				// 批量版本查询
				knowledge.POST("/versions", versionHandler.BatchListVersions)
				knowledge.POST("/batch-with-version", versionHandler.BatchGetWithVersion)
			}

			// 知识应用管理
			applyConfigs := cloudAuth.Group("/knowledge-apply/config")
			{
				applyConfigs.POST("", applyHandler.CreateOrUpdateConfig)
				applyConfigs.GET("", applyHandler.ListConfigs)
				applyConfigs.GET("/:uuid", applyHandler.GetConfig)
				applyConfigs.DELETE("/:uuid", applyHandler.DeleteConfig)
				applyConfigs.GET("/:uuid/history", applyHandler.GetConfigHistory)
				applyConfigs.POST("/:uuid/knowledge", applyHandler.AddKnowledge)
				applyConfigs.DELETE("/:uuid/knowledge", applyHandler.RemoveKnowledge)
				applyConfigs.PATCH("/:uuid/status", applyHandler.UpdateConfigStatus)
				applyConfigs.POST("/:uuid/retry", applyHandler.RetryPush)
			}

			// 如流群聊信息
			infoflowCloud := cloudAuth.Group("/infoflow")
			{
				infoflowCloud.POST("/group/info", infoFlowHandler.HandleGetGroupInfo)
			}
		}

		// 需要Header认证的接口组
		headerAuth := v1.Group("")
		headerAuth.Use(middleware.AuthHeader())
		{
			// 如流消息上报(按照x-auth-client-token进行鉴权，给如流QA荣松用的)，查询按照（x-ac-token和x-auth-client-token）鉴权
			infoflow := headerAuth.Group("/infoflow")
			{
				infoflow.POST("/group/upload", infoFlowHandler.HandleGroupMessageUpload)
				infoflow.POST("/group/messages", infoFlowHandler.HandleListGroupMessages)
				infoflow.POST("/group/messages/delete-internal", infoFlowHandler.HandleDeleteGroupMessagesInternal)
				infoflow.POST("/group/messages/delete", infoFlowHandler.HandleDeleteGroupMessages)
				infoflow.POST("/group/event", infoFlowHandler.HandleGroupMemberEvent)
				infoflow.POST("/group/cache/refresh", infoFlowHandler.HandleRefreshGroupMemberCache)
				infoflow.POST("/oneone/upload", infoFlowHandler.HandleOneOnOneMessageUpload)
				infoflow.POST("/oneone/messages", infoFlowHandler.HandleListOneOnOneMessages)
			}

			// 知识下载数据上报
			metrics := headerAuth.Group("/metrics")
			{
				metrics.POST("/download", metricHandler.HandleReportDownload)
				metrics.POST("/consolidation", metricHandler.HandleReportConsolidation)
			}
		}
	}

	// Open API v1路由组 (供业务方系统调用)
	openAPI := r.Group("/open-api/v1")
	{
		openAPIAuth := openAPI.Group("")
		openAPIAuth.Use(middleware.AuthOpenAPI())
		{
			// 知识本管理
			knowledgebases := openAPIAuth.Group("/knowledgebases")
			{
				knowledgebases.POST("", knowledgebaseHandler.CreateKnowledgebase)
				knowledgebases.GET("/my", knowledgebaseHandler.ListMyKnowledgebases) // 新增：查询用户有权限的知识本
				knowledgebases.GET("", knowledgebaseHandler.ListKnowledgebases)
				knowledgebases.GET("/name/:name", knowledgebaseHandler.GetKnowledgebaseByName)
				knowledgebases.GET("/:kb_uuid", knowledgebaseHandler.GetKnowledgebase)
				knowledgebases.PUT("/:kb_uuid", knowledgebaseHandler.UpdateKnowledgebase)
				knowledgebases.DELETE("/:kb_uuid", knowledgebaseHandler.DeleteKnowledgebase)

				// 成员
				knowledgebases.POST("/:kb_uuid/members", memberHandler.AddMember)
				knowledgebases.GET("/:kb_uuid/members", memberHandler.ListMembers)
				knowledgebases.GET("/:kb_uuid/members/expanded", memberHandler.ListExpandedMembers)
				knowledgebases.GET("/:kb_uuid/admins", memberHandler.ListAdmins)
				knowledgebases.PUT("/:kb_uuid/members/:uuid", memberHandler.UpdateMemberRole)
				knowledgebases.DELETE("/:kb_uuid/members/:uuid", memberHandler.RemoveMember)

				// 标签
				knowledgebases.POST("/:kb_uuid/tags", tagHandler.CreateTag)
				knowledgebases.GET("/:kb_uuid/tags", tagHandler.ListTags)
				knowledgebases.PUT("/:kb_uuid/tags/:uuid", tagHandler.UpdateTag)
				knowledgebases.DELETE("/:kb_uuid/tags/:uuid", tagHandler.DeleteTag)

				// 目录
				knowledgebases.POST("/:kb_uuid/directories", directoryHandler.CreateDirectory)
				knowledgebases.GET("/:kb_uuid/directories", directoryHandler.ListDirectories)
				knowledgebases.GET("/:kb_uuid/directories/tree", directoryHandler.GetDirectoryTree)
				knowledgebases.GET("/:kb_uuid/directories/:uuid", directoryHandler.GetDirectory)
				knowledgebases.PUT("/:kb_uuid/directories/:uuid", directoryHandler.UpdateDirectory)
				knowledgebases.DELETE("/:kb_uuid/directories/:uuid", directoryHandler.DeleteDirectory)
				knowledgebases.POST("/:kb_uuid/directories/:uuid/move", directoryHandler.MoveDirectory)

				// 知识
				knowledgebases.POST("/:kb_uuid/knowledge", knowledgeHandler.CreateKnowledge)
				knowledgebases.GET("/:kb_uuid/knowledge", knowledgeHandler.ListKnowledge)
				knowledgebases.POST("/:kb_uuid/knowledge/by-tags", knowledgeHandler.ListKnowledgeByTags)

				// 操作历史
				knowledgebases.GET("/operation-logs", operationLogHandler.ListOperationLogs)

				// 群聊有权限的知识本列表
				knowledgebases.GET("/group", knowledgebaseHandler.GetGroupKnowledgebases)
			}

			// 知识操作（不依赖知识本UUID）
			knowledge := openAPIAuth.Group("/knowledge")
			{
				knowledge.POST("/batch", knowledgeHandler.GetKnowledgeBatch)
				knowledge.GET("/:uuid", knowledgeHandler.GetKnowledge)
				knowledge.PUT("/:uuid", knowledgeHandler.UpdateKnowledge)
				knowledge.POST("/:uuid/file", knowledgeHandler.UpdateKnowledgeFile)
				knowledge.DELETE("/:uuid", knowledgeHandler.DeleteKnowledge)
				knowledge.POST("/:uuid/move", knowledgeHandler.MoveKnowledge)
				knowledge.POST("/:uuid/tags", knowledgeHandler.BindTags)
				knowledge.DELETE("/:uuid/tags", knowledgeHandler.UnbindTags)
				knowledge.GET("/:uuid/tags", knowledgeHandler.GetKnowledgeTags)

				// 版本管理
				knowledge.GET("/:uuid/versions", versionHandler.ListVersions)
				knowledge.GET("/:uuid/versions/:version", versionHandler.GetVersion)
				knowledge.POST("/:uuid/rollback", versionHandler.RollbackToVersion)

				// 批量版本查询
				knowledge.POST("/versions", versionHandler.BatchListVersions)
				knowledge.POST("/batch-with-version", versionHandler.BatchGetWithVersion)
			}

			// 知识应用管理
			applyConfigs := openAPIAuth.Group("/knowledge-apply/config")
			{
				applyConfigs.POST("", applyHandler.CreateOrUpdateConfig)
				applyConfigs.GET("", applyHandler.ListConfigs)
				applyConfigs.GET("/:uuid", applyHandler.GetConfig)
				applyConfigs.DELETE("/:uuid", applyHandler.DeleteConfig)
				applyConfigs.GET("/:uuid/history", applyHandler.GetConfigHistory)
				applyConfigs.POST("/:uuid/knowledge", applyHandler.AddKnowledge)
				applyConfigs.DELETE("/:uuid/knowledge", applyHandler.RemoveKnowledge)
				applyConfigs.PATCH("/:uuid/status", applyHandler.UpdateConfigStatus)
				applyConfigs.POST("/:uuid/retry", applyHandler.RetryPush)
			}

			// OpenAPI客户端管理
			openapiClients := openAPIAuth.Group("/openapi-clients")
			{
				openapiClients.POST("", openapiClientHandler.CreateClient)
				openapiClients.GET("", openapiClientHandler.ListClients)
				openapiClients.GET("/:token", openapiClientHandler.GetClient)
				openapiClients.PUT("/:token", openapiClientHandler.UpdateClient)
				openapiClients.DELETE("/:token", openapiClientHandler.DeleteClient)
				openapiClients.POST("/:token/disable", openapiClientHandler.DisableClient)
			}

			// Brain 知识提取与检索
			brain := openAPIAuth.Group("/brain")
			{
				brain.PUT("/page", brainHandler.PutPage)
				brain.GET("/page", brainHandler.ListPages)
				brain.GET("/page/*slug", brainHandler.GetPage)
				brain.DELETE("/page/*slug", brainHandler.DeletePage)
				brain.GET("/backlinks/*slug", brainHandler.GetBacklinks)
				brain.GET("/graph/*slug", brainHandler.TraverseGraph)
				brain.GET("/timeline/*slug", brainHandler.GetTimeline)
				brain.POST("/timeline/*slug", brainHandler.AddTimelineEntry)
				brain.GET("/search", brainHandler.SearchPages)
				brain.GET("/query", brainHandler.QueryPages)
				brain.GET("/config/default-model", brainHandler.GetDefaultModel)
				brain.PUT("/config/default-model", brainHandler.SetDefaultModel)
			}
		}
	}

	return r
}
