package brokerClient

import (
	"broker/helpers"
	pb "broker/internals/protos/generated/broker/api/proto"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
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
		log.Printf("Trying to connect to node: %s", node)
		conn, err := grpc.NewClient(node, grpc.WithTransportCredentials(insecure.NewCredentials()))
		client := pb.NewBrokerClient(conn)
		connectionCheck := helpers.NewConnectionChecker()
		errFound := connectionCheck.CheckConnection(client)
		log.Printf("%s", errFound)
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

func (c *BrokerClient) GetClient() pb.BrokerClient {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.GetClient()
}

func (c *BrokerClient) GetCurrentNode() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.CurrentNode
}
