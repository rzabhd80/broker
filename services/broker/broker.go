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
	"os"
	"sync"
	"time"
)

type InitConfig struct {
	RedisHost     string
	RedisPort     string
	NodeID        string
	RaftPort      string
	TransportPort string
	SnapshotPath  string
	NodeHost      string
	ClusterNodes  string
	IsInitiator   bool
}

// LoadConfigFromEnv loads configuration from environment variables
func LoadConfigFromEnv() (*InitConfig, error) {
	return &InitConfig{
		RedisHost:     os.Getenv("REDIS_HOST"),
		RedisPort:     os.Getenv("REDIS_PORT"),
		NodeID:        os.Getenv("NODE_ID"),
		RaftPort:      os.Getenv("RAFT_PORT"),
		TransportPort: os.Getenv("TRANSPORT_PORT"),
		SnapshotPath:  os.Getenv("SNAPSHOT_PATH"),
		NodeHost:      os.Getenv("NODE_HOST"),
		ClusterNodes:  os.Getenv("CLUSTER_NODES"),
		IsInitiator:   os.Getenv("INITIATOR") == "true",
	}, nil
}

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

var (
	ServerInstance *BrokerServer
	initOnce       sync.Once
	initErr        error
)

// GetBrokerServer provides safe access to the broker instance
func GetBrokerServer() (*BrokerServer, error) {
	if ServerInstance == nil {
		return nil, errors.New("broker server not initialized - call SetupBrokerServer first")
	}
	return ServerInstance, nil
}

// SetupBrokerServer now uses sync.Once to ensure single initialization
func SetupBrokerServer() (*BrokerServer, error) {
	initOnce.Do(func() {
		if ServerInstance != nil {
			initErr = errors.New("broker server already instantiated")
			return
		}

		broker := &BrokerServer{
			EnvConfig: &env.DevelopmentBrokerConfig{
				TransportPort: os.Getenv("TRANSPORT_PORT"),
				SnapShotPath:  os.Getenv("SNAPSHOT_PATH"),
			},
			fsm: FsmMachine{
				Messages: make(map[string][]models.Message),
				rwMutex:  sync.RWMutex{},
			},
			mutex: sync.RWMutex{},
		}

		// Setup Raft cluster
		raftInstance, snapshot, err := broker.SetupRaft()
		if err != nil {
			initErr = fmt.Errorf("failed to setup Raft: %v", err)
			return
		}
		broker.Raft = raftInstance
		broker.SnapShotStore = snapshot

		// Initialize Redis client
		redisHost := os.Getenv("REDIS_HOST")
		redisPort := os.Getenv("REDIS_PORT")
		if redisHost == "" || redisPort == "" {
			initErr = fmt.Errorf("REDIS_HOST and REDIS_PORT must be set")
			return
		}

		redisClient, err := redisService.InitRedis()
		if err != nil {
			initErr = fmt.Errorf("failed to init Redis: %v", err)
			return
		}
		broker.RedisClient = redisClient

		// Initialize in-memory database
		dbmsInstance, err := dbms.InitRAM(broker)
		if err != nil {
			initErr = fmt.Errorf("failed to init RAM storage: %v", err)
			return
		}
		broker.dbms = dbmsInstance

		ServerInstance = broker

		nodeID := os.Getenv("NODE_ID")
		logrus.Infof("Broker server initialized successfully on node %s", nodeID)
	})

	return ServerInstance, initErr
}

func InitializeBroker(config *InitConfig) (*BrokerServer, error) {
	var initWg sync.WaitGroup
	var initErr error
	initWg.Add(1)

	go func() {
		defer initWg.Done()

		// Initialize the broker server
		broker, err := SetupBrokerServer()
		if err != nil {
			initErr = fmt.Errorf("failed to setup broker server: %v", err)
			return
		}

		// Wait for Raft cluster to stabilize
		if config.IsInitiator {
			log.Printf("Node %s is the initiator, waiting for cluster formation", config.NodeID)
			time.Sleep(5 * time.Second) // Give time for other nodes to start

			// Wait for leadership
			timeout := time.After(30 * time.Second)
			for {
				if broker.Raft.State() == raft.Leader {
					log.Printf("Node %s became leader", config.NodeID)
					break
				}
				select {
				case <-timeout:
					initErr = fmt.Errorf("timeout waiting for leadership")
					return
				case <-time.After(1 * time.Second):
					continue
				}
			}
		} else {
			// Non-initiator nodes should wait to join the cluster
			log.Printf("Node %s waiting to join cluster", config.NodeID)
			time.Sleep(10 * time.Second) // Give more time for initiator to set up
		}

		// Verify Redis connection
		_, err = broker.RedisClient.Ping(context.Background()).Result()
		if err != nil {
			initErr = fmt.Errorf("redis connection failed: %v", err)
			return
		}

		log.Printf("Node %s initialization complete", config.NodeID)
	}()

	// Wait for initialization with timeout
	done := make(chan struct{})
	go func() {
		initWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		if initErr != nil {
			return nil, initErr
		}
		return ServerInstance, nil
	case <-time.After(45 * time.Second):
		return nil, fmt.Errorf("initialization timeout")
	}
}

