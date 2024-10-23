# A Fault Tolenrat Message Broker using Raft Algorithm


## Overview

This project is a **multi-service Golang application** designed to implement a fault tolenrant distributed message broker system. The project follows a phased development approach, incorporating message broker services, publisher and subscriber services, Raft-based leader election, peer-node synchronization, and cluster simulation using Docker Compose. **Protocol Buffers (Protobuf)** is crucial for communication between publishers and subscribers, while **HashiCorp Raft** ensures distributed consensus and leader election across broker nodes. The project also integrates monitoring tools like K6 and Prometheus in the final phase.

## Table of Contents
- [Message Broker Project](#message-broker-project)
  - [Overview](#overview)
  - [Table of Contents](#table-of-contents)
  - [Core Architecture](#core-architecture)
  - [Phases of Development](#phases-of-development)
    - [Phase 1: Core Architecture](#phase-1-core-architecture)
    - [Phase 2: Message Broker Service](#phase-2-message-broker-service)
    - [Phase 3: Publisher Service](#phase-3-publisher-service)
    - [Phase 4: Subscriber Service](#phase-4-subscriber-service)
    - [Phase 5: Raft Integration](#phase-5-raft-integration)
    - [Phase 6: Monitoring Integration](#phase-6-monitoring-integration)
  - [Leader Discovery and Node Communication](#leader-discovery-and-node-communication)
  - [Using Redis for Pub/Sub](#using-redis-for-pubsub)
  - [Environment Configuration](#environment-configuration)
  - [Getting Started](#getting-started)
  - [Contributing](#contributing)

## Core Architecture

The core architecture is a multi-service Golang application with various modules, including:

- **`cmd`**: Command-line interface logic.
- **`services`**: Business logic for each service (broker, publisher, subscriber).
- **`internals`**: Shared internal utilities and helpers.
- **`dbms`**: Direct database interaction using raw SQL queries for performance optimization. No ORM is used.
- **`protos`**: Protocol Buffers definitions for message broker communication.
- **`env`**: Environment configuration management.
- **`config`**: Central configuration service for handling environment variables.

Each module is designed to be scalable and modular, allowing for easy extension in future iterations.

## Phases of Development

### Phase 1: Core Architecture

The first phase focuses on setting up the core services, bootstrapping the project with essential modules, and defining the communication patterns. Raw SQL queries are used for database operations to optimize performance. The architecture is modular to allow easy scaling and the addition of future services.

### Phase 2: Message Broker Service

The message broker service is developed using **Golang** and the **HashiCorp Raft** library for distributed consensus. The broker handles the following tasks:

- Publishing messages to subscribers.
- Maintaining leader election and consensus using Raft.
- Managing the finite state machine (FSM) to replicate messages across peer nodes.
- Simulating a Raft-based cluster using Docker Compose.

#### Finite State Machine (FSM)
The **FSM** in Raft is used to log and store messages that are broadcast to all subscribers across broker nodes. The FSM ensures consistency by replicating the message log to all peer nodes. Raft handles leader election and log replication internally.

```go
func (f *FSM) Apply(log *raft.Log) interface{} {
    var msg model.Message
    err := json.Unmarshal(log.Data, &msg)
    if err != nil {
        return err
    }

    f.messageLog[msg.Subject] = append(f.messageLog[msg.Subject], msg)
    return nil
}

### Phase 3: Publisher Service

In this phase, we develop the publisher service, which is responsible for sending messages to the broker service. The publisher service utilizes **gRPC** and **Protocol Buffers (Protobuf)** for efficient communication. When a message is published, it is first stored in the database, ensuring persistence, and then it is sent to the Redis Pub/Sub system under the relevant subject (topic). This mechanism allows for seamless communication between the publisher and the broker, enabling effective message distribution.

### Phase 4: Subscriber Service

The subscriber service listens to specific subjects for incoming messages. It leverages Redis' Pub/Sub capabilities to receive notifications whenever new messages are published to subscribed topics. Upon receiving a message, the subscriber processes it, making it available to downstream services or components that require the information. This service is crucial for enabling real-time communication between publishers and subscribers.

### Phase 5: Raft Integration

In this phase, we integrate **HashiCorp Raft** to implement leader election and log replication across the broker nodes. Only the Raft leader is authorized to handle message publications, ensuring consistency and reliability in message processing. 

The **Finite State Machine (FSM)** of Raft plays a vital role in this phase. It is responsible for maintaining the state of the messages and ensuring that every new message is applied to the leader's log and subsequently replicated to all follower nodes. This guarantees that all nodes have a consistent view of the message log.

Leader discovery is implemented using Raft's internal mechanisms, allowing nodes to identify the current leader. Once elected, the leader becomes the primary point of contact for message operations, while follower nodes redirect their requests to the leader, ensuring that all changes are processed correctly.

### Phase 6: Monitoring Integration

The final phase involves integrating monitoring and performance testing tools into the architecture. **K6** is used for performance testing, allowing the system to be stress-tested under various workloads to ensure robustness and scalability. Meanwhile, **Prometheus** is configured to collect metrics such as message throughput, latency, and replication performance, providing insights into the broker's operational health and facilitating proactive monitoring.

### Leader Discovery and Node Communication

Leader discovery allows nodes within the Raft cluster to identify which peer is currently the leader. This mechanism is essential for ensuring that only the leader processes state changes and message replication requests. Non-leader nodes will direct their requests to the leader, maintaining the consistency of the system.

### Using Redis for Pub/Sub

The message broker uses Redis for its Pub/Sub functionality. Publishers send messages to specific subjects (topics) in Redis, while subscribers listen to those subjects for real-time updates. This approach decouples the producer and consumer, allowing for flexible and scalable messaging patterns.

### Environment Configuration

The project utilizes environment variables for configuration management, including settings for Redis, the database, and Raft. These variables are loaded from a `.env` file and managed by a centralized configuration service, ensuring that sensitive information is not hardcoded into the application.

## Getting Started

1. **Clone the repository**.
2. **Set up the environment** by copying the example `.env` file.
3. **Build and run the services** using Docker Compose.
4. **Access the broker service** at the specified localhost address.

## Contributing

We welcome contributions! To contribute, please fork the repository, create a feature branch, commit your changes, and submit a pull request. Ensure adherence to coding conventions and provide documentation for any new features or modifications.

