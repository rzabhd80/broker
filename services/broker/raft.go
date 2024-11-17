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
	localNodeID := fmt.Sprintf("node-%s", os.Getenv("NODE_ID"))
	config.LocalID = raft.ServerID(localNodeID)

	// Get host IP address
	hostname, err := os.Hostname()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get hostname: %v", err)
	}

	addrs, err := net.LookupHost(hostname)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to lookup host: %v", err)
	}

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

		// Use a map to prevent duplicate node IDs
		serverMap := make(map[string]raft.Server)

		// Add the current node first
		serverMap[string(config.LocalID)] = raft.Server{
			ID:      config.LocalID,
			Address: raft.ServerAddress(advertiseAddr),
		}

		// Add other nodes to the configuration
		clusterNodes := strings.Split(os.Getenv("CLUSTER_NODES"), ",")
		log.Printf("Processing cluster nodes configuration: %v", clusterNodes)

		for _, node := range clusterNodes {
			node = strings.TrimSpace(node)
			if node == "" {
				continue
			}

			host, _, _ := strings.Cut(node, ":")
			if host == fmt.Sprintf("broker%s", broker.EnvConfig.NodeId) {
				log.Printf("Skipping own node: %s", node)
				continue
			}

			// Extract node number from the broker address (e.g., "broker2:6001" -> "2")
			parts := strings.Split(node, ":")
			if len(parts) != 2 {
				log.Printf("Warning: Invalid node address format: %s", node)
				continue
			}

			brokerName := parts[0]
			nodeNum := strings.TrimPrefix(brokerName, "broker")

			// Validate node number
			if nodeNum == "" {
				log.Printf("Warning: Could not extract node number from: %s", brokerName)
				continue
			}

			nodeID := fmt.Sprintf("node-%s", nodeNum)

			// Check if this node ID already exists
			if _, exists := serverMap[nodeID]; exists {
				log.Printf("Warning: Duplicate node ID detected: %s. Skipping.", nodeID)
				continue
			}

			log.Printf("Adding node to configuration: ID=%s, Address=%s", nodeID, node)

			// Add to the map
			serverMap[nodeID] = raft.Server{
				ID:      raft.ServerID(nodeID),
				Address: raft.ServerAddress(node),
			}
		}

		// Convert map to slice for final configuration
		servers := make([]raft.Server, 0, len(serverMap))
		for _, server := range serverMap {
			servers = append(servers, server)
		}

		log.Printf("Final server configuration: %+v", servers)

		// Validate minimum cluster size
		if len(servers) < 1 {
			return nil, nil, fmt.Errorf("insufficient number of servers for cluster configuration")
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
