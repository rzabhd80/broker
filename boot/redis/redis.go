package redis

import (
	"broker/internals/models"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"log"
	"os"
	"time"
)

func InitRedis() (*redis.Client, error) {
	fmt.Printf("connecting to redis")
	var poolSize int = 10000
	address := os.Getenv("REDIS_HOST")
	port := os.Getenv("REDIS_PORT")
	redisClient := redis.NewClient(&redis.Options{
		Addr:     address + ":" + port,
		PoolSize: poolSize,
		DB:       0,
	})
	log.Printf("redis client %s", redisClient)
	status, err := redisClient.Ping(context.Background()).Result()
	if err != nil {
		log.Printf("status is %s", status)
		fmt.Printf("connecting to redis failed %s", err)
		return &redis.Client{}, err
	}
	fmt.Printf("connecting to redis successful")
	return redisClient, nil
}

func StoreMessage(redisClient redis.Client, message models.Message, subject string) (int, error) {
	item := models.MessageStorage{
		Body:       message.Body,
		Expiration: time.Now().Add(message.Expiration),
	}

	itemJSON, err := json.Marshal(item)
	if err != nil {
		return 0, err
	}

	_, err = redisClient.LPush(context.Background(), subject, itemJSON).Result()
	if err != nil {
		return 0, err
	}

	length, err := redisClient.LLen(context.Background(), subject).Result()
	if err != nil {
		return 0, err
	}

	id := int(length) - 1

	return id, nil
}

func FetchMessage(redisClient redis.Client, messageId int, subject string) (*models.Message, error) {
	reply, err := redisClient.LIndex(context.Background(), subject, int64(messageId)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, errors.New("message with id provided is not valid or never published")
	} else if err != nil {
		return nil, err
	}

	var item models.MessageStorage
	if err := json.Unmarshal([]byte(reply), &item); err != nil {
		return nil, err
	}

	if time.Now().After(item.Expiration) {
		return nil, errors.New("message with id provided is expired")
	}

	msg := models.Message{
		Id:         messageId,
		Body:       item.Body,
		Expiration: 0,
	}

	return &msg, nil
}
