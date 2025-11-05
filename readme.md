# P2P DHT Key-Value Store

A peer-to-peer distributed hash table (DHT) implementation using the Kademlia algorithm in Go. This project provides a decentralized, fault-tolerant key-value storage system with no central authority.

## Features

- **Kademlia Algorithm**: Industry-standard DHT implementation with XOR distance metric
- **Decentralized Architecture**: No single point of failure, fully peer-to-peer
- **Self-Organization**: Automatic network topology discovery and maintenance
- **Fault Tolerance**: Handles node failures and network partitions gracefully
- **Concurrent Operations**: Efficient parallel lookups using goroutines
- **UDP-based RPC**: Lightweight message passing between nodes

## Installation

### Prerequisites
- Go 1.16 or higher

### Build

```bash
go build -o dht main.go
```

## Usage

### Start Bootstrap Node

```bash
./dht 8001
```

### Start Additional Nodes

```bash
./dht 8002 localhost:8001
./dht 8003 localhost:8001
```

### Commands

Once running, nodes provide an interactive shell:

- `put <key> <value>` - Store a key-value pair
- `get <key>` - Retrieve a value
- `status` - Show node status and routing table
- `quit` - Exit

### Example Session

```bash
> put hello world
Stored key 'hello' at 3 nodes

> get hello
Value: world

> status
=== Node Status ===
Node ID: 1a2b3c4d5e6f...
Address: localhost:8002
Bucket 5: 2 nodes
Total known nodes: 3
Stored keys: 2
==================

> quit
Goodbye!
```

## How It Works

### Core Components

- **Node Identity**: 160-bit unique ID (SHA-1 hash)
- **Routing Table**: 160 K-buckets organized by XOR distance
- **RPC Layer**: UDP-based message passing
- **Storage**: Local in-memory key-value store

### Key Parameters

- **K = 8**: Bucket size and replication factor
- **ALPHA = 3**: Parallel lookup concurrency
- **ID_LENGTH = 160**: Node ID size in bits

### RPC Operations

1. **PING/PONG**: Liveness check
2. **FIND_NODE**: Request closest nodes to target
3. **FIND_VALUE**: Search for key in network
4. **STORE**: Store key-value at nodes

### Kademlia Algorithm

The DHT uses XOR distance metric for routing:
- Nodes are organized in a binary tree structure
- Each node maintains K-buckets for different distance ranges
- Lookups are performed iteratively, querying progressively closer nodes
- Data is replicated across K closest nodes to the key hash

## Architecture

```
┌─────────────────┐
│   DHT Node      │
│                 │
│ ┌─────────────┐ │
│ │ Routing     │ │
│ │ Table       │ │
│ │ (K-Buckets) │ │
│ └─────────────┘ │
│                 │
│ ┌─────────────┐ │
│ │ Storage     │ │
│ │ (K-V Map)   │ │
│ └─────────────┘ │
│                 │
│ ┌─────────────┐ │
│ │ RPC Handler │ │
│ │ (UDP)       │ │
│ └─────────────┘ │
└─────────────────┘
```

## Limitations

- **No Persistence**: Data is lost when nodes restart
- **No Security**: No authentication or encryption
- **In-Memory Only**: Storage limited by available RAM
- **Basic Failure Detection**: No sophisticated health checks

## Future Improvements

- Add persistent storage (BoltDB, BadgerDB)
- Implement node authentication and message encryption
- Add data expiration (TTL)
- Improve failure detection and recovery
- Add metrics and monitoring

## Real-World Applications

This implementation demonstrates concepts used in:
- **BitTorrent DHT**: Trackerless peer discovery
- **IPFS**: Decentralized file system routing
- **Ethereum**: Blockchain node discovery

## License

MIT License
