package main

import (
	"broker/internals/protos/generated/broker/api/proto"
	"broker/services/broker"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
	"os"
)

func main() {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.Out = os.Stdout
	listener, err := net.Listen("tcp", ":8081")
	if err != nil {
		logger.Fatalf("cannot create listener: %v", err)
	}

	server := grpc.NewServer()
	brokerInstance, err := broker.SetupBrokerServer()
	proto.RegisterBrokerServer(server, brokerInstance)
	if err := server.Serve(listener); err != nil {
		logger.Fatalf("Server serve failed: %v", err)
	}
}
