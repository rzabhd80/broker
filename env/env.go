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
	Initiator     bool     `env:"INITIATOR"`
	NodeId        string   `env:"NODE_ID"`
	Network       string   `env:"NETWORK"`
	ClusterNodes  []string `env:"CLUSTER_NODES" envSeparator:","`
	TransportPort string   `env:"TRANSPORT_PORT"`
	RedisHost     string   `env:"BROKER_REDIS_HOST"`
	RedisPort     string   `env:"REDIS_PORT"`
	RedisPassword string   `env:"REDIS_PASSWORD"`
	SnapShotPath  string   `env:"SNAPSHOT_PATH"`
}

type DevelopmentClientBrokerConfig struct {
	NodeName   string   `env:"NODE_NAME"`
	KnownHosts []string `env:"KNOWN_HOSTS" envSeparator:","`
}

func (cnf DevelopmentBrokerConfig) info() (string, error) {
	return fmt.Sprintf(""), nil
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

func ReadDevelopmentBrokerClient() (*DevelopmentClientBrokerConfig, error) {
	conf := &DevelopmentClientBrokerConfig{}
	err := env.Parse(conf)
	if err != nil {
		return nil, err
	}
	return conf, nil
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
	if found := helpers.CheckFileExists(".env"); found && os.Getenv("MODE") == "develop" {
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
