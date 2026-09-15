package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"strings"
	"time"

	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/log"
)

type RedisCache struct {
	Prefix  string
	Client  *redis.Client
	Enabled bool // 缓存开关，false 时所有操作直接跳过
}

func (b *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	// 缓存开关关闭,直接跳过
	if !b.Enabled {
		log.Logger.Debugf("[Set] cache disabled, skip setting key:%v", key)
		return nil
	}

	now := time.Now().Nanosecond()
	defer func(begin int) {
		log.Logger.Debugf("[Set] redis request cost: %dms, key:%v", (time.Now().Nanosecond()-begin)/1000000, key)
	}(now)

	if "" == key || value == nil {
		return fmt.Errorf("invalid input, key: %v, value: %v", key, value)
	}

	var data string
	str, ok := value.(string)
	if !ok {
		bytes, err := json.Marshal(value)
		if err != nil {
			return err
		}
		data = string(bytes)
	} else {
		data = str
	}

	// 统一加前缀
	key = b.Prefix + key

	log.Logger.Debugf("set to redis start, key:%v, value:%v", key, data)
	if err := b.Client.Set(ctx, key, data, expiration).Err(); err != nil {
		return fmt.Errorf("set to redis error:%w", err)
	}
	log.Logger.Debugf("set to redis successfully, key:%v", key)

	return nil
}

func (b *RedisCache) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	now := time.Now().Nanosecond()
	defer func(begin int) {
		log.Logger.Debugf("[SetNX] redis request cost: %dms, key:%v", (time.Now().Nanosecond()-begin)/1000000, key)
	}(now)

	if "" == key || value == nil {
		return fmt.Errorf("invalid input, key: %v, value: %v", key, value)
	}

	var data string
	str, ok := value.(string)
	if !ok {
		bytes, err := json.Marshal(value)
		if err != nil {
			return err
		}
		data = string(bytes)
	} else {
		data = str
	}

	// 统一加前缀
	key = b.Prefix + key

	log.Logger.Debugf("setnx to redis start, key:%v, value:%v", key, data)
	isSet, err := b.Client.SetNX(ctx, key, data, expiration).Result()
	if err != nil {
		return fmt.Errorf("setnx to redis error:%w", err)
	}
	if isSet {
		log.Logger.Debugf("setnx to redis successfully, key:%v", key)
		return nil
	}

	return fmt.Errorf("setnx to redis failed, already exist")
}

func (b *RedisCache) Get(ctx context.Context, key string) (string, error) {
	// 缓存开关关闭,返回缓存未命中
	if !b.Enabled {
		log.Logger.Debugf("[Get] cache disabled, return cache miss for key:%v", key)
		return "", redis.Nil
	}

	now := time.Now().Nanosecond()
	defer func(begin int) {
		log.Logger.Debugf("[Get] redis request cost: %dms, key:%v", (time.Now().Nanosecond()-begin)/1000000, key)
	}(now)

	if "" == key {
		return "", fmt.Errorf("key is empty")
	}

	if !strings.Contains(key, b.Prefix) {
		// 统一加前缀
		key = b.Prefix + key
	}

	log.Logger.Debugf("get from redis start, key:%v", key)
	value, err := b.Client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", fmt.Errorf("redis key [%v] not found, error:%w", key, err)
		}
		return "", fmt.Errorf("redis key [%v] get error:%w", key, err)
	}
	log.Logger.Debugf("get from redis successfully, key:%v, vaelue:%v", key, value)
	return value, nil
}

func (b *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	now := time.Now().Nanosecond()
	defer func(begin int) {
		log.Logger.Debugf("[Exists] redis request cost: %dms, key:%v", (time.Now().Nanosecond()-begin)/1000000, key)
	}(now)

	if "" == key {
		return false, fmt.Errorf("key is empty")
	}

	if !strings.Contains(key, b.Prefix) {
		// 统一加前缀
		key = b.Prefix + key
	}

	exist, err := b.Client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("redis key [%v] exists error:%w", key, err)
	}
	return exist > 0, nil
}

