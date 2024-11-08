package main

import (
	"broker/env"
	"broker/services/publisher"
	"github.com/sirupsen/logrus"
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
	publisher.RunClientMassiveMessage(hosts)

}
