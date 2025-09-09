package redis

import (
	"context"

	"github.com/go-redis/redis/v8"
	"logvault/config"
)

// InitRedis initializes and returns a Redis client
func InitRedis(cfg config.Config) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Address,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	_, err := rdb.Ping(context.Background()).Result()
	return rdb, err
}
