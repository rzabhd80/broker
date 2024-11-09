package main

import (
	pb "broker/internals/protos/generated/broker/api/proto"
	"broker/services/brokerClient"
	"context"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for demo
	},
}

// Global channel for broadcasting messages to all WebSocket clients
var broadcast = make(chan []byte)

// Map to keep track of all connected WebSocket clients
var clients = make(map[*websocket.Conn]bool)

func main() {
	// Setup enhanced logging
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	logger.Out = os.Stdout

	// Setup client with debug info
	hosts := []string{"localhost:5001"}
	brokerClientNode := brokerClient.NewBrokerClient(hosts)
	if err := brokerClientNode.Connect(); err != nil {
		logger.Fatalf("Failed to setup broker client: %v", err)
		return
	}
	logger.Debugf("Connected to broker at: %s", brokerClientNode.CurrentNode)

	// Enhanced gRPC connection with keepalive
	kacp := keepalive.ClientParameters{
		Time:                10 * time.Second,
		Timeout:             time.Second,
		PermitWithoutStream: true,
	}

	conn, err := grpc.Dial(
		brokerClientNode.CurrentNode,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(kacp),
		grpc.WithBlock(),
	)
	if err != nil {
		logger.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()
	logger.Debug("Successfully established gRPC connection")

	// Create broker client
	client := pb.NewBrokerClient(conn)

	// Create context with timeout for initial connection
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Setup subscription with debug info
	subject := "sample"
	logger.Debugf("Subscribing to subject: %s", subject)
	subscribeRequest := &pb.SubscribeRequest{Subject: subject}

	stream, err := client.Subscribe(ctx, subscribeRequest)
	if err != nil {
		logger.Fatalf("Subscribe failed: %v", err)
	}
	logger.Debug("Successfully created subscription stream")

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create message channel with buffer
	messageChan := make(chan *pb.MessageResponse, 100)

	// Start WebSocket broadcaster
	go handleBroadcasts()

	// Setup HTTP server for WebSocket and static files
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/", serveHome)

	// Start HTTP server
	go func() {
		logger.Info("Starting WebSocket server on :8080")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			logger.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Start receive goroutine with debug info
	logger.Info("Starting to process messages...")
	go func() {
		logger.Debug("Starting message receive loop")
		for {
			logger.Debug("Waiting for next message...")
			msg, err := stream.Recv()
			if err == io.EOF {
				logger.Info("Stream closed by server")
				close(messageChan)
				return
			}
			if err != nil {
				logger.Errorf("Error receiving message: %v", err)
				close(messageChan)
				return
			}
			logger.Debugf("Received raw message: %v", msg)
			select {
			case messageChan <- msg:
				logger.Debug("Message sent to processing channel")
				// Broadcast to WebSocket clients
				broadcast <- msg.Body
			default:
				logger.Warn("Message channel full, dropping message")
			}
		}
	}()

	// Test message publishing (optional, for debugging)
	go func() {
		time.Sleep(5 * time.Second)
		testMsg := &pb.PublishRequest{
			Subject: subject,
			Body:    []byte("test message"),
		}
		_, err := client.Publish(context.Background(), testMsg)
		if err != nil {
			logger.Errorf("Failed to publish test message: %v", err)
		} else {
			logger.Debug("Published test message")
		}
	}()

	// Main loop
	for {
		select {
		case msg, ok := <-messageChan:
			if !ok {
				logger.Info("Message channel closed")
				return
			}
			logger.Infof("Processing message: %s", string(msg.Body))

		case sig := <-sigChan:
			logger.Infof("Received signal: %v", sig)
			cancel()
			return

		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				logger.Error("Subscription timed out")
			} else {
				logger.Info("Context cancelled")
			}
			return
		}
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logrus.Errorf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	clients[conn] = true
	defer delete(clients, conn)

	// Keep the connection alive
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func handleBroadcasts() {
	for message := range broadcast {
		for client := range clients {
			err := client.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				logrus.Errorf("Failed to send message: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "./boot/subscriber/index.html")
}
