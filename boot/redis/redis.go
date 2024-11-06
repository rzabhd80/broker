package redis

import (
	"context"
	"github.com/go-redis/redis/v8"
)

func InitRedis(address string, port string, password string) (*redis.Client, error) {
	var poolSize int = 10000
	redisClient := redis.NewClient(&redis.Options{
		Addr:     address + ":" + port,
		Password: password,
		PoolSize: poolSize,
		DB:       0,
	})

	_, err := redisClient.Ping(context.Background()).Result()
	if err != nil {
		return &redis.Client{}, err
	}
	return redisClient, nil
}
