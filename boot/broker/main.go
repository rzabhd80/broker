package main

import (
	"broker/boot/redis"
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
	// redis
	address, found := os.LookupEnv("BROKER_REDIS_HOST")
	password, foundPass := os.LookupEnv("BROKER_REDIS_PASS")
	port, foundPort := os.LookupEnv("BROKER_REDIS_PORT")
	if !foundPort || !found || !foundPass {
		logrus.Fatalf("could get get credentials for connecting to the redis")
	}
	redisClient, err := redis.InitRedis(address, port, password)

	if err != nil {
		logger.Fatalf("Failed to connect to Redis: %v", err)
	}
	logger.Println("Connected to redis successfully")

	// tcp server
	listener, err := net.Listen("tcp", ":8081")
	if err != nil {
		logger.Fatalf("cannot create listener: %v", err)
	}

	server := grpc.NewServer()
	proto.RegisterBrokerServer(server, &broker.BrokerServer{
		Broker: broker2.NewModule(redisClient, db.POSTGRES),
		Tracer: nil,
	})
}
