package agent

import (
	"context"

	"github.com/nireo/mci/pb"
)

type AgentServer struct {
	pb.UnimplementedAgentServer
	agent *Agent
}

func NewAgentServer(agent *Agent) *AgentServer {
	return &AgentServer{
		agent: agent,
	}
}

func (s *AgentServer) ExecuteJob(ctx context.Context, req *pb.Job) (*pb.Empty, error) {
	err := s.agent.HandleJob(req)
	return &pb.Empty{}, err
}
