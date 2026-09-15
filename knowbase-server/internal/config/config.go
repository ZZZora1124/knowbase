// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Config 全局配置结构
type Config struct {
	CacheCfg             *CacheConfig          `mapstructure:"cacheConfig"`
	DBCfg                DBConfig              `mapstructure:"dbCfg"`
	BOSConfig            BOSConfig             `mapstructure:"bosConfig"`
	AuthenticationConfig *AuthenticationConfig `mapstructure:"authentication"`
	UUAPConfig           *UUAPConfig           `mapstructure:"uuap"`
	InfoFlowConfig       *InfoFlowConfig       `mapstructure:"infoflow"`
	AgentWebhooks        []*AgentWebhookConfig `mapstructure:"agentWebhooks"`
	Log                  LogConfig             `mapstructure:"log"`
	ContextMapConfig     *ContextMapConfig     `mapstructure:"contextMap"`
}

// DBConfig 数据库配置
type DBConfig struct {
	Connection string `mapstructure:"connection"` // 环境变量Key
}

// BOSConfig BOS对象存储配置
type BOSConfig struct {
	AK       string `mapstructure:"ak"`       // 环境变量Key
	SK       string `mapstructure:"sk"`       // 环境变量Key
	Bucket   string `mapstructure:"bucket"`   // Bucket名称
	Endpoint string `mapstructure:"endpoint"` // Endpoint地址
}

// CacheConfig 缓存配置
type CacheConfig struct {
	Redis *RedisConfig `mapstructure:"redis"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// AuthenticationConfig 认证服务配置
type AuthenticationConfig struct {
	URL          string `mapstructure:"url"`
	ClientId     string `mapstructure:"clientId"`
	ClientSecret string `mapstructure:"clientSecret"`
	RetryCount   int    `mapstructure:"retryCount"`
	Timeout      int    `mapstructure:"timeout"`
}

// UUAPConfig UUAP邮件组API配置
type UUAPConfig struct {
	URL          string `mapstructure:"url"`
	ClientId     string `mapstructure:"clientId"`
	ClientSecret string `mapstructure:"clientSecret"`
	RetryCount   int    `mapstructure:"retryCount"`
	Timeout      int    `mapstructure:"timeout"`
}

// InfoFlowConfig 如流API配置
type InfoFlowConfig struct {
	URL               string                           `mapstructure:"url"`
	AppKey            string                           `mapstructure:"appKey"`
	AppSecret         string                           `mapstructure:"appSecret"`
	RetryCount        int                              `mapstructure:"retryCount"`
	Timeout           int                              `mapstructure:"timeout"`
	Cleanup           *InfoFlowCleanupConfig           `mapstructure:"cleanup"`
	GroupCacheRefresh *InfoFlowGroupCacheRefreshConfig `mapstructure:"groupCacheRefresh"`
}

// InfoFlowCleanupConfig 如流消息清理配置
type InfoFlowCleanupConfig struct {
	Enabled           bool   `mapstructure:"enabled"`
	Cron              string `mapstructure:"cron"`
	RetentionDays     int    `mapstructure:"retention_days"`
	LockExpireSeconds int    `mapstructure:"lock_expire_seconds"`
}

// InfoFlowGroupCacheRefreshConfig 群成员列表缓存刷新配置
type InfoFlowGroupCacheRefreshConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Cron    string `mapstructure:"cron"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level"`       // debug, info, warn, error
	FilePath   string `mapstructure:"file_path"`   // 日志文件路径
	MaxSize    int    `mapstructure:"max_size"`    // 单个文件最大大小(MB)
	MaxBackups int    `mapstructure:"max_backups"` // 保留旧文件的最大个数
	MaxAge     int    `mapstructure:"max_age"`     // 保留旧文件的最大天数
	Compress   bool   `mapstructure:"compress"`    // 是否压缩
}

// AgentWebhookConfig Agent Webhook配置
type AgentWebhookConfig struct {
	AgentCode  string            `mapstructure:"agentCode"`
	WebhookURL string            `mapstructure:"webhookUrl"`
	Headers    map[string]string `mapstructure:"headers"`
	Timeout    int               `mapstructure:"timeout"`    // 超时时间（秒）
	RetryCount int               `mapstructure:"retryCount"` // 重试次数
	RetryDelay int               `mapstructure:"retryDelay"` // 重试间隔（秒）
	Enabled    bool              `mapstructure:"enabled"`
}

