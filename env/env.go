package env

import (
	"broker/helpers"
	"fmt"
	"github.com/caarlos0/env"
	"os"
)

type BrokerEnv interface {
	info() (string, error)
}

type BrokerConfig struct {
	Broker BrokerEnv
}

type DevelopmentBrokerConfig struct {
	NodeId      string   `env:"NOD_ID"`
	ClusterNods []string `env:"Cluster_Nods"`
	Host        string   `env:"HOST"`
	Port        string   `env:"PORT"`
	Network     string   `env:"NETWORK"`
}

func (cnf DevelopmentBrokerConfig) info() (string, error) {
	return fmt.Sprintf("host: %s port: %s", cnf.Host, cnf.Port), nil
}

type DeploymentBrokerConfig struct {
	PodName      string `env:"POD_NAME"`
	PodNameSpace string `env:"POD_NAMESPACE"`
	RaftNode     string `env:"RAFT_NODE"`
	BrokerNode   string `env:"BROKER_NODE"`
	Host         string `env:"HOST"`
	Port         string `env:"PORT"`
}

func (cnf DeploymentBrokerConfig) info() (string, error) {
	return fmt.Sprintf("host: %s port: %s", cnf.Host, cnf.Port), nil
}

func readDevelopBrokerEnvironment() (*DevelopmentBrokerConfig, error) {
	cfg := &DevelopmentBrokerConfig{}
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

func SetBrokerEnvironment() (*DevelopmentBrokerConfig, *DeploymentBrokerConfig, error) {
	var envConfDevelop *DevelopmentBrokerConfig
	var envConfDeploy *DeploymentBrokerConfig
	if found := helpers.CheckFileExists("dev.env"); found && os.Getenv("MODE") == "develop" {
		if cnf, err := readDevelopBrokerEnvironment(); err != nil {
			return nil, nil, err
		} else {
			envConfDevelop = cnf
		}
	} else if found := helpers.CheckFileExists("deploy.env"); found && os.Getenv("MODE") == "deployment" {
		if cnf, err := readDeploymentBrokerEnvironment(); err != nil {
			return nil, nil, err
		} else {
			envConfDeploy = cnf
		}
	}
	return envConfDevelop, envConfDeploy, nil
}