// ValidateConfig checks if all required configuration is present
func (c *InitConfig) ValidateConfig() error {
	if c.RedisHost == "" || c.RedisPort == "" {
		return fmt.Errorf("redis configuration missing")
	}
	if c.NodeID == "" {
		return fmt.Errorf("NODE_ID not set")
	}
	if c.TransportPort == "" {
		return fmt.Errorf("TRANSPORT_PORT not set")
	}
	if c.SnapshotPath == "" {
		return fmt.Errorf("SNAPSHOT_PATH not set")
	}
	return nil
}

func (broker *BrokerServer) Publish(ctx context.Context, request *proto.PublishRequest) (*proto.PublishResponse, error) {
	// Input validation
	if request == nil {
		logrus.Error("Publish request is nil")
		return nil, fmt.Errorf("invalid request: request cannot be nil")
	}
	if len(request.Subject) == 0 {
		logrus.Error("Publish subject is empty")
		return nil, fmt.Errorf("invalid request: subject cannot be empty")
	}
	if len(request.Body) == 0 {
		logrus.Error("Publish message body is empty")
		return nil, fmt.Errorf("invalid request: message body cannot be empty")
	}

	// Get broker instance safely
	b, err := GetBrokerServer()
	if err != nil {
		logrus.Errorf("Failed to get broker instance: %v", err)
		return nil, fmt.Errorf("broker initialization error: %v", err)
	}

	// Verify broker components
	if b.Raft == nil {
		logrus.Error("Raft instance not initialized")
		return nil, fmt.Errorf("broker component error: raft instance not initialized")
	}
	if b.RedisClient == nil {
		logrus.Error("Redis client not initialized")
		return nil, fmt.Errorf("broker component error: redis client not initialized")
	}

	// Leadership check with detailed logging
	nodeID := os.Getenv("NODE_ID")
	if b.Raft.State() != raft.Leader {
		leader, _ := b.Raft.LeaderWithID()
		logrus.Infof("Node %s cannot publish - not the leader. Current leader: %s", nodeID, leader)
		if leader == "" {
			return nil, fmt.Errorf("cluster error: no leader available")
		}
		return nil, fmt.Errorf("cluster error: not the leader, current leader is %s", leader)
	}

	// Create message with expiration
	msg := models.Message{
		Id:         0, // Will be set after Raft consensus
		Body:       string(request.Body),
		Expiration: time.Duration(request.ExpirationSeconds) * time.Second,
	}

	// Prepare command for Raft consensus
	cmd := Command{
		Op:      "publish",
		Subject: request.Subject,
		Message: msg,
	}

	// Marshal command with error handling
	cmdBytes, err := json.Marshal(cmd)
	if err != nil {
		logrus.Errorf("Failed to marshal publish command: %v", err)
		return nil, fmt.Errorf("internal error: failed to prepare publish command: %v", err)
	}

	// Apply to Raft log with timeout
	logrus.Debugf("Node %s applying command to Raft log", nodeID)
	applyFuture := b.Raft.Apply(cmdBytes, 500*time.Millisecond)
	if err := applyFuture.Error(); err != nil {
		logrus.Errorf("Node %s failed to apply command to Raft log: %v", nodeID, err)
		return nil, fmt.Errorf("consensus error: failed to replicate message: %v", err)
	}

	// Handle the response from Raft
	response := applyFuture.Response()
	if err, ok := response.(error); ok {
		logrus.Errorf("Command application failed on node %s: %v", nodeID, err)
		return nil, fmt.Errorf("consensus error: failed to process message: %v", err)
	}

	// Extract message ID from response
	messageId, ok := response.(int)
	if !ok {
		logrus.Error("Invalid message ID type returned from Raft apply")
		return nil, fmt.Errorf("internal error: invalid message ID type")
	}
	msg.Id = int(messageId)

	// Double-check leadership before Redis publish
	if b.Raft.State() != raft.Leader {
		logrus.Errorf("Node %s lost leadership during publish operation", nodeID)
		return nil, fmt.Errorf("consensus error: leadership lost during publish operation")
	}

	// Marshal message for Redis
	msgJSON, err := json.Marshal(msg)
	if err != nil {
		logrus.Errorf("Failed to marshal message for Redis: %v", err)
		return nil, fmt.Errorf("internal error: failed to prepare message for storage: %v", err)
	}

	// Publish to Redis with context
	err = b.RedisClient.Publish(ctx, request.Subject, msgJSON).Err()
	if err != nil {
		logrus.Errorf("Failed to publish message to Redis: %v", err)
		return nil, fmt.Errorf("storage error: failed to publish message: %v", err)
	}

	// Set message expiration in Redis if specified
	if request.ExpirationSeconds > 0 {
		key := fmt.Sprintf("message:%s:%d", request.Subject, messageId)
		err = b.RedisClient.Set(ctx, key, msgJSON, time.Duration(request.ExpirationSeconds)*time.Second).Err()
		if err != nil {
			logrus.Errorf("Failed to set message expiration in Redis: %v", err)
			// Don't return error here as message is already published
			// Just log the error and continue
		}
	}

	logrus.Infof("Successfully published message %d to subject %s by leader node %s",
		messageId, request.Subject, nodeID)

	// Return success response
	return &proto.PublishResponse{
		Id: int32(messageId),
	}, nil
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
