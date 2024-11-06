package broker

import (
	"fmt"
	"github.com/hashicorp/raft"
	"os"
	"time"
)

func (broker *BrokerServer) SetupRaft() (*raft.Raft, raft.SnapshotStore, error) {
	logs := raft.NewInmemStore()
	stable := raft.NewInmemStore()
	snapshots, err := raft.NewFileSnapshotStore(broker.EnvConfig.SnapShotPath, 2, os.Stderr)
	config := raft.DefaultConfig()
	peerAddress := broker.EnvConfig.ClusterNodes
	transportAddr := fmt.Sprintf(":%s", broker.EnvConfig.Port)
	transport, err := raft.NewTCPTransport(transportAddr, nil, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return nil, nil, err
	}

	r, err := raft.NewRaft(config, &broker.fsm, logs, stable, snapshots, transport)
	if err != nil {
		return nil, nil, err
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
		return nil, nil, err
	}

	for _, peerAddr := range peerAddress {
		future := r.AddVoter(raft.ServerID("node-id-peer"), raft.ServerAddress(peerAddr), 0, 0)
		if future.Error() != nil {
			return nil, nil, future.Error()
		}
	}

	return r, snapshots, nil
}
