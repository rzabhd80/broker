package main

import (
	"broker/env"
	"broker/services/publisher"
	"fmt"
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
	fmt.Printf("%s", clientEnv.KnownHosts)
	var hosts = []string{"localhost:5003"}
	fmt.Printf("%s", hosts)
	publisher.RunClientMassiveMessage(hosts)
}
