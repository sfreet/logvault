package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisClient struct {
	client *redis.Client
}

func NewRedisClient(addr, password string, db int) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// Ping the Redis server to check the connection
	_, err := client.Ping(client.Context()).Result()
	if err != nil {
		return nil, err
	}

	return &RedisClient{client: client}, nil
}

func (r *RedisClient) Get(key string) (string, error) {
	return r.client.Get(r.client.Context(), key).Result()
}

func (r *RedisClient) Set(key string, value string, expiration time.Duration) error {
	return r.client.Set(r.client.Context(), key, value, expiration).Err()
}

func (r *RedisClient) Del(keys ...string) error {
	return r.client.Del(r.client.Context(), keys...).Err()
}

func (r *RedisClient) HGetAll(key string) (map[string]string, error) {
	return r.client.HGetAll(r.client.Context(), key).Result()
}

func (r *RedisClient) LPush(key string, values ...interface{}) error {
	return r.client.LPush(r.client.Context(), key, values...).Err()
}

func (r *RedisClient) LRange(key string, start, stop int64) ([]string, error) {
	return r.client.LRange(r.client.Context(), key, start, stop).Result()
}

func (r *RedisClient) GetAllKeys(ctx context.Context) ([]string, error) {
	keys, err := r.client.Keys(ctx, "*").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get all keys: %w", err)
	}
	return keys, nil
}

func (r *RedisClient) GetKeysByPattern(ctx context.Context, pattern string) ([]string, error) {
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get keys by pattern %s: %w", pattern, err)
	}
	return keys, nil
}
