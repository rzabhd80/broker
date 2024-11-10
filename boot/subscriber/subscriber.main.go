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

var broadcast = make(chan []byte)
var clients = make(map[*websocket.Conn]bool)

func main() {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	logger.Out = os.Stdout

	hosts := []string{"localhost:5001"}
	brokerClientNode := brokerClient.NewBrokerClient(hosts)

	kacp := keepalive.ClientParameters{
		Time:                10 * time.Second,
		Timeout:             time.Second,
		PermitWithoutStream: true,
	}

	// Signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start WebSocket broadcaster
	go handleBroadcasts()

	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/", serveHome)

	go func() {
		logger.Info("Starting WebSocket server on :8082")
		if err := http.ListenAndServe(":8083", nil); err != nil {
			logger.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	for {
		// Attempt to connect to gRPC server
		if err := brokerClientNode.Connect(); err != nil {
			logger.Errorf("Failed to setup broker client: %v", err)
			time.Sleep(5 * time.Second) // Wait before retrying
			continue
		}

		conn, err := grpc.Dial(
			brokerClientNode.CurrentNode,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithKeepaliveParams(kacp),
			grpc.WithBlock(),
		)
		if err != nil {
			logger.Errorf("Failed to connect: %v", err)
			time.Sleep(5 * time.Second) // Wait before retrying
			continue
		}
		defer conn.Close()
		logger.Debug("Successfully established gRPC connection")

		client := pb.NewBrokerClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)

		// Subscribe to subject
		subject := "sample"
		logger.Debugf("Subscribing to subject: %s", subject)
		subscribeRequest := &pb.SubscribeRequest{Subject: subject}

		stream, err := client.Subscribe(ctx, subscribeRequest)
		if err != nil {
			logger.Errorf("Subscribe failed: %v", err)
			cancel()
			time.Sleep(5 * time.Second) // Wait before retrying
			continue
		}
		logger.Debug("Successfully created subscription stream")

		messageChan := make(chan *pb.MessageResponse, 100)
		go receiveMessages(stream, messageChan, broadcast, logger)

		for {
			select {
			case msg, ok := <-messageChan:
				if !ok {
					logger.Info("Message channel closed, attempting to reconnect...")
					cancel()
					break
				}
				logger.Infof("Processing message: %s", string(msg.Body))

			case sig := <-sigChan:
				logger.Infof("Received signal: %v", sig)
				cancel()
				return

			case <-ctx.Done():
				logger.Warn("Context canceled, attempting to reconnect...")
				cancel()
				break
			}
		}

		time.Sleep(5 * time.Second) // Wait before trying to reconnect
	}
}

func receiveMessages(stream pb.Broker_SubscribeClient, messageChan chan<- *pb.MessageResponse, broadcast chan<- []byte, logger *logrus.Logger) {
	defer close(messageChan)
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			logger.Info("Stream closed by server")
			return
		}
		if err != nil {
			logger.Errorf("Error receiving message: %v", err)
			return
		}
		logger.Debugf("Received raw message: %v", msg)
		select {
		case messageChan <- msg:
			logger.Debug("Message sent to processing channel")
			broadcast <- msg.Body
		default:
			logger.Warn("Message channel full, dropping message")
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
