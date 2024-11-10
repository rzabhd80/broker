package main

import (
	"broker/internals/protos/generated/broker/api/proto"
	"broker/services/broker"
	"fmt"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
)

func main() {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.Out = os.Stdout
	raftPort := os.Getenv("RAFT_PORT")
	raftPort = fmt.Sprintf(":%s", raftPort)
	listener, err := net.Listen("tcp", raftPort)
	if err != nil {
		logger.Fatalf("cannot create listener: %v", err)
	}
	config, err := broker.LoadConfigFromEnv()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate configuration
	if err := config.ValidateConfig(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Initialize broker
	brokerInstance, err := broker.InitializeBroker(config)
	if err != nil {
		log.Fatalf("Failed to initialize broker: %v", err)
	}

	server := grpc.NewServer()
	proto.RegisterBrokerServer(server, brokerInstance)
	logger.Println("Broker server started")
	if err := server.Serve(listener); err != nil {
		logger.Fatalf("Server serve failed: %v", err)
	}
}
