package broker

import (
	"broker/helpers"
	"fmt"
	"github.com/hashicorp/raft"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

func (broker *BrokerServer) SetupRaft() (*raft.Raft, raft.SnapshotStore, error) {
	log.Printf("Starting Raft setup for node %s", os.Getenv("NODE_ID"))

	// Create stores
	logs := raft.NewInmemStore()
	stable := raft.NewInmemStore()

	// Ensure snapshot directory exists
	if err := os.MkdirAll(broker.EnvConfig.SnapShotPath, 0755); err != nil {
		return nil, nil, fmt.Errorf("failed to create snapshot directory: %v", err)
	}

	snapshots, err := raft.NewFileSnapshotStore(broker.EnvConfig.SnapShotPath, 2, os.Stderr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create snapshot store: %v", err)
	}

	// Configure Raft
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(fmt.Sprintf("node-%s", os.Getenv("NODE_ID")))

	// Get host IP address
	hostname, err := os.Hostname()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get hostname: %v", err)
	}

	addrs, err := net.LookupHost(hostname)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to lookup host: %v", err)
	}

	// Use the first non-loopback IPv4 address
	var hostIP string
	for _, addr := range addrs {
		ip := net.ParseIP(addr)
		if ip.To4() != nil && !ip.IsLoopback() {
			hostIP = ip.String()
			break
		}
	}

	if hostIP == "" {
		return nil, nil, fmt.Errorf("no suitable IP address found")
	}

	// Setup transport with explicit bind and advertise addresses
	bindAddr := fmt.Sprintf("0.0.0.0:%s", broker.EnvConfig.TransportPort)
	advertiseAddr := fmt.Sprintf("%s:%s", hostIP, broker.EnvConfig.TransportPort)

	// Create TCP transport with explicit advertise address
	addr, err := net.ResolveTCPAddr("tcp", advertiseAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve advertise address: %v", err)
	}

	transport, err := raft.NewTCPTransport(
		bindAddr,
		addr, // Explicit advertise address
		3,    // Max pool size
		10*time.Second,
		os.Stderr,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create transport: %v", err)
	}

	r, err := raft.NewRaft(config, &broker.fsm, logs, stable, snapshots, transport)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create raft: %v", err)
	}

	// Handle cluster configuration
	if helpers.IsInitiator(os.Getenv("INITIATOR")) {
		log.Printf("Configuring initiator node %s", config.LocalID)

		// Bootstrap configuration
		servers := []raft.Server{
			{
				ID:      config.LocalID,
				Address: raft.ServerAddress(advertiseAddr),
			},
		}

		// Add other nodes to the configuration
		clusterNodes := strings.Split(os.Getenv("CLUSTER_NODES"), ",")
		for i, node := range clusterNodes {
			if strings.TrimSpace(node) == "" {
				continue
			}

			// Skip if this is our own node address
			if strings.Contains(node, fmt.Sprintf("broker%s", broker.EnvConfig.NodeId)) {
				continue
			}

			// Parse node number from the broker name (e.g., "broker1" -> "1")
			nodeNum := i + 1
			nodeID := fmt.Sprintf("node-%d", nodeNum)

			// Add other nodes
			servers = append(servers, raft.Server{
				ID:      raft.ServerID(nodeID),
				Address: raft.ServerAddress(node),
			})
		}

		// Bootstrap with all servers
		cfg := raft.Configuration{Servers: servers}
		future := r.BootstrapCluster(cfg)
		if err := future.Error(); err != nil {
			return nil, nil, fmt.Errorf("failed to bootstrap cluster: %v", err)
		}

		log.Printf("Successfully bootstrapped cluster with %d nodes", len(servers))
	}

	return r, snapshots, nil
}
