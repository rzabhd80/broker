package interfaces

import (
	"broker/internals/models"
	pb "broker/internals/protos/generated/broker/api/proto"
	"context"
	"github.com/go-redis/redis/v8"
)

type MessageStore interface {
	StoreMessage(message models.Message, subject string) (int, error)
	FetchMessage(messageId int, subject string) (*models.Message, error)
}

type BrokerServices interface {
	GetRedisClient() *redis.Client
}

type Publisher interface {
	Publish(client pb.BrokerClient, ctx context.Context) error
}

type BrokerConnector interface {
	Connect() error
	GetClient() pb.BrokerClient
	GetCurrentNode() string
}
