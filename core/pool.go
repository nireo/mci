package core

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nireo/mci/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type AgentPool struct {
	agents       []string
	clients      map[string]pb.AgentClient
	cMutex       sync.RWMutex
	nextIdx      atomic.Uint64
	grpcDialOpts []grpc.DialOption
}

func NewAgentPool(agentAddresses []string) (*AgentPool, error) {
	if len(agentAddresses) == 0 {
		return nil, errors.New("agent pool requires at least one agent address")
	}
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	return &AgentPool{
		agents:       agentAddresses,
		clients:      make(map[string]pb.AgentClient),
		grpcDialOpts: opts,
	}, nil
}

func (ap *AgentPool) getClient(agentAddr string) (pb.AgentClient, error) {
	ap.cMutex.RLock()
	client, ok := ap.clients[agentAddr]
	ap.cMutex.RUnlock()

	if ok {
		return client, nil
	}

	ap.cMutex.Lock()
	defer ap.cMutex.Unlock()

	client, ok = ap.clients[agentAddr]
	if ok {
		return client, nil
	}

	log.Printf("Establishing gRPC connection to agent: %s", agentAddr)
	conn, err := grpc.Dial(agentAddr, ap.grpcDialOpts...)
	if err != nil {
		log.Printf("Failed to dial agent %s: %v", agentAddr, err)
		return nil, fmt.Errorf("failed to dial agent %s: %w", agentAddr, err)
	}

	client = pb.NewAgentClient(conn)
	ap.clients[agentAddr] = client
	log.Printf("Successfully established gRPC connection to agent: %s", agentAddr)
	return client, nil
}

// SelectAndExecuteJob selects an agent using round-robin and sends the job.
func (ap *AgentPool) SelectAndExecuteJob(ctx context.Context, job *pb.Job) error {
	currentIndex := ap.nextIdx.Add(1) - 1
	agentIndex := int(currentIndex % uint64(len(ap.agents)))

	agentAddr := ap.agents[agentIndex]
	log.Printf("Selected agent %s for job ID %s", agentAddr, job.Id)

	client, err := ap.getClient(agentAddr)
	if err != nil {
		return fmt.Errorf("could not get client for agent %s: %w", agentAddr, err)
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute) // 30-second timeout for ExecuteJob
	defer cancel()

	log.Printf("Sending job ID %s to agent %s", job.Id, agentAddr)
	_, err = client.ExecuteJob(ctx, job)
	if err != nil {
		log.Printf("Failed to execute job ID %s on agent %s: %v", job.Id, agentAddr, err)
		return fmt.Errorf("failed RPC ExecuteJob on agent %s: %w", agentAddr, err)
	}

	log.Printf("Successfully sent job ID %s to agent %s", job.Id, agentAddr)
	return nil
}

// CloseConnections closes all cached gRPC client connections.
func (ap *AgentPool) CloseConnections() {
	ap.cMutex.Lock()
	defer ap.cMutex.Unlock()
	log.Println("Closing agent connections...")
	for addr, client := range ap.clients {
		if conn, ok := client.(interface{ Close() error }); ok {
			if err := conn.Close(); err != nil {
				log.Printf("Error closing connection to agent %s: %v", addr, err)
			} else {
				log.Printf("Closed connection to agent %s", addr)
			}
		}
	}
	ap.clients = make(map[string]pb.AgentClient) // Clear the map
	log.Println("Agent connections closed.")
}
