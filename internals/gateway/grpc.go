package gateway

import (
	"broker/env"
	"broker/internals/protos/generated/broker/api/proto"
	"broker/services/broker"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
	"os"
	"os/signal"
	"syscall"
)

type GrpcGateWay struct {
	EnvDev    *env.DevelopmentBrokerConfig
	EnvDeploy *env.DeploymentBrokerConfig
	Logger    logrus.Logger
}

func (gw *GrpcGateWay) EnvSelector() env.BrokerEnv {
	mode := os.Getenv("MODE")
	switch mode {
	case "DEVELOPMENT":
		return gw.EnvDeploy
	case "DEPLOYMENT":
		return gw.EnvDeploy
	default:
		return nil
	}
}

//TODO Implement publisher and subscribers

func (gw *GrpcGateWay) init(ctx context.Context) error {
	envConfDev, envConfDeploy, err := env.SetBrokerEnvironment()
	gw.EnvDev = envConfDev
	gw.EnvDeploy = envConfDeploy
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", gw.EnvDev.TransportPort))
	if err != nil {
		logrus.Panic("could not read the env file")
		panic(nil)
	}
	contx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	server := grpc.NewServer()
	//TODO change the broker instance
	proto.RegisterBrokerServer(server, &broker.BrokerServer{})
	go func() {
		err := server.Serve(listener)
		if err != nil {
			return
		}
	}()
	<-contx.Done()
	server.GracefulStop()
	return nil
}
