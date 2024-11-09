package dbms

import (
	"broker/interfaces"
	"broker/internals/models"
	"context"
	"encoding/json"
	"errors"
	"github.com/go-redis/redis/v8"
	"log"
	"time"
)

type Memory struct {
	Dbms
	client *redis.Client
}

func InitRAM(server interfaces.BrokerServices) (*Memory, error) {
	redisClient := server.GetRedisClient()
	log.Printf("redis client in db %s", redisClient)
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
