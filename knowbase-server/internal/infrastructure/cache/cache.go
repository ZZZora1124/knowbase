package cache

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-redis/redis/v8"

	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/config"
)

var cache Cache

type Cache interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Exists(ctx context.Context, key string) (bool, error)
	Delete(ctx context.Context, key string) error
	DeleteByPattern(ctx context.Context, pattern string) error
	SAdd(ctx context.Context, key, value string) error
	SRem(ctx context.Context, key, value string) error
	SMembers(ctx context.Context, key string) ([]string, error)
	ZAdd(ctx context.Context, key string, member string, score float64) error
	ZRem(ctx context.Context, key, member string) error
	ZRangeByScore(ctx context.Context, key string, min, max float64) ([]string, error)
	ZRange(ctx context.Context, key string, start, end int64) ([]string, error)
	HSet(ctx context.Context, key, field string, value interface{}) error
	HGet(ctx context.Context, key, field string) (string, error)
	HDel(ctx context.Context, key string, field ...string) error
	HKeys(ctx context.Context, key string) ([]string, error)
}

func NewCache(cfg *config.CacheConfig) error {
	if cfg == nil {
		return fmt.Errorf("cache config is nil")
	}
	// 初始化redis缓存
	if cfg.Redis != nil {
		password := os.Getenv(cfg.Redis.Password)
		if password != "" {
			cfg.Redis.Password = password
		}
		rdb := redis.NewClient(&redis.Options{
			Addr:     cfg.Redis.Addr,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})

		_, err := rdb.Ping(context.Background()).Result()
		if err != nil {
			return fmt.Errorf("init redis client error: %w", err)
		}

		cache = &RedisCache{
			Client:  rdb,
			Prefix:  "knowledge:server:",
			Enabled: true,
		}

		return nil
	}

	return fmt.Errorf("invalid cache config")
}

func GetCache() Cache {
	return cache
}
