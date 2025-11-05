package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	K            = 8
	ALPHA        = 3
	ID_LENGTH    = 160
	BUCKET_COUNT = ID_LENGTH
	RPC_TIMEOUT  = 5 * time.Second
)

type NodeID [20]byte

type Node struct {
	ID   NodeID
	Addr string
}

type MessageType string

const (
	PING       MessageType = "PING"
	PONG       MessageType = "PONG"
	FIND_NODE  MessageType = "FIND_NODE"
	FIND_VALUE MessageType = "FIND_VALUE"
	STORE      MessageType = "STORE"
	RESPONSE   MessageType = "RESPONSE"
)

type Message struct {
	Type      MessageType `json:"type"`
	Sender    Node        `json:"sender"`
	Key       string      `json:"key,omitempty"`
	Value     string      `json:"value,omitempty"`
	Target    NodeID      `json:"target,omitempty"`
	Nodes     []Node      `json:"nodes,omitempty"`
	RequestID string      `json:"request_id"`
}

type KBucket struct {
	nodes    []Node
	lastSeen time.Time
	mu       sync.RWMutex
}

func (kb *KBucket) addNode(node Node) bool {
	kb.mu.Lock()
	defer kb.mu.Unlock()

	for i, n := range kb.nodes {
		if n.ID == node.ID {
			kb.nodes = append([]Node{node}, append(kb.nodes[:i], kb.nodes[i+1:]...)...)
			kb.lastSeen = time.Now()
			return true
		}
	}

	if len(kb.nodes) < K {
		kb.nodes = append([]Node{node}, kb.nodes...)
		kb.lastSeen = time.Now()
		return true
	}

	return false
}

func (kb *KBucket) getNodes() []Node {
	kb.mu.RLock()
	defer kb.mu.RUnlock()
	nodes := make([]Node, len(kb.nodes))
	copy(nodes, kb.nodes)
	return nodes
}

type DHTNode struct {
	self        Node
	buckets     [BUCKET_COUNT]*KBucket
	storage     map[string]string
	listener    net.PacketConn
	mu          sync.RWMutex
	pendingRPCs map[string]chan Message
	rpcMu       sync.RWMutex
}

func NewDHTNode(addr string) (*DHTNode, error) {
	hash := sha1.Sum([]byte(addr + fmt.Sprintf("%d", time.Now().UnixNano())))

	node := &DHTNode{
		self: Node{
			ID:   hash,
			Addr: addr,
		},
		storage:     make(map[string]string),
		pendingRPCs: make(map[string]chan Message),
	}

	for i := 0; i < BUCKET_COUNT; i++ {
		node.buckets[i] = &KBucket{}
	}

	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	listener, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}

	node.listener = listener

	go node.handleMessages()

	return node, nil
}

func xorDistance(a, b NodeID) *big.Int {
	result := big.NewInt(0)
	for i := 0; i < len(a); i++ {
		result.Lsh(result, 8)
		result.Or(result, big.NewInt(int64(a[i]^b[i])))
	}
	return result
}

func (dht *DHTNode) bucketIndex(target NodeID) int {
	distance := xorDistance(dht.self.ID, target)
	if distance.Cmp(big.NewInt(0)) == 0 {
		return 0
	}

	bitLen := distance.BitLen()
	if bitLen > BUCKET_COUNT {
		return BUCKET_COUNT - 1
	}
	return BUCKET_COUNT - bitLen
}

func (dht *DHTNode) addToRoutingTable(node Node) {
	if node.ID == dht.self.ID {
		return
	}

	index := dht.bucketIndex(node.ID)
	dht.buckets[index].addNode(node)
}

func (dht *DHTNode) findClosestNodes(target NodeID, count int) []Node {
	var allNodes []Node

	for _, bucket := range dht.buckets {
		allNodes = append(allNodes, bucket.getNodes()...)
	}

	sort.Slice(allNodes, func(i, j int) bool {
		distI := xorDistance(allNodes[i].ID, target)
		distJ := xorDistance(allNodes[j].ID, target)
		return distI.Cmp(distJ) < 0
	})

	if len(allNodes) < count {
		return allNodes
	}
	return allNodes[:count]
}

func (dht *DHTNode) sendRPC(target Node, msg Message) (Message, error) {
	msg.Sender = dht.self
	msg.RequestID = fmt.Sprintf("%d", time.Now().UnixNano())

	responseChan := make(chan Message, 1)
	dht.rpcMu.Lock()
	dht.pendingRPCs[msg.RequestID] = responseChan
	dht.rpcMu.Unlock()

	defer func() {
		dht.rpcMu.Lock()
		delete(dht.pendingRPCs, msg.RequestID)
		dht.rpcMu.Unlock()
	}()

	data, err := json.Marshal(msg)
	if err != nil {
		return Message{}, err
	}

	addr, err := net.ResolveUDPAddr("udp", target.Addr)
	if err != nil {
		return Message{}, err
	}

	_, err = dht.listener.WriteTo(data, addr)
	if err != nil {
		return Message{}, err
	}

	select {
	case response := <-responseChan:
		return response, nil
	case <-time.After(RPC_TIMEOUT):
		return Message{}, fmt.Errorf("RPC timeout")
	}
}

