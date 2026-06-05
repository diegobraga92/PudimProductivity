package shared

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

const redisPingTimeout = 3 * time.Second

type Cache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewCache(redisURL string, ttl time.Duration) *Cache {
	// If redisURL is empty, returns a no-op cache that always misses.
	if redisURL == "" {
		log.Warn().Msg("Redis URL not configured — caching disabled")
		return &Cache{client: nil, ttl: ttl}
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Warn().Err(err).Msg("invalid Redis URL — caching disabled")
		return &Cache{client: nil, ttl: ttl}
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), redisPingTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Warn().Err(err).Msg("Redis connection failed — caching disabled")
		return &Cache{client: nil, ttl: ttl}
	}

	log.Info().Str("addr", opts.Addr).Msg("Redis cache connected")
	return &Cache{client: client, ttl: ttl}
}

func (c *Cache) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	if c.client == nil {
		return false, nil // cache miss (no Redis)
	}

	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return false, nil // cache miss
		}
		log.Warn().Err(err).Str("key", key).Msg("Redis GET error")
		return false, nil // fail open — treat as miss
	}

	if err := json.Unmarshal(data, dest); err != nil {
		log.Warn().Err(err).Str("key", key).Msg("Redis cache unmarshal error")
		return false, nil
	}

	return true, nil
}

func (c *Cache) Set(ctx context.Context, key string, value interface{}) error {
	if c.client == nil {
		return nil // no-op
	}

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache marshal error: %w", err)
	}

	if err := c.client.Set(ctx, key, data, c.ttl).Err(); err != nil {
		log.Warn().Err(err).Str("key", key).Msg("Redis SET error")
		return err
	}

	return nil
}

func (c *Cache) Del(ctx context.Context, keys ...string) error {
	if c.client == nil {
		return nil // no-op
	}

	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		log.Warn().Strs("keys", keys).Err(err).Msg("Redis DEL error")
		return err
	}

	return nil
}

func (c *Cache) Close() error {
	if c.client == nil {
		return nil
	}
	return c.client.Close()
}

// CacheKey helpers for consistent key naming.
type CacheKey string

const (
	CacheKeyTask      CacheKey = "task:%s"       // task:{id}
	CacheKeyTaskList  CacheKey = "tasks:list"    // tasks:list (user-specific keys to be added with auth)
)

func Key(format CacheKey, args ...interface{}) string {
	return fmt.Sprintf(string(format), args...)
}