// ContextMapConfig context-map 服务配置
type ContextMapConfig struct {
	BaseURL       string `mapstructure:"baseURL"`       // context-map 服务地址
	PlatformToken string `mapstructure:"platformToken"` // 平台 token（环境变量 Key）
	Timeout       int    `mapstructure:"timeout"`       // 超时时间（秒），默认 30
	RetryCount    int    `mapstructure:"retryCount"`    // 最大重试次数（不含首次），默认 3；0 表示不重试
	RetryDelayMs  int    `mapstructure:"retryDelayMs"`  // 重试初始间隔（毫秒），默认 200，使用指数退避
}

var globalConfig *Config

// LoadConfig 加载配置文件
func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析配置
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	globalConfig = &config
	return &config, nil
}

// GetConfig 获取全局配置
func GetConfig() *Config {
	return globalConfig
}

// GetDBConnection 从环境变量获取数据库连接字符串
func (c *Config) GetDBConnection() string {
	return os.Getenv(c.DBCfg.Connection)
}

// GetBOSAK 从环境变量获取BOS AccessKey
func (c *Config) GetBOSAK() string {
	return os.Getenv(c.BOSConfig.AK)
}

// GetBOSSK 从环境变量获取BOS SecretKey
func (c *Config) GetBOSSK() string {
	return os.Getenv(c.BOSConfig.SK)
}

// GetAuthenticationConfig 获取认证配置
func GetAuthenticationConfig() (*AuthenticationConfig, error) {
	sc := GetConfig()
	if sc.AuthenticationConfig == nil {
		return nil, fmt.Errorf("authenticationConfig is nil")
	}
	if len(os.Getenv(sc.AuthenticationConfig.ClientId)) > 0 && len(os.Getenv(sc.AuthenticationConfig.ClientSecret)) > 0 {
		sc.AuthenticationConfig.ClientId = os.Getenv(sc.AuthenticationConfig.ClientId)
		sc.AuthenticationConfig.ClientSecret = os.Getenv(sc.AuthenticationConfig.ClientSecret)
	}
	return sc.AuthenticationConfig, nil
}

// GetUUAPConfig 获取UUAP配置
func GetUUAPConfig() (*UUAPConfig, error) {
	sc := GetConfig()
	if sc.UUAPConfig == nil {
		return nil, fmt.Errorf("uuapConfig is nil")
	}
	if len(os.Getenv(sc.UUAPConfig.ClientId)) > 0 && len(os.Getenv(sc.UUAPConfig.ClientSecret)) > 0 {
		sc.UUAPConfig.ClientId = os.Getenv(sc.UUAPConfig.ClientId)
		sc.UUAPConfig.ClientSecret = os.Getenv(sc.UUAPConfig.ClientSecret)
	}
	return sc.UUAPConfig, nil
}

// GetInfoFlowConfig 获取如流配置
func GetInfoFlowConfig() (*InfoFlowConfig, error) {
	sc := GetConfig()
	if sc.InfoFlowConfig == nil {
		return nil, fmt.Errorf("infoflowConfig is nil")
	}
	// 从环境变量获取敏感配置
	if envSecret := os.Getenv(sc.InfoFlowConfig.AppSecret); len(envSecret) > 0 {
		sc.InfoFlowConfig.AppSecret = envSecret
	}
	if envAppkey := os.Getenv(sc.InfoFlowConfig.AppKey); len(envAppkey) > 0 {
		sc.InfoFlowConfig.AppKey = os.Getenv(sc.InfoFlowConfig.AppKey)
	}
	return sc.InfoFlowConfig, nil
}

// GetContextMapConfig 获取 context-map 配置
func GetContextMapConfig() (*ContextMapConfig, error) {
	sc := GetConfig()
	if sc.ContextMapConfig == nil {
		return nil, fmt.Errorf("contextMapConfig is nil")
	}
	// 从环境变量获取平台 token
	if envToken := os.Getenv(sc.ContextMapConfig.PlatformToken); len(envToken) > 0 {
		sc.ContextMapConfig.PlatformToken = envToken
	}
	return sc.ContextMapConfig, nil
}