func (dht *DHTNode) handleMessages() {
	buffer := make([]byte, 4096)

	for {
		n, addr, err := dht.listener.ReadFrom(buffer)
		if err != nil {
			continue
		}

		var msg Message
		if err := json.Unmarshal(buffer[:n], &msg); err != nil {
			continue
		}

		go dht.processMessage(msg, addr)
	}
}

func (dht *DHTNode) processMessage(msg Message, addr net.Addr) {
	dht.addToRoutingTable(msg.Sender)

	switch msg.Type {
	case PING:
		dht.handlePing(msg, addr)
	case FIND_NODE:
		dht.handleFindNode(msg, addr)
	case FIND_VALUE:
		dht.handleFindValue(msg, addr)
	case STORE:
		dht.handleStore(msg, addr)
	case PONG, RESPONSE:
		dht.handleResponse(msg)
	}
}

func (dht *DHTNode) handlePing(msg Message, addr net.Addr) {
	response := Message{
		Type:      PONG,
		Sender:    dht.self,
		RequestID: msg.RequestID,
	}
	dht.sendResponse(response, addr)
}

func (dht *DHTNode) handleFindNode(msg Message, addr net.Addr) {
	closestNodes := dht.findClosestNodes(msg.Target, K)
	response := Message{
		Type:      RESPONSE,
		Sender:    dht.self,
		Nodes:     closestNodes,
		RequestID: msg.RequestID,
	}
	dht.sendResponse(response, addr)
}

func (dht *DHTNode) handleFindValue(msg Message, addr net.Addr) {
	dht.mu.RLock()
	value, exists := dht.storage[msg.Key]
	dht.mu.RUnlock()

	response := Message{
		Type:      RESPONSE,
		Sender:    dht.self,
		RequestID: msg.RequestID,
	}

	if exists {
		response.Value = value
	} else {
		keyHash := sha1.Sum([]byte(msg.Key))
		response.Nodes = dht.findClosestNodes(keyHash, K)
	}

	dht.sendResponse(response, addr)
}

func (dht *DHTNode) handleStore(msg Message, addr net.Addr) {
	dht.mu.Lock()
	dht.storage[msg.Key] = msg.Value
	dht.mu.Unlock()

	response := Message{
		Type:      RESPONSE,
		Sender:    dht.self,
		RequestID: msg.RequestID,
	}
	dht.sendResponse(response, addr)
}

func (dht *DHTNode) handleResponse(msg Message) {
	dht.rpcMu.RLock()
	responseChan, exists := dht.pendingRPCs[msg.RequestID]
	dht.rpcMu.RUnlock()

	if exists {
		select {
		case responseChan <- msg:
		default:
		}
	}
}

func (dht *DHTNode) sendResponse(msg Message, addr net.Addr) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	dht.listener.WriteTo(data, addr)
}

func (dht *DHTNode) Bootstrap(bootstrapAddr string) error {
	if bootstrapAddr == "" {
		fmt.Println("Starting as bootstrap node")
		return nil
	}

	bootstrapNode := Node{
		Addr: bootstrapAddr,
	}

	pingMsg := Message{Type: PING}
	response, err := dht.sendRPC(bootstrapNode, pingMsg)
	if err != nil {
		return fmt.Errorf("failed to ping bootstrap node: %v", err)
	}

	bootstrapNode.ID = response.Sender.ID
	dht.addToRoutingTable(bootstrapNode)

	dht.lookupNode(dht.self.ID)

	fmt.Printf("Bootstrapped to network via %s\n", bootstrapAddr)
	return nil
}

func (dht *DHTNode) lookupNode(target NodeID) []Node {
	contacted := make(map[NodeID]bool)
	shortlist := dht.findClosestNodes(target, K)

	for len(shortlist) > 0 && len(contacted) < K {
		var toQuery []Node
		for _, node := range shortlist {
			if !contacted[node.ID] && len(toQuery) < ALPHA {
				toQuery = append(toQuery, node)
			}
		}

		if len(toQuery) == 0 {
			break
		}

		var wg sync.WaitGroup
		var mu sync.Mutex
		var newNodes []Node

		for _, node := range toQuery {
			contacted[node.ID] = true
			wg.Add(1)

			go func(n Node) {
				defer wg.Done()

				msg := Message{
					Type:   FIND_NODE,
					Target: target,
				}

				response, err := dht.sendRPC(n, msg)
				if err != nil {
					return
				}

				mu.Lock()
				newNodes = append(newNodes, response.Nodes...)
				mu.Unlock()
			}(node)
		}

		wg.Wait()

		shortlist = append(shortlist, newNodes...)
		sort.Slice(shortlist, func(i, j int) bool {
			distI := xorDistance(shortlist[i].ID, target)
			distJ := xorDistance(shortlist[j].ID, target)
			return distI.Cmp(distJ) < 0
		})

		if len(shortlist) > K {
			shortlist = shortlist[:K]
		}
	}

	return shortlist
}