func (b *RedisCache) Delete(ctx context.Context, key string) error {
	// 缓存开关关闭,直接跳过
	if !b.Enabled {
		log.Logger.Debugf("[Delete] cache disabled, skip deleting key:%v", key)
		return nil
	}

	now := time.Now().Nanosecond()
	defer func(begin int) {
		log.Logger.Debugf("[Delete] redis request cost: %dms, key:%v", (time.Now().Nanosecond()-begin)/1000000, key)
	}(now)

	if "" == key {
		return fmt.Errorf("key is empty")
	}

	if !strings.Contains(key, b.Prefix) {
		// 统一加前缀
		key = b.Prefix + key
	}

	err := b.Client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("redis key [%v] delete error:%w", key, err)
	}
	return nil
}

func (b *RedisCache) SAdd(ctx context.Context, key, value string) error {
	now := time.Now().Nanosecond()
	defer func(begin int) {
		log.Logger.Debugf("[SAdd] redis request cost: %dms, key:%v, value:%v", (time.Now().Nanosecond()-begin)/1000000, key, value)
	}(now)

	if "" == key || "" == value {
		return fmt.Errorf("invalid input, key: %v, value: %v", key, value)
	}

	// 统一加前缀
	key = b.Prefix + key

	log.Logger.Debugf("sadd to redis start, key:%v, value:%v", key, value)
	if err := b.Client.SAdd(ctx, key, value).Err(); err != nil {
		return fmt.Errorf("sadd to redis error:%w", err)
	}
	log.Logger.Debugf("sadd to redis successfully, key:%v, value:%v", key, value)

	return nil
}

func (b *RedisCache) SRem(ctx context.Context, key, value string) error {
	now := time.Now().Nanosecond()
	defer func(begin int) {
		log.Logger.Debugf("[SRem] redis request cost: %dms, key:%v, value:%v", (time.Now().Nanosecond()-begin)/1000000, key, value)
	}(now)

	if "" == key || "" == value {
		return fmt.Errorf("invalid input, key: %v, value: %v", key, value)
	}

	// 统一加前缀
	key = b.Prefix + key

	log.Logger.Debugf("srem from redis start, key:%v, value:%v", key, value)
	if err := b.Client.SRem(ctx, key, value).Err(); err != nil {
		return fmt.Errorf("srem from redis error:%w", err)
	}
	log.Logger.Debugf("srem from redis successfully, key:%v, value:%v", key, value)

	return nil
}

func (b *RedisCache) SMembers(ctx context.Context, key string) ([]string, error) {
	now := time.Now().Nanosecond()
	defer func(begin int) {
		log.Logger.Debugf("[SMembers] redis request cost: %dms, key:%v", (time.Now().Nanosecond()-begin)/1000000, key)
	}(now)

	if "" == key {
		return nil, fmt.Errorf("invalid input, key: %v", key)
	}

	// 统一加前缀
	key = b.Prefix + key

	log.Logger.Debugf("smembers from redis start, key:%v", key)
	res, err := b.Client.SMembers(ctx, key).Result()
	if err != nil {
		return res, fmt.Errorf("smembers from redis error:%w", err)
	}
	log.Logger.Debugf("smembers from redis successfully, key:%v", key)

	return res, nil
}

func (b *RedisCache) ZAdd(ctx context.Context, key string, member string, score float64) error {
	now := time.Now().Nanosecond()
	defer func(begin int) {
		log.Logger.Debugf("[ZAdd] redis request cost: %dms, key:%v, member:%v, score:%f", (time.Now().Nanosecond()-begin)/1000000, key, member, score)
	}(now)

	if "" == key || "" == member {
		return fmt.Errorf("invalid input, key: %v, member: %v", key, member)
	}

	// 统一加前缀
	key = b.Prefix + key

	log.Logger.Debugf("zadd to redis start, key:%v, member:%v, score:%f", key, member, score)
	if err := b.Client.ZAdd(ctx, key, &redis.Z{
		Score:  score,
		Member: member,
	}).Err(); err != nil {
		return fmt.Errorf("zadd to redis error:%w", err)
	}
	log.Logger.Debugf("zadd to redis successfully, key:%v, member:%v, score:%f", key, member, score)

	return nil
}

