package broker

import (
	"fmt"
	"github.com/hashicorp/raft"
	"os"
	"time"
)

func (broker *BrokerServer) SetupRaft() (*raft.Raft, error) {
	logs := raft.NewInmemStore()
	stable := raft.NewInmemStore()
	snapshots := raft.NewDiscardSnapshotStore()
	config := raft.DefaultConfig()
	peerAddress := broker.envConfig.ClusterNods
	transportAddr := fmt.Sprintf(":%s", broker.envConfig.Port)
	transport, err := raft.NewTCPTransport(transportAddr, nil, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return nil, err
	}

	r, err := raft.NewRaft(config, &broker.fsm, logs, stable, snapshots, transport)
	if err != nil {
		return nil, err
	}

	future := r.BootstrapCluster(raft.Configuration{
		Servers: []raft.Server{
			{
				ID:      raft.ServerID(os.Getenv("NODE_ID")),
				Address: raft.ServerAddress(transportAddr),
			},
		},
	})
	if err := future.Error(); err != nil {
		return nil, err
	}

	// Add the other peers as voters in the cluster
	for _, peerAddr := range peerAddress {
		future := r.AddVoter(raft.ServerID("node-id-peer"), raft.ServerAddress(peerAddr), 0, 0)
		if future.Error() != nil {
			return nil, future.Error()
		}
	}

	return r, nil
}