func (dht *DHTNode) Put(key, value string) error {
	keyHash := sha1.Sum([]byte(key))
	closestNodes := dht.lookupNode(keyHash)

	stored := 0
	for _, node := range closestNodes {
		msg := Message{
			Type:  STORE,
			Key:   key,
			Value: value,
		}

		_, err := dht.sendRPC(node, msg)
		if err == nil {
			stored++
		}
	}

	if len(closestNodes) == 0 || stored < K/2 {
		dht.mu.Lock()
		dht.storage[key] = value
		dht.mu.Unlock()
		stored++
	}

	if stored == 0 {
		return fmt.Errorf("failed to store key in network")
	}

	fmt.Printf("Stored key '%s' at %d nodes\n", key, stored)
	return nil
}

func (dht *DHTNode) Get(key string) (string, error) {
	keyHash := sha1.Sum([]byte(key))

	dht.mu.RLock()
	if value, exists := dht.storage[key]; exists {
		dht.mu.RUnlock()
		return value, nil
	}
	dht.mu.RUnlock()

	contacted := make(map[NodeID]bool)
	shortlist := dht.findClosestNodes(keyHash, K)

	for len(shortlist) > 0 {
		var toQuery []Node
		for _, node := range shortlist {
			if !contacted[node.ID] && len(toQuery) < ALPHA {
				toQuery = append(toQuery, node)
			}
		}

		if len(toQuery) == 0 {
			break
		}

		for _, node := range toQuery {
			contacted[node.ID] = true

			msg := Message{
				Type: FIND_VALUE,
				Key:  key,
			}

			response, err := dht.sendRPC(node, msg)
			if err != nil {
				continue
			}

			if response.Value != "" {
				return response.Value, nil
			}

			shortlist = append(shortlist, response.Nodes...)
		}

		sort.Slice(shortlist, func(i, j int) bool {
			distI := xorDistance(shortlist[i].ID, keyHash)
			distJ := xorDistance(shortlist[j].ID, keyHash)
			return distI.Cmp(distJ) < 0
		})

		if len(shortlist) > K {
			shortlist = shortlist[:K]
		}
	}

	return "", fmt.Errorf("key not found in network")
}

func (dht *DHTNode) PrintStatus() {
	fmt.Printf("\n=== Node Status ===\n")
	fmt.Printf("Node ID: %x\n", dht.self.ID)
	fmt.Printf("Address: %s\n", dht.self.Addr)

	totalNodes := 0
	for i, bucket := range dht.buckets {
		nodes := bucket.getNodes()
		if len(nodes) > 0 {
			fmt.Printf("Bucket %d: %d nodes\n", i, len(nodes))
			totalNodes += len(nodes)
		}
	}

	dht.mu.RLock()
	storageCount := len(dht.storage)
	dht.mu.RUnlock()

	fmt.Printf("Total known nodes: %d\n", totalNodes)
	fmt.Printf("Stored keys: %d\n", storageCount)
	fmt.Println("==================")
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <port> [bootstrap_addr]")
		fmt.Println("Example: go run main.go 8001")
		fmt.Println("         go run main.go 8002 localhost:8001")
		os.Exit(1)
	}

	port := os.Args[1]
	addr := "localhost:" + port

	var bootstrapAddr string
	if len(os.Args) > 2 {
		bootstrapAddr = os.Args[2]
	}

	// Create DHT node
	node, err := NewDHTNode(addr)
	if err != nil {
		log.Fatal("Failed to create DHT node:", err)
	}
	defer node.listener.Close()

	fmt.Printf("DHT node started on %s\n", addr)

	if err := node.Bootstrap(bootstrapAddr); err != nil {
		log.Fatal("Failed to bootstrap:", err)
	}

	fmt.Println("\nCommands:")
	fmt.Println("  put <key> <value> - Store a key-value pair")
	fmt.Println("  get <key>         - Retrieve a value")
	fmt.Println("  status            - Show node status")
	fmt.Println("  quit              - Exit")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\n> ")
		if !scanner.Scan() {
			break
		}

		parts := strings.Fields(scanner.Text())
		if len(parts) == 0 {
			continue
		}

		command := strings.ToLower(parts[0])
		switch command {
		case "put":
			if len(parts) != 3 {
				fmt.Println("Usage: put <key> <value>")
				continue
			}

			key, value := parts[1], parts[2]
			if err := node.Put(key, value); err != nil {
				fmt.Printf("Error storing key: %v\n", err)
			}

		case "get":
			if len(parts) != 2 {
				fmt.Println("Usage: get <key>")
				continue
			}

			key := parts[1]
			value, err := node.Get(key)
			if err != nil {
				fmt.Printf("Error retrieving key: %v\n", err)
			} else {
				fmt.Printf("Value: %s\n", value)
			}

		case "status":
			node.PrintStatus()

		case "quit", "exit":
			fmt.Println("Goodbye!")
			return

		default:
			fmt.Println("Unknown command. Use: put, get, status, quit")
		}
	}
}
