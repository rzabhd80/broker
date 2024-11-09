package publisher

import (
	pb "broker/internals/protos/generated/broker/api/proto"
	"broker/services/brokerClient"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"os"
	"sync"
	"time"
)

func Publish(client pb.BrokerClient, ctx context.Context) error {
	publishResponse, err := client.Publish(ctx, &pb.PublishRequest{
		Subject:           "sample",
		Body:              []byte(fmt.Sprintf("message sent at : %v", time.Now())),
		ExpirationSeconds: 10000,
	})
	if err != nil {
		return err
	}

	logrus.Printf("Published message with ID: %d\n", publishResponse.Id)
	return nil
}

func runSingle(host string) {
	conn, err := grpc.Dial(host, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer func(conn *grpc.ClientConn) {
		err := conn.Close()
		if err != nil {

		}
	}(conn)

	client := pb.NewBrokerClient(conn)
	err = Publish(client, context.Background())

	if err != nil {
		fmt.Println("Publish failed: %v", err)
	}

}

func RunClientMassiveMessage(hosts []string) {
	brokerClientNode := brokerClient.NewBrokerClient(hosts)
	err := brokerClientNode.Connect()
	if err != nil {
		return
	}
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.Out = os.Stdout

	conn, err := grpc.NewClient(brokerClientNode.CurrentNode, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatalf("Failed to connect: %v", err)
	}
	defer func(conn *grpc.ClientConn) {
		err := conn.Close()
		if err != nil {
			logger.Fatalf("Failed to close: %v", err)
		}
	}(conn)

	client := pb.NewBrokerClient(conn)
	ctx := context.Background()

	var wg sync.WaitGroup
	ticker := time.NewTicker(5 * time.Second)

	done := make(chan bool)

	wg.Add(1)
	go func(client pb.BrokerClient) {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				go func() {
					err := Publish(client, ctx)
					if err != nil {
						logger.Fatalf("Publish failed: %v", err)
					}
				}()
			}
		}
	}(client)

	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Minute)
		ticker.Stop()
		done <- true
	}()
	wg.Wait()
}
