package core

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/nireo/mci/pb"
	"google.golang.org/grpc/metadata"
)

type Server struct {
	pb.UnimplementedCoreServer
	BaseLogDir string
}

func (s *Server) StreamLogs(stream pb.Core_StreamLogsServer) error {
	var jobID string
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		log.Println("fialed to get metadata from incoming stream")
		return fmt.Errorf("missing metadata in stream context")
	}

	jobIdValues := md.Get("job_id")
	if len(jobIdValues) == 0 || jobIdValues[0] == "" {
		log.Println("'job_id' not found or is empty in stream metadata")
		return fmt.Errorf("missing or empty 'job_id' in stream metadata")
	}

	jobID = jobIdValues[0]
	log.Printf("[%s] log stream connection opened", jobID)

	jobFilePath := filepath.Join(s.BaseLogDir, jobID)
	file, err := os.OpenFile(jobFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)
	if err != nil {
		return err
	}
	defer func() {
		log.Printf("[%s] closing log file: %s", jobID, jobFilePath)
		if err := file.Close(); err != nil {
			log.Printf("[%s] error closing log file: %s", jobID, err)
		}
	}()

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&pb.Empty{})
		}
		if err != nil {
			return err
		}

		if req == nil || req.Message == "" {
			continue
		}

		if _, err := file.WriteString(req.Message + "\n"); err != nil {
			return fmt.Errorf("failed to write to log file: %w", err)
		}
	}
}
