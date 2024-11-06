package dbms

import (
	"broker/internals/models"
	"broker/services/broker"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"time"
)

type Memory struct {
	Dbms
	client *redis.Client
}

func InitRAM(server *broker.BrokerServer) (*Memory, error) {
	redisClient := server.RedisClient

	ctx := context.Background()
	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		fmt.Println("Error:", err)
		return nil, err
	}

	return &Memory{
		client: redisClient,
	}, nil
}

func (redisClient *Memory) StoreMessage(message models.Message, subject string) (int, error) {
	item := models.MessageStorage{
		Body:       message.Body,
		Expiration: time.Now().Add(message.Expiration),
	}

	itemJSON, err := json.Marshal(item)
	if err != nil {
		return 0, err
	}

	_, err = redisClient.client.LPush(context.Background(), subject, itemJSON).Result()
	if err != nil {
		return 0, err
	}

	length, err := redisClient.client.LLen(context.Background(), subject).Result()
	if err != nil {
		return 0, err
	}

	id := int(length) - 1

	return id, nil
}

func (redisClient *Memory) FetchMessage(messageId int, subject string) (*models.Message, error) {
	reply, err := redisClient.client.LIndex(context.Background(), subject, int64(messageId)).Result()
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
