package helpers

import (
	pb "broker/internals/protos/generated/broker/api/proto"
	"context"
	"time"
)

type ConnectionChecker struct{}

func NewConnectionChecker() *ConnectionChecker {
	return &ConnectionChecker{}
}

func (c *ConnectionChecker) CheckConnection(client pb.BrokerClient) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Publish(ctx, &pb.PublishRequest{
		Subject:           "_connection_check",
		Body:              []byte("connection_test"),
		ExpirationSeconds: 100, // Minimum expiration
	})

	return err
}
