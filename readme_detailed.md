# P2P DHT Key-Value Store

A peer-to-peer distributed hash table (DHT) implementation using the Kademlia algorithm in Go. This project demonstrates advanced distributed systems concepts including decentralized routing, fault tolerance, and self-organizing networks.

## 📋 Table of Contents
- [Project Goals](#project-goals)
- [Architecture Overview](#architecture-overview)
- [Implementation Details](#implementation-details)
- [Installation & Usage](#installation--usage)
- [Code Structure](#code-structure)
- [Algorithm Deep Dive](#algorithm-deep-dive)
- [Real-World Applications](#real-world-applications)
- [Testing Scenarios](#testing-scenarios)
- [Limitations & Future Improvements](#limitations--future-improvements)

## 🎯 Project Goals

### Primary Objectives
1. **Implement Kademlia DHT**: Build a working distributed hash table using the industry-standard Kademlia algorithm
2. **Decentralized Architecture**: Create a truly peer-to-peer system with no central authority or single point of failure
3. **Self-Organization**: Enable nodes to automatically discover and maintain network topology
4. **Fault Tolerance**: Handle node failures, network partitions, and message loss gracefully
5. **Scalability**: Support networks ranging from a few nodes to potentially thousands

### Learning Outcomes
- **Distributed Systems**: Understanding consensus, consistency, and partition tolerance
- **Network Programming**: Low-level UDP communication and RPC implementation
- **Concurrency**: Advanced Go goroutines and channel patterns
- **Algorithms**: XOR distance metric, iterative lookups, and routing table management
- **System Design**: Building resilient, scalable distributed applications

## 🏗️ Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Node A        │    │   Node B        │    │   Node C        │
│  ID: 0x1a2b...  │◄──►│  ID: 0x3c4d...  │◄──►│  ID: 0x5e6f...  │
│  Port: 8001     │    │  Port: 8002     │    │  Port: 8003     │
│                 │    │                 │    │                 │
│ ┌─────────────┐ │    │ ┌─────────────┐ │    │ ┌─────────────┐ │
│ │ Routing     │ │    │ │ Routing     │ │    │ │ Routing     │ │
│ │ Table       │ │    │ │ Table       │ │    │ │ Table       │ │
│ │ (K-Buckets) │ │    │ │ (K-Buckets) │ │    │ │ (K-Buckets) │ │
│ └─────────────┘ │    │ └─────────────┘ │    │ └─────────────┘ │
│                 │    │                 │    │                 │
│ ┌─────────────┐ │    │ ┌─────────────┐ │    │ ┌─────────────┐ │
│ │ Local       │ │    │ │ Local       │ │    │ │ Local       │ │
│ │ Storage     │ │    │ │ Storage     │ │    │ │ Storage     │ │
│ └─────────────┘ │    │ └─────────────┘ │    │ └─────────────┘ │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### Core Components

1. **Node Identity**: Each node has a unique 160-bit ID generated from SHA-1 hash
2. **Routing Table**: 160 K-buckets organizing known nodes by XOR distance
3. **RPC Layer**: UDP-based message passing for peer communication
4. **Storage Layer**: Local key-value storage for data responsibility
5. **Lookup Algorithm**: Iterative search for finding nodes and data

## 🔧 Implementation Details

### Node Structure
```go
type DHTNode struct {
    self        Node                    // This node's identity
    buckets     [160]*KBucket          // Routing table (K-buckets)
    storage     map[string]string      // Local key-value store
    listener    net.PacketConn         // UDP listener
    pendingRPCs map[string]chan Message // Async RPC handling
}
```

### Key Parameters
- **K = 8**: Maximum nodes per K-bucket and replication factor
- **ALPHA = 3**: Concurrency parameter for parallel lookups
- **ID_LENGTH = 160**: Node ID size in bits (SHA-1 hash)
- **RPC_TIMEOUT = 5s**: Maximum time to wait for RPC responses

### RPC Operations

#### Core RPCs
1. **PING/PONG**: Liveness check and node discovery
2. **FIND_NODE**: Request K closest nodes to target ID
3. **FIND_VALUE**: Search for key, return value or closest nodes
4. **STORE**: Store key-value pair at responsible nodes

#### Message Flow Example
```
Node A wants to store key "mykey":

1. A calculates hash(mykey) = 0xabc123...
2. A finds K closest nodes to 0xabc123...
3. A sends STORE RPC to each closest node
4. Nodes store the key-value pair locally
5. A receives confirmations
```

### XOR Distance Metric

The heart of Kademlia is the XOR distance metric:
```go
distance = nodeID_A ⊕ nodeID_B
```

**Properties:**
- Symmetric: d(A,B) = d(B,A)
- Identity: d(A,A) = 0
- Triangle inequality: d(A,C) ≤ d(A,B) + d(B,C)

This creates a binary tree structure where each node sees the network from its own perspective.

### Routing Table (K-Buckets)

K-buckets organize nodes by distance ranges:
- Bucket 0: Nodes with distance [2^159, 2^160)
- Bucket 1: Nodes with distance [2^158, 2^159)
- ...
- Bucket 159: Nodes with distance [1, 2)

Each bucket stores up to K nodes, preferring long-lived connections.

## 🚀 Installation & Usage

### Prerequisites
- Go 1.16 or higher
- Network connectivity between nodes

### Quick Start

1. **Clone and build:**
```bash
git clone <repository>
cd dht-p2p
go build -o dht main.go
```

2. **Start bootstrap node:**
```bash
./dht 8001
```
Output:
```
DHT node started on localhost:8001
Starting as bootstrap node
Node ID: 1a2b3c4d5e6f...
```

3. **Start additional nodes:**
```bash
# Terminal 2
./dht 8002 localhost:8001

# Terminal 3  
./dht 8003 localhost:8001
```

### Interactive Commands

Once running, each node provides an interactive shell:

```bash
> put hello world
Stored key 'hello' at 3 nodes

> get hello  
Value: world

> put user:123 {"name":"alice","age":30}
Stored key 'user:123' at 2 nodes

> status
=== Node Status ===
Node ID: 1a2b3c4d5e6f7890abcdef...
Address: localhost:8002
Bucket 5: 2 nodes
Bucket 12: 1 nodes
Total known nodes: 3
Stored keys: 5
==================

> quit
Goodbye!
```

### Advanced Usage Examples

#### Storing Complex Data
```bash
> put config:app {"database":"postgres","port":5432,"debug":true}
> put image:cat.jpg /path/to/image/data/base64encoded...
> put counter:views 1337
```

#### Network Testing
```bash
# Start multiple nodes
for port in {8001..8010}; do
    ./dht $port localhost:8001 &
done

# Test data distribution
> put test1 value1
> put test2 value2  
> put test3 value3
```

## 📁 Code Structure

### Main Components

```
main.go
├── Constants & Configuration
│   ├── K, ALPHA, ID_LENGTH
│   └── RPC_TIMEOUT
├── Data Structures  
│   ├── NodeID, Node
│   ├── Message, MessageType
│   └── KBucket, DHTNode
├── Core DHT Logic
│   ├── XOR distance calculation
│   ├── Routing table management
│   └── Node lookup algorithms
├── RPC Communication
│   ├── UDP message handling
│   ├── Async RPC with timeouts
│   └── Message serialization
├── DHT Operations
│   ├── Bootstrap process
│   ├── Put/Get operations  
│   └── Node discovery
└── CLI Interface
    ├── Interactive commands
    ├── Status reporting
    └── Error handling
```

### Key Functions Breakdown

#### Network Layer
- `NewDHTNode()`: Initialize node with UDP listener
- `handleMessages()`: Main message processing loop
- `sendRPC()`: Async RPC with timeout handling
- `processMessage()`: Route messages to handlers

#### Kademlia Core  
- `xorDistance()`: Calculate XOR metric between node IDs
- `bucketIndex()`: Find appropriate K-bucket for node
- `findClosestNodes()`: Get K closest nodes to target
- `addToRoutingTable()`: Update routing table with new nodes

#### DHT Operations
- `Bootstrap()`: Join network via bootstrap node
- `lookupNode()`: Iterative node lookup algorithm  
- `Put()`: Store key-value in network
- `Get()`: Retrieve value from network

## 🧠 Algorithm Deep Dive

### Bootstrap Process
1. Contact bootstrap node with PING
2. Add bootstrap node to routing table
3. Perform lookup for own node ID
4. Populate routing table with discovered nodes

```
Node B joins via Node A:

B ──PING──► A
B ◄─PONG─── A (A's ID: 0x1a2b...)
B ──FIND_NODE(B's ID)──► A  
B ◄─RESPONSE(closest nodes)─── A
B contacts returned nodes...
```

### Iterative Lookup Algorithm
```go
function lookupNode(target):
    contacted = {}
    shortlist = findClosestNodes(target, K)
    
    while shortlist not empty:
        toQuery = selectUncontacted(shortlist, ALPHA)
        if toQuery empty: break
        
        for each node in toQuery:
            response = sendRPC(node, FIND_NODE(target))
            shortlist.add(response.nodes)
            contacted.add(node)
        
        shortlist = sortByDistance(shortlist, target)[0:K]
    
    return shortlist
```

### Put Operation Flow
1. Calculate `target_id = hash(key)`
2. Find K closest nodes to `target_id`
3. Send STORE RPC to each closest node
4. Nodes store key-value locally
5. Return success if majority confirms

### Get Operation Flow  
1. Calculate `target_id = hash(key)`
2. Check local storage first
3. If not found, perform iterative lookup:
   - Send FIND_VALUE to closest known nodes
   - If value found, return immediately
   - Otherwise, continue with returned closer nodes
4. Return error if exhausted all possibilities

## 🌍 Real-World Applications

### BitTorrent DHT
- **Use Case**: Trackerless torrent file discovery
- **Implementation**: Modified Kademlia for peer discovery
- **Scale**: Millions of nodes worldwide

### IPFS (InterPlanetary File System)
- **Use Case**: Decentralized content addressing
- **Implementation**: Kademlia for content routing
- **Features**: Content-addressable storage

### Ethereum Node Discovery
- **Use Case**: P2P network formation
- **Implementation**: Modified Kademlia for peer discovery
- **Purpose**: Bootstrap blockchain node connections

### Our Implementation Similarities
- XOR distance metric for routing
- K-bucket routing table organization  
- Iterative lookup procedures
- Fault-tolerant node discovery

## 🧪 Testing Scenarios

### Basic Functionality Tests

#### Single Node Network
```bash
# Start one node
./dht 8001

# Test local storage
> put test value
> get test
Value: value
```

#### Two Node Network
```bash
# Terminal 1
./dht 8001

# Terminal 2  
./dht 8002 localhost:8001

# Test cross-node storage
Node1> put shared data123
Node2> get shared
Value: data123
```

#### Multi-Node Network
```bash
# Start 5 nodes
./dht 8001
./dht 8002 localhost:8001  
./dht 8003 localhost:8001
./dht 8004 localhost:8001
./dht 8005 localhost:8001

# Test data distribution
> put key1 value1  # Should replicate to multiple nodes
> put key2 value2
> put key3 value3
```

### Fault Tolerance Tests

#### Node Failure Simulation
1. Start 4 nodes
2. Store data across network
3. Kill 1-2 nodes (Ctrl+C)
4. Verify data still retrievable from remaining nodes

#### Network Partition
1. Start nodes in two groups
2. Store data in each group
3. Reconnect groups
4. Verify data accessibility across partition

### Performance Tests

#### Storage Capacity
```bash
# Bulk load test
for i in {1..1000}; do
    echo "put key$i value$i" 
done | ./dht 8001
```

#### Lookup Performance
```bash
# Time different operations
time echo "get random_key" | ./dht 8001
```

## ⚠️ Limitations & Future Improvements

### Current Limitations

1. **No Persistence**: Data lost when nodes restart
2. **Basic Security**: No authentication or encryption
3. **Simple Failure Detection**: No sophisticated node liveness checking
4. **Memory Storage Only**: No disk-based storage option
5. **UDP Only**: No TCP fallback for large messages

### Potential Improvements

#### Persistence Layer
```go
type PersistentStorage interface {
    Store(key, value string) error
    Load(key string) (string, error)
    Delete(key string) error
    ListKeys() []string
}

// Implementations: BoltDB, BadgerDB, LevelDB
```

#### Security Enhancements
- Node identity verification with digital signatures  
- Message encryption for sensitive data
- Rate limiting and spam protection
- Sybil attack resistance

#### Advanced Features
- **Republishing**: Periodic re-storage of data
- **Data Expiration**: TTL-based key expiration
- **Load Balancing**: Distribute storage load evenly
- **Bandwidth Optimization**: Message compression
- **Monitoring**: Metrics and observability

#### Production Considerations
```go
// Configuration management
type Config struct {
    K             int           `yaml:"k"`
    Alpha         int           `yaml:"alpha"`
    Timeout       time.Duration `yaml:"timeout"`
    BindAddress   string        `yaml:"bind_address"`
    StorageType   string        `yaml:"storage_type"`
    LogLevel      string        `yaml:"log_level"`
}

// Health monitoring  
type Metrics struct {
    ActiveNodes     int64
    StoredKeys      int64
    SuccessfulGets  int64
    FailedGets      int64
    NetworkLatency  time.Duration
}
```

#### Scalability Enhancements
- Connection pooling for high-throughput scenarios
- Batch operations for multiple key-value pairs
- Hierarchical DHT for very large networks
- Geographic awareness for latency optimization

## 🎓 Educational Value

This project demonstrates several advanced computer science concepts:

- **Distributed Algorithms**: Kademlia routing, consensus mechanisms
- **Network Programming**: UDP communication, async I/O, timeout handling  
- **Data Structures**: Hash tables, binary trees, priority queues
- **Concurrency**: Goroutines, channels, race condition prevention
- **System Design**: Fault tolerance, scalability, consistency trade-offs
- **Cryptography**: Hash functions, distance metrics

## 📚 References & Further Reading

- [Kademlia Paper](https://pdos.csail.mit.edu/~petar/papers/maymounkov-kademlia-lncs.pdf) - Original Kademlia specification
- [BitTorrent DHT](http://www.bittorrent.org/beps/bep_0005.html) - Real-world Kademlia implementation
- [Distributed Systems Concepts](https://web.mit.edu/6.824/www/) - MIT distributed systems course
- [Go Concurrency Patterns](https://blog.golang.org/concurrency-patterns) - Advanced Go concurrency

---

## 🤝 Contributing

Contributions welcome! Areas for improvement:
- Add comprehensive test suite
- Implement persistence layer  
- Add security features
- Optimize network performance
- Add detailed logging/metrics

## 📄 License

MIT License - Feel free to use for educational and commercial purposes.