func (b *RedisCache) ZRem(ctx context.Context, key string, member string) error {
	now := time.Now().Nanosecond()
	defer func(begin int) {
		log.Logger.Debugf("[ZRem] redis request cost: %dms, key:%v, member:%v", (time.Now().Nanosecond()-begin)/1000000, key, member)
	}(now)

	if "" == key || 0 == len(member) {
		return fmt.Errorf("invalid input, key: %v, member: %v", key, member)
	}

	// 统一加前缀
	key = b.Prefix + key

	log.Logger.Debugf("zrem from redis start, key:%v, member:%v", key, member)
	if err := b.Client.ZRem(ctx, key, member).Err(); err != nil {
		return fmt.Errorf("zrem from redis error:%w", err)
	}
	log.Logger.Debugf("zrem from redis successfully, key:%v, member:%v", key, member)

	return nil
}

func (b *RedisCache) ZRangeByScore(ctx context.Context, key string, min, max float64) ([]string, error) {
	now := time.Now().Nanosecond()
	defer func(begin int) {
		log.Logger.Debugf("[ZRangeByScore] redis request cost: %dms, key:%v, min:%f, max:%f",
			(time.Now().Nanosecond()-begin)/1000000, key, min, max)
	}(now)

	if "" == key {
		return nil, fmt.Errorf("invalid input, key: %v", key)
	}

	if min >= max {
		return nil, fmt.Errorf("invalid input, key: %v, min: %f, max: %f", key, min, max)
	}

	// 统一加前缀
	key = b.Prefix + key

	log.Logger.Debugf("zrangebyscore from redis start, key:%v, min:%f, max:%f", key, min, max)
	res, err := b.Client.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: fmt.Sprintf("%f", min),
		Max: fmt.Sprintf("%f", max),
	}).Result()
	if err != nil {
		return res, fmt.Errorf("zrangebyscore from redis error:%w", err)
	}
	log.Logger.Debugf("zrangebyscore from redis successfully, key:%v, min:%f, max:%f", key, min, max)

	return res, nil
}

func (b *RedisCache) ZRange(ctx context.Context, key string, start, end int64) ([]string, error) {
	now := time.Now().Nanosecond()
	defer func(begin int) {
		log.Logger.Debugf("[ZRange] redis request cost: %dms, key:%v, start:%d, end:%d",
			(time.Now().Nanosecond()-begin)/1000000, key, start, end)
	}(now)

	if "" == key {
		return nil, fmt.Errorf("invalid input, key: %v", key)
	}

	// 统一加前缀
	key = b.Prefix + key

	log.Logger.Debugf("zrange from redis start, key:%v, start:%d, end:%d", key, start, end)
	res, err := b.Client.ZRange(ctx, key, start, end).Result()
	if err != nil {
		return res, fmt.Errorf("zrange from redis error:%w", err)
	}
	log.Logger.Debugf("zrange from redis successfully, key:%v, start:%d, end:%d", key, start, end)

	return res, nil
}

func (b *RedisCache) HSet(ctx context.Context, key, field string, value interface{}) error {
	now := time.Now().Nanosecond()
	defer func(begin int) {
		log.Logger.Debugf("[HSet] redis request cost: %dms, key:%v", (time.Now().Nanosecond()-begin)/1000000, key)
	}(now)

	if "" == key || "" == field || value == nil {
		return fmt.Errorf("invalid input, key: %v, filed: %v, value: %v", key, field, value)
	}

	var data string
	str, ok := value.(string)
	if !ok {
		bytes, err := json.Marshal(value)
		if err != nil {
			return err
		}
		data = string(bytes)
	} else {
		data = str
	}

	// 统一加前缀
	key = b.Prefix + key

	log.Logger.Debugf("hset to redis start, key:%v, field:%v, value:%v", key, field, data)
	if err := b.Client.HSet(ctx, key, field, data).Err(); err != nil {
		return fmt.Errorf("hset to redis error:%w", err)
	}
	log.Logger.Debugf("hset to redis successfully, key:%v, field:%v", key, field)

	return nil
}

