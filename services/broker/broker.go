package broker

import (
	redisService "broker/boot/redis"
	"broker/env"
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
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
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
	RedisClient   *redis.Client
	dbms          dbms.Dbms
	isClosed      bool
	mutex         sync.Mutex
}

func SetupBrokerServer() (*BrokerServer, error) {
	if ServerInstance != nil {
		return nil, errors.New("Broker Server Already Instantiated")
	}
	broker := &BrokerServer{
		EnvConfig: &env.DevelopmentBrokerConfig{},
		fsm:       FsmMachine{Messages: make(map[string][]models.Message)},
		mutex:     sync.Mutex{},
	}

	raftInstance, snapshot, err := broker.SetupRaft()
	if err != nil {
		return nil, err
	}

	broker.Raft = raftInstance
	broker.SnapShotStore = snapshot
	broker.RedisClient, err = redisService.InitRedis(broker.EnvConfig.RedisHost, broker.EnvConfig.RedisPort,
		broker.EnvConfig.RedisPassword)
	broker.dbms, err = dbms.InitRAM(broker)
	ServerInstance = broker
	return broker, nil
}

func (broker *BrokerServer) Publish(ctx context.Context, request *proto.PublishRequest) (*proto.PublishResponse, error) {
	if broker.Raft.State() != raft.Leader {
		// If not the leader, set the leader's address in the gRPC metadata
		leader, _ := broker.Raft.LeaderWithID()
		leaderAddr := string(leader)
		md := metadata.Pairs("leader-addr", leaderAddr)
		err := grpc.SetHeader(ctx, md)
		if err != nil {
			return nil, err
		}

		return nil, status.Errorf(codes.FailedPrecondition, "not the leader, please connect to %s", leaderAddr)
	}
	msg := models.Message{
		Id:         0,
		Body:       string(request.Body),
		Expiration: time.Duration(request.ExpirationSeconds),
	}
	var errorFound bool = false
	messageId, err := broker.dbms.StoreMessage(msg, request.Subject)
	if err != nil {
		errorFound = true
	}
	msg.Id = messageId
	msgJSON, err := json.Marshal(msg)
	if err != nil {
		errorFound = true
	}
	err = broker.RedisClient.Publish(ctx, request.Subject, msgJSON).Err()
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
	if broker.isClosed {
		return errors.New("broker Is Down")
	}
	broker.mutex.Lock()
	defer broker.mutex.Unlock()

	ch := make(chan models.Message, 100)
	channelContext := context.Background()
	pubsub := broker.RedisClient.Subscribe(channelContext, request.Subject)
	_, err := pubsub.Receive(channelContext)
	if err != nil {
		return err
	}

	go func() {
		defer func(pubsub *redis.PubSub) {
			err := pubsub.Close()
			if err != nil {
				logrus.Error("ERROR: Connection With Sub Interrupted")
			}
		}(pubsub)

		for {
			msg, err := pubsub.ReceiveMessage(channelContext)
			if err != nil {
				close(ch)
				return
			}

			var message models.Message
			err = json.Unmarshal([]byte(msg.Payload), &message)
			if err != nil {
				continue
			}

			select {
			case ch <- message:
			default:
			}
		}
	}()
	for {
		select {
		case msg := <-ch:
			protoMsg := &proto.MessageResponse{Body: []byte(msg.Body)}

			if err := subscribeServer.Send(protoMsg); err != nil {
				logrus.Error("ERROR: Could Not Send Message To Subscriber")
				return err
			}
		case <-subscribeServer.Context().Done():
			logrus.Println("Subscriber Left")
			return err
		}
	}
}

func (broker *BrokerServer) Fetch(ctx context.Context, request *proto.FetchRequest) (*proto.MessageResponse, error) {
	if broker.isClosed {
		return nil, errors.New("Broker Channel Closed")
	}

	msg, err := broker.dbms.FetchMessage(int(request.Id), request.Subject)
	if err != nil {
		return nil, errors.New("Could Not Fetch Message")
	}

	return &proto.MessageResponse{Body: []byte(msg.Body)}, nil
}
