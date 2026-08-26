// Package cache provides a small Redis-backed read-through cache with
// namespace-scoped version invalidation.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const defaultTTL = 30 * time.Second

// Cache is a Redis-backed JSON cache. It is safe for concurrent use.
type Cache struct {
	client *redis.Client
	ttl    time.Duration
}

// New connects to Redis and verifies reachability with a ping. It returns an
// error when Redis is unavailable so the caller can disable caching
// gracefully. A non-positive ttl falls back to the default.
func New(ctx context.Context, redisURL string, ttl time.Duration) (*Cache, error) {
	if redisURL == "" {
		return nil, errors.New("cache: empty redis url")
	}
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(opts)
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}
	if ttl <= 0 {
		ttl = defaultTTL
	}
	return &Cache{client: client, ttl: ttl}, nil
}

// Get unmarshals the cached JSON value into dest. It returns (true, nil) on a
// hit and (false, nil) on a miss. A corrupt entry is reported as an error.
func (c *Cache) Get(ctx context.Context, key string, dest any) (bool, error) {
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return false, fmt.Errorf("cache: unmarshal %q: %w", key, err)
	}
	return true, nil
}

// Set marshals value and stores it under key with the cache TTL.
func (c *Cache) Set(ctx context.Context, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache: marshal %q: %w", key, err)
	}
	return c.client.Set(ctx, key, data, c.ttl).Err()
}

// Del removes one or more keys. Missing keys are not an error.
func (c *Cache) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return c.client.Del(ctx, keys...).Err()
}

// Version returns the current invalidation version for a namespace, or 0 when
// it has never been bumped.
func (c *Cache) Version(ctx context.Context, ns string) (int64, error) {
	v, err := c.client.Get(ctx, versionKey(ns)).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return v, err
}

// Bump increments the namespace version, invalidating every cached entry that
// was keyed under a previous version. It returns the new version.
func (c *Cache) Bump(ctx context.Context, ns string) (int64, error) {
	return c.client.Incr(ctx, versionKey(ns)).Result()
}

// Close releases the underlying Redis connection.
func (c *Cache) Close() error {
	return c.client.Close()
}

func versionKey(ns string) string {
	return "cache:version:" + ns
}
