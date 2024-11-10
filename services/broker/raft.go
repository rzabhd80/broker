package broker

import (
	"broker/helpers"
	"fmt"
	"github.com/hashicorp/raft"
	"log"
	"os"
	"time"
)

func (broker *BrokerServer) SetupRaft() (*raft.Raft, raft.SnapshotStore, error) {
	log.Printf("raft creation")
	logs := raft.NewInmemStore()
	stable := raft.NewInmemStore()
	snapshots, err := raft.NewFileSnapshotStore(broker.EnvConfig.SnapShotPath, 2, os.Stderr)
	config := raft.DefaultConfig()
	peerAddress := broker.EnvConfig.ClusterNodes
	config.LocalID = raft.ServerID(os.Getenv("NODE_ID"))

	transportAddr := fmt.Sprintf("127.0.0.1:%s", broker.EnvConfig.TransportPort)
	transport, err := raft.NewTCPTransport(transportAddr, nil, 3, 10*time.Second, os.Stderr)
	if err != nil {
		fmt.Printf("error in raft instance %s", err)
		return nil, nil, err
	}

	r, err := raft.NewRaft(config, &broker.fsm, logs, stable, snapshots, transport)
	if err != nil {
		log.Printf("raft creation failed %s", err)
		return nil, nil, err
	}
	log.Println("we got a new raft")
	log.Println(broker.EnvConfig.Initiator)
	if helpers.IsInitiator(os.Getenv("INITIATOR")) == true {
		log.Printf("node is initiator")
		future := r.BootstrapCluster(raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      raft.ServerID(broker.EnvConfig.NodeId),
					Address: raft.ServerAddress(transportAddr),
				},
			},
		})
		if err := future.Error(); err != nil {
			fmt.Printf("future error %s", err)
			return nil, nil, err
		}

		for _, peerAddr := range peerAddress {
			nodeID := raft.ServerID(fmt.Sprintf("node-%s", peerAddr))
			log.Println("adding voters")
			future := r.AddVoter(nodeID, raft.ServerAddress(peerAddr), 0, 0)
			log.Printf("future %s")
			if future.Error() != nil {
				fmt.Printf("future eror %s", err)
				return nil, nil, future.Error()
			}
		}
	} else {
		log.Printf("node is follower")
		future := r.AddVoter(raft.ServerID(broker.EnvConfig.NodeId), raft.ServerAddress("0.0.0.0"), 0, 0)
		if future.Error() != nil {
			fmt.Printf("couldnt add voter%s", err)
			return nil, nil, future.Error()
		}
	}
	return r, snapshots, nil
}
