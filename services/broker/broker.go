package broker

import (
	"broker/env"
	"broker/internals/protos/generated/broker/api/proto"
	"broker/internals/repositories"
	"context"
	"github.com/hashicorp/raft"
)

type BrokerServer struct {
	proto.UnimplementedBrokerServer
	BrokerRepository repository.BrokerRepository
	Raft             raft.Raft
	SnapShotStore    raft.SnapshotStore
	fsm              FsmMachine
	envConfig        *env.DevelopmentBrokerConfig
}

func (broker *BrokerServer) Publish(ctx context.Context, request *proto.PublishRequest) (*proto.PublishResponse, error) {
	return nil, nil

}

func (broker *BrokerServer) Subscribe(request *proto.SubscribeRequest,
	subscribeServer proto.Broker_SubscribeServer) error {
	return nil
}

func (broker *BrokerServer) Fetch(ctx context.Context, request *proto.FetchRequest) (*proto.MessageResponse, error) {
	return nil, nil
}
