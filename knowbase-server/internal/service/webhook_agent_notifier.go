// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/config"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/log"
)

// WebhookAgentNotifier 基于Webhook的Agent通知器
type WebhookAgentNotifier struct {
	configs        map[string]*config.AgentWebhookConfig // agentCode -> config
	httpClient     *http.Client
	mu             sync.RWMutex
	defaultTimeout time.Duration
}

// NewWebhookAgentNotifier 创建Webhook通知器
func NewWebhookAgentNotifier(configs []*config.AgentWebhookConfig) *WebhookAgentNotifier {
	notifier := &WebhookAgentNotifier{
		configs: make(map[string]*config.AgentWebhookConfig),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		defaultTimeout: 30 * time.Second,
	}

	for _, cfg := range configs {
		if cfg.Enabled {
			notifier.configs[cfg.AgentCode] = cfg
		}
	}

	return notifier
}

// RegisterAgent 动态注册Agent配置
func (n *WebhookAgentNotifier) RegisterAgent(cfg *config.AgentWebhookConfig) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if cfg.Enabled {
		n.configs[cfg.AgentCode] = cfg
	} else {
		delete(n.configs, cfg.AgentCode)
	}
}

// UnregisterAgent 注销Agent配置
func (n *WebhookAgentNotifier) UnregisterAgent(agentCode string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	delete(n.configs, agentCode)
}

// Notify 发送应用通知到Agent服务端
func (n *WebhookAgentNotifier) Notify(notification *ApplyNotification) error {
	n.mu.RLock()
	config, exists := n.configs[notification.AgentCode]
	n.mu.RUnlock()

	if !exists {
		log.Logger.Warnf("Agent [%s] 未配置webhook，跳过通知", notification.AgentCode)
		return nil // 未配置的Agent直接跳过，不报错
	}

	return n.sendWebhook(config, notification)
}

// NotifyBatch 批量发送通知
func (n *WebhookAgentNotifier) NotifyBatch(notifications []*ApplyNotification) []error {
	errors := make([]error, len(notifications))
	var wg sync.WaitGroup

	for i, notification := range notifications {
		wg.Add(1)
		go func(idx int, notif *ApplyNotification) {
			defer wg.Done()
			errors[idx] = n.Notify(notif)
		}(i, notification)
	}

	wg.Wait()
	return errors
}

// sendWebhook 发送Webhook请求
func (n *WebhookAgentNotifier) sendWebhook(cfg *config.AgentWebhookConfig, notification *ApplyNotification) error {
	// 序列化通知数据
	payload, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("序列化通知数据失败: %w", err)
	}

	// 设置超时（配置中的Timeout为秒数）
	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = n.defaultTimeout
	}

	// 重试配置
	retryCount := cfg.RetryCount
	if retryCount == 0 {
		retryCount = 3
	}
	retryDelay := time.Duration(cfg.RetryDelay) * time.Second
	if retryDelay == 0 {
		retryDelay = 1 * time.Second
	}

	var lastErr error
	for attempt := 0; attempt <= retryCount; attempt++ {
		if attempt > 0 {
			log.Logger.Infof("Agent [%s] 第 %d 次重试推送", cfg.AgentCode, attempt)
			time.Sleep(retryDelay)
		}

		err := n.doRequest(cfg, payload, timeout)
		if err == nil {
			log.Logger.Infof("Agent [%s] 推送成功, action=%s, configUuid=%s",
				cfg.AgentCode, notification.Action, notification.ConfigUUID)
			return nil
		}

		lastErr = err
		log.Logger.Warnf("Agent [%s] 推送失败 (attempt %d/%d): %v",
			cfg.AgentCode, attempt+1, retryCount+1, err)
	}

	return fmt.Errorf("Agent [%s] 推送失败，已重试 %d 次: %w", cfg.AgentCode, retryCount, lastErr)
}

// doRequest 执行HTTP请求
func (n *WebhookAgentNotifier) doRequest(cfg *config.AgentWebhookConfig, payload []byte, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.WebhookURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Knowbase-Event", "knowledge.apply")
	req.Header.Set("X-Knowbase-Timestamp", fmt.Sprintf("%d", time.Now().Unix()))

	// 添加自定义请求头
	if cfg.Headers != nil {
		for key, value := range cfg.Headers {
			req.Header.Set(key, value)
		}
	}

	// 发送请求
	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, _ := io.ReadAll(resp.Body)

	// 检查响应状态
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("响应状态码异常: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// generateHmacSignature 生成HMAC-SHA256签名
func generateHmacSignature(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}

// NoopAgentNotifier 空实现的通知器（用于测试或禁用通知场景）
type NoopAgentNotifier struct{}

// NewNoopAgentNotifier 创建空通知器
func NewNoopAgentNotifier() *NoopAgentNotifier {
	return &NoopAgentNotifier{}
}

// Notify 空实现
func (n *NoopAgentNotifier) Notify(notification *ApplyNotification) error {
	log.Logger.Debugf("NoopAgentNotifier: 跳过通知 agent=%s, action=%s",
		notification.AgentCode, notification.Action)
	return nil
}

// NotifyBatch 空实现
func (n *NoopAgentNotifier) NotifyBatch(notifications []*ApplyNotification) []error {
	return make([]error, len(notifications))
}