package brokerClient

import (
	pb "broker/internals/protos/generated/broker/api/proto"
	"broker/services/publisher"
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"sync"
)

type BrokerClient struct {
	mu          sync.RWMutex
	conn        *grpc.ClientConn
	KnownNodes  []string // List of all node addresses
	CurrentNode string
}

func NewBrokerClient(nodes []string) *BrokerClient {
	return &BrokerClient{
		KnownNodes: nodes,
	}
}

func (c *BrokerClient) Connect() error {
	for _, node := range c.KnownNodes {
		conn, err := grpc.NewClient(node, grpc.WithTransportCredentials(insecure.NewCredentials()))

		client := pb.NewBrokerClient(conn)
		errFound := publisher.Publish(client, context.Background())
		
		if err == nil && errFound == nil {
			c.mu.Lock()
			c.conn = conn
			c.CurrentNode = node
			c.mu.Unlock()
			return nil
		}
	}
	return fmt.Errorf("failed to connect to any node")
}
