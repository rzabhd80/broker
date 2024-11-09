const grpc = require('@grpc/grpc-js');
const protoLoader = require('@grpc/proto-loader');
const WebSocket = require('ws');
const path = require('path');

const PROTO_PATH = path.resolve(__dirname, './broker.proto'); // Adjust path to your .proto file
const BROKER_ADDRESS = 'localhost:5001'; // gRPC broker address

// Load gRPC proto
const packageDefinition = protoLoader.loadSync(PROTO_PATH, {
    keepCase: true,
    longs: String,
    enums: String,
    defaults: true,
    oneofs: true,
});
const proto = grpc.loadPackageDefinition(packageDefinition).broker; // Adjust 'broker' based on proto

// Create WebSocket server
const wss = new WebSocket.Server({ port: 8080 });

wss.on('connection', (ws) => {
    console.log('Frontend connected via WebSocket');
    ws.on('close', () => {
        console.log('Frontend disconnected');
    });
});

// Helper function to broadcast messages to all WebSocket clients
function broadcastToClients(message) {
    wss.clients.forEach((client) => {
        if (client.readyState === WebSocket.OPEN) {
            client.send(JSON.stringify(message));
        }
    });
}

// Connect to gRPC broker and subscribe
function subscribeToBroker(channel) {
    const client = new proto.Broker(BROKER_ADDRESS, grpc.credentials.createInsecure());

    const stream = client.Subscribe({ subject: channel });
    stream.on('data', (message) => {
        console.log('Received message:', message);
        broadcastToClients(message);
    });

    stream.on('error', (err) => {
        console.error('Stream error:', err);
    });

    stream.on('end', () => {
        console.log('Stream ended');
    });
}

// Start subscription on specified channel
const channel = 'sample'; // Set your channel/topic here
subscribeToBroker(channel);
console.log(`Listening on WebSocket ws://localhost:8080 and subscribing to gRPC channel "${channel}"`);
