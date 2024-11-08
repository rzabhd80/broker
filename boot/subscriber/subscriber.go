package main

import (
	"broker/env"
	pb "broker/internals/protos/generated/broker/api/proto"
	"broker/services/brokerClient"
	"context"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"os"
)

func main() {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.Out = os.Stdout
	clientEnv, err := env.ReadDevelopmentBrokerClient()
	if err != nil {
		logger.Fatalf("Could Not Parse Env")
	}
	var hosts []string = clientEnv.KnownHosts
	brokerClientNode := brokerClient.NewBrokerClient(hosts)
	conn, err := grpc.NewClient(brokerClientNode.CurrentNode, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer func(conn *grpc.ClientConn) {
		err := conn.Close()
		if err != nil {

		}
	}(conn)

	client := pb.NewBrokerClient(conn)
	subscribeRequest := &pb.SubscribeRequest{Subject: "sample"}

	stream, err := client.Subscribe(context.Background(), subscribeRequest)
	if err != nil {
		log.Fatalf("Subscribe failed: %v", err)
	}

	for {
		_, err := stream.Recv()
		if err != nil {
			log.Fatalf("Error receiving message: %v", err)
			break
		}
	}
}
