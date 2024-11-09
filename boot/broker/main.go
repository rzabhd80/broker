package main

import (
	"broker/internals/protos/generated/broker/api/proto"
	"broker/services/broker"
	"fmt"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
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

	server := grpc.NewServer()
	brokerInstance, err := broker.SetupBrokerServer()
	proto.RegisterBrokerServer(server, brokerInstance)
	logger.Println("Broker server started")
	if err := server.Serve(listener); err != nil {
		logger.Fatalf("Server serve failed: %v", err)
	}
}
