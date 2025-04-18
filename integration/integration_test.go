package integration

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nireo/mci/agent"
	"github.com/nireo/mci/core"
	"github.com/nireo/mci/pb"
	"github.com/nireo/mci/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func setupCoreServer(t *testing.T) (client pb.CoreClient, dir string, cleanup func()) {
	t.Helper()
	dir = t.TempDir()

	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	coreserver := &core.Server{
		BaseLogDir: dir,
	}
	pb.RegisterCoreServer(s, coreserver)
	go func() {
		if err := s.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			log.Printf("WARN: Server exited with non-stopped error: %v", err)
		}
	}()

	conn, err := grpc.DialContext(context.Background(),
		"bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "Failed to dial bufnet")

	cleanup = func() {
		err := conn.Close()
		assert.NoError(t, err, "Error closing client connection")
		s.Stop()
		err = lis.Close()
		assert.NoError(t, err, "Error closing listener")
	}
	client = pb.NewCoreClient(conn)
	return client, dir, cleanup
}

func TestSingleAgentAndCore(t *testing.T) {
	coreClient, dir, cleanup := setupCoreServer(t)
	defer cleanup()
	a := agent.NewAgent(coreClient)

	repoPath, _, err := testutil.CreateTestGitRepo(t, "testrepo3")
	require.NoError(t, err)

	pipelineContent := `
image: alpine:latest
stages:
  - name: setup
    commands:
      - echo "Setup phase"
      - mkdir data
  - name: process
    commands:
      - echo "Processing..." > data/output.txt
      - ls -l /workspace/data
  - name: check
    commands:
      - cat data/output.txt
      - exit 0
`
	commitSHA, err := testutil.AddAndCommitFile(t, repoPath, ".mci.yaml", pipelineContent, "Add successful pipeline")
	require.NoError(t, err)

	repoURL := "file://" + filepath.ToSlash(repoPath)
	jobID := uuid.NewString()
	job := &pb.Job{
		Id:        jobID,
		CommitSha: commitSHA,
		RepoUrl:   repoURL,
	}

	err = a.HandleJob(job)
	require.NoError(t, err)

	time.Sleep(5 * time.Second)

	content, err := os.ReadFile(filepath.Join(dir, jobID))
	require.NoError(t, err)

	fmt.Fprintln(os.Stderr, string(content))
}
