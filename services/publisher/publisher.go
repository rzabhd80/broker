package publisher

import (
	pb "broker/internals/protos/generated/broker/api/proto"
	"broker/services/brokerClient"
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	clients   = make(map[*websocket.Conn]bool)
	clientsMu sync.Mutex
)

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	clientsMu.Lock()
	clients[conn] = true
	clientsMu.Unlock()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			clientsMu.Lock()
			delete(clients, conn)
			clientsMu.Unlock()
			break
		}
	}
}

func broadcastToClients(id int64, subject string, body []byte) {
	message := map[string]interface{}{
		"id":      id,
		"subject": subject,
		"body":    string(body),
	}

	clientsMu.Lock()
	for client := range clients {
		err := client.WriteJSON(message)
		if err != nil {
			client.Close()
			delete(clients, client)
		}
	}
	clientsMu.Unlock()
}

func startWebSocketServer() {
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/", handleHomePage)
	go func() {
		if err := http.ListenAndServe(":8081", nil); err != nil {
			log.Fatal("WebSocket server error:", err)
		}
	}()
}

func handleHomePage(w http.ResponseWriter, r *http.Request) {
	html := `
	<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>Published Messages</title>
	</head>
	<body>
		<h1>Published Messages</h1>
		<ul id="messages"></ul>
		<script>
			const ws = new WebSocket("ws://" + location.host + "/ws");
			ws.onmessage = function(event) {
				const msg = JSON.parse(event.data);
				const listItem = document.createElement("li");
				listItem.textContent = "ID: " + msg.id + ", Subject: " + msg.subject + ", Body: " + msg.body;
				document.getElementById("messages").appendChild(listItem);
			};
		</script>
	</body>
	</html>
	`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func Publish(client pb.BrokerClient, ctx context.Context) error {
	msgBody := []byte(fmt.Sprintf("message sent at : %v", time.Now()))
	publishResponse, err := client.Publish(ctx, &pb.PublishRequest{
		Subject:           "sample",
		Body:              msgBody,
		ExpirationSeconds: 10000,
	})
	if err != nil {
		return err
	}

	broadcastToClients(int64(publishResponse.Id), "sample", msgBody)

	logrus.Printf("Published message with ID: %d\n", publishResponse.Id)
	return nil
}

func runSingle(host string) {
	conn, err := grpc.Dial(host, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer func(conn *grpc.ClientConn) {
		err := conn.Close()
		if err != nil {

		}
	}(conn)

	client := pb.NewBrokerClient(conn)
	err = Publish(client, context.Background())

	if err != nil {
		fmt.Println("Publish failed: %v", err)
	}
}

func RunClientMassiveMessage(hosts []string) {
	// Start WebSocket server and HTML page
	startWebSocketServer()

	brokerClientNode := brokerClient.NewBrokerClient(hosts)
	err := brokerClientNode.Connect()
	if err != nil {
		return
	}
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.Out = os.Stdout

	conn, err := grpc.NewClient(brokerClientNode.CurrentNode, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatalf("Failed to connect: %v", err)
	}
	defer func(conn *grpc.ClientConn) {
		err := conn.Close()
		if err != nil {
			logger.Fatalf("Failed to close: %v", err)
		}
	}(conn)

	client := pb.NewBrokerClient(conn)
	ctx := context.Background()

	var wg sync.WaitGroup
	ticker := time.NewTicker(5 * time.Second)

	done := make(chan bool)

	wg.Add(1)
	go func(client pb.BrokerClient) {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				go func() {
					err := Publish(client, ctx)
					if err != nil {
						logger.Fatalf("Publish failed: %v", err)
					}
				}()
			}
		}
	}(client)

	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Minute)
		ticker.Stop()
		done <- true
	}()
	wg.Wait()
}
