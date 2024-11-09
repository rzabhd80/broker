package broker

import (
	redisService "broker/boot/redis"
	"broker/env"
	"broker/interfaces"
	"broker/internals/dbms"
	"broker/internals/models"
	"broker/internals/protos/generated/broker/api/proto"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/hashicorp/raft"
	"github.com/sirupsen/logrus"
	"log"
	"sync"
	"time"
)

var ServerInstance *BrokerServer

type BrokerServer struct {
	proto.UnimplementedBrokerServer
	Raft          *raft.Raft
	SnapShotStore raft.SnapshotStore
	fsm           FsmMachine
	EnvConfig     *env.DevelopmentBrokerConfig
	messageStore  interfaces.MessageStore
	RedisClient   *redis.Client
	dbms          *dbms.Memory
	isClosed      bool
	mutex         sync.RWMutex
}

func (broker *BrokerServer) GetRedisClient() *redis.Client {
	return broker.RedisClient
}

func SetupBrokerServer() (*BrokerServer, error) {
	if ServerInstance != nil {
		return nil, errors.New("Broker Server Already Instantiated")
	}
	broker := &BrokerServer{
		EnvConfig: &env.DevelopmentBrokerConfig{},
		fsm:       FsmMachine{Messages: make(map[string][]models.Message)},
		mutex:     sync.RWMutex{},
	}

	//raftInstance, snapshot, err := broker.SetupRaft()
	//if err != nil {
	//	logrus.Println("broker server couldnt be started")
	//	return nil, err
	//}
	//broker.messageStore, err = dbms.InitRAM(broker)
	//
	//if err != nil {
	//	logrus.Println("mem db server couldnt be started")
	//	panic("couldnt set up db")
	//}
	//broker.Raft = raftInstance
	//broker.SnapShotStore = snapshot
	redisClient, err := redisService.InitRedis()
	fmt.Printf("redis client %s", redisClient)
	broker.RedisClient = redisClient
	if err != nil {
		fmt.Printf("couldnt set yp redis client")
	}
	dbmsInstance, err := dbms.InitRAM(broker)
	broker.dbms = dbmsInstance
	log.Printf("redis ram storage initiated %s", broker.dbms)
	ServerInstance = broker
	return broker, nil
}

func (broker *BrokerServer) Publish(ctx context.Context, request *proto.PublishRequest) (*proto.PublishResponse, error) {
	log.Printf("publishing message")
	msg := models.Message{
		Id:         0,
		Body:       string(request.Body),
		Expiration: time.Duration(request.ExpirationSeconds),
	}
	messageId, err := redisService.StoreMessage(*broker.RedisClient, msg, request.Subject)
	fmt.Println("message saved")
	var errorFound bool = false
	if err != nil {
		errorFound = true
	}
	msg.Id = messageId
	msgJSON, err := json.Marshal(msg)
	if err != nil {
		errorFound = true
	}
	fmt.Printf("%sd", msg.Id)
	fmt.Printf("%s", msg.Id)
	err = broker.RedisClient.Publish(ctx, request.Subject, msgJSON).Err()
	fmt.Println("message published")
	if err != nil {
		errorFound = true
	}
	if errorFound {
		logrus.Println(fmt.Sprintf("ERROR Publishing Message With ID: %d into Subject: %s", msg.Id,
			request.Subject))
		return nil, err
	}
	logrus.Println(fmt.Sprintf("Published Message With ID: %d into Subject: %s", msg.Id, request.Subject))
	return &proto.PublishResponse{Id: int32(msg.Id)}, nil

}
func (broker *BrokerServer) Subscribe(request *proto.SubscribeRequest,
	subscribeServer proto.Broker_SubscribeServer) error {

	log.Printf("Subscription request received for subject: %s", request.Subject)

	if broker.isClosed {
		return fmt.Errorf("broker is down")
	}

	// Use read lock
	broker.mutex.RLock()
	defer broker.mutex.RUnlock()

	ctx := subscribeServer.Context()

	// Debug Redis connection
	status := broker.RedisClient.Ping(ctx)
	if err := status.Err(); err != nil {
		log.Printf("Redis connection check failed: %v", err)
		return fmt.Errorf("redis connection error: %w", err)
	}
	log.Printf("Redis connection verified")

	// Subscribe to Redis channel
	pubsub := broker.RedisClient.Subscribe(ctx, request.Subject)
	defer pubsub.Close()

	// Verify subscription
	if _, err := pubsub.Receive(ctx); err != nil {
		log.Printf("Redis subscription failed: %v", err)
		return fmt.Errorf("failed to subscribe: %w", err)
	}
	log.Printf("Redis subscription confirmed for subject: %s", request.Subject)

	// Get message channel
	msgChan := pubsub.Channel()
	log.Printf("Redis message channel established")

	// Send initial confirmation message (optional)
	confirmMsg := &proto.MessageResponse{
		Body: []byte("Subscription active"),
	}
	if err := subscribeServer.Send(confirmMsg); err != nil {
		log.Printf("Failed to send confirmation message: %v", err)
		return err
	}
	log.Printf("Sent confirmation message to subscriber")

	// Message processing loop
	for {
		select {
		case msg, ok := <-msgChan:
			if !ok {
				log.Printf("Redis message channel closed")
				return nil
			}
			log.Printf("Received message from Redis: %s", msg.Payload)

			protoMsg := &proto.MessageResponse{
				Body: []byte(msg.Payload),
			}

			if err := subscribeServer.Send(protoMsg); err != nil {
				log.Printf("Failed to send message to subscriber: %v", err)
				return fmt.Errorf("failed to send message: %w", err)
			}
			log.Printf("Successfully sent message to subscriber")

		case <-ctx.Done():
			log.Printf("Subscriber context cancelled")
			return nil
		}
	}
}
func (broker *BrokerServer) Fetch(ctx context.Context, request *proto.FetchRequest) (*proto.MessageResponse, error) {
	if broker.isClosed {
		return nil, errors.New("Broker Channel Closed")
	}

	msg, err := redisService.FetchMessage(*broker.RedisClient, int(request.Id), request.Subject)
	if err != nil {
		return nil, errors.New("Could Not Fetch Message")
	}

	return &proto.MessageResponse{Body: []byte(msg.Body)}, nil
}
