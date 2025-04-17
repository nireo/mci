package agent

import (
	"context"

	"github.com/nireo/mci/pb"
)

type AgentServer struct {
	pb.UnimplementedAgentServer
	jobQueue chan *pb.Job
}

func (s *AgentServer) ExecuteJob(ctx context.Context, req *pb.Job) (*pb.Empty, error) {
	return nil, nil
}