func (b *RedisCache) HGet(ctx context.Context, key, field string) (string, error) {
	now := time.Now().Nanosecond()
	defer func(begin int) {
		log.Logger.Debugf("[HGet] redis request cost: %dms, key:%v, field:%v", (time.Now().Nanosecond()-begin)/1000000, key, field)
	}(now)

	if "" == key || "" == field {
		return "", fmt.Errorf("invalid input, key: %v, field: %v", key, field)
	}

	// 统一加前缀
	key = b.Prefix + key

	log.Logger.Debugf("hget from redis start, key:%v, field:%v", key, field)
	res, err := b.Client.HGet(ctx, key, field).Result()
	if err != nil {
		return "", fmt.Errorf("hget from redis error:%w", err)
	}
	log.Logger.Debugf("hget from redis successfully, key:%v, field:%v", key, field)

	return res, nil
}

func (b *RedisCache) HDel(ctx context.Context, key string, field ...string) error {
	now := time.Now().Nanosecond()
	defer func(begin int) {
		log.Logger.Debugf("[HDel] redis request cost: %dms, key:%v, field:%v", (time.Now().Nanosecond()-begin)/1000000, key, field)
	}(now)

	if "" == key || 0 == len(field) {
		return fmt.Errorf("invalid input, key: %v, field: %v", key, field)
	}

	// 统一加前缀
	key = b.Prefix + key

	log.Logger.Debugf("hdel from redis start, key:%v, field:%v", key, field)
	err := b.Client.HDel(ctx, key, field...).Err()
	if err != nil {
		return fmt.Errorf("hdel from redis error:%w", err)
	}
	log.Logger.Debugf("hdel from redis successfully, key:%v, field:%v", key, field)

	return nil
}

func (b *RedisCache) HKeys(ctx context.Context, key string) ([]string, error) {
	now := time.Now().Nanosecond()
	defer func(begin int) {
		log.Logger.Debugf("[HKeys] redis request cost: %dms, key:%v", (time.Now().Nanosecond()-begin)/1000000, key)
	}(now)

	if "" == key {
		return nil, fmt.Errorf("invalid input, key: %v", key)
	}

	// 统一加前缀
	key = b.Prefix + key

	log.Logger.Debugf("hkeys from redis start, key:%v", key)
	res, err := b.Client.HKeys(ctx, key).Result()
	if err != nil {
		return res, fmt.Errorf("hkeys from redis error:%w", err)
	}
	log.Logger.Debugf("hkeys from redis successfully, key:%v", key)

	return res, nil
}

// DeleteByPattern 根据模式批量删除缓存
// pattern: Redis key 模式,如 "visible_models:*"
func (b *RedisCache) DeleteByPattern(ctx context.Context, pattern string) error {
	// 缓存开关关闭,直接跳过
	if !b.Enabled {
		log.Logger.Debugf("[DeleteByPattern] cache disabled, skip deleting pattern:%v", pattern)
		return nil
	}

	now := time.Now().Nanosecond()
	defer func(begin int) {
		log.Logger.Debugf("[DeleteByPattern] redis request cost: %dms, pattern:%v",
			(time.Now().Nanosecond()-begin)/1000000, pattern)
	}(now)

	if "" == pattern {
		return fmt.Errorf("pattern is empty")
	}

	// 统一加前缀
	fullPattern := b.Prefix + pattern

	log.Logger.Debugf("delete by pattern from redis start, pattern:%v", fullPattern)

	// 使用 SCAN 命令遍历匹配的 key (生产环境推荐,避免阻塞)
	// COUNT=100 减少网络 RTT 次数，提升批量删除性能
	const scanCount = 100
	iter := b.Client.Scan(ctx, 0, fullPattern, scanCount).Iterator()
	deleteCount := 0
	for iter.Next(ctx) {
		if err := b.Client.Del(ctx, iter.Val()).Err(); err != nil {
			log.Logger.Errorf("Failed to delete cache key %s: %v", iter.Val(), err)
		} else {
			deleteCount++
		}
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("scan pattern error:%w", err)
	}

	log.Logger.Debugf("delete by pattern successfully, pattern:%v, deleted count:%d", fullPattern, deleteCount)
	return nil
}

