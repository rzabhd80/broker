package env

import (
	"github.com/caarlos0/env"
	"github.com/rzabhd80/broker/helpers/helper"
)

type TestBrokerConfig struct {
	NodeId      string `env:"NOD_ID"`
	ClusterNods string `env:"Cluster_Nods"`
	Host        string `env:"HOST"`
	Port        string `env:"PORT"`
	Network     string `env:"NETWORK"`
}

type DeploymentBrokerConfig struct {
	PodName      string `env:"POD_NAME"`
	PodNameSpace string `env:"POD_NAMESPACE"`
	RaftNode     string `env:"RAFT_NODE"`
	BrokerNode   string `env:"BROKER_NODE"`
}

func readTestBrokerEnvironment() (*TestBrokerConfig, error) {
	cfg := &TestBrokerConfig{}
	err := env.Parse(cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func readDeploymentBrokerEnvironment() (*DeploymentBrokerConfig, error) {
	cfg := &DeploymentBrokerConfig{}
	err := env.Parse(cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func SetBrokerEnvironment() error {

}
