package core

import (
	"context"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/nireo/mci/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

func setupTestServer(t *testing.T) (client pb.CoreClient, dir string, cleanup func()) {
	t.Helper()
	dir = t.TempDir()

	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	coreserver := &Server{
		BaseLogDir: dir,
	}
	pb.RegisterCoreServer(s, coreserver)
	go func() {
		if err := s.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			log.Printf("WARN: Server exited with non-stopped error: %v", err)
		}
	}()

	conn, err := grpc.DialContext(context.Background(), // Add a context (e.g., context.Background())
		"bufnet", // Target name remains the same
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "Failed to dial bufnet") // Check error from DialContext

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

func TestStreamLogs_Success(t *testing.T) {
	client, dir, cleanup := setupTestServer(t)
	defer cleanup()

	testJobID := uuid.NewString()
	fp := filepath.Join(dir, testJobID)
	md := metadata.New(map[string]string{"job_id": testJobID})
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	stream, err := client.StreamLogs(ctx)
	require.NoError(t, err, "client failed to start stream logs")

	messagesToSend := []string{"Starting build...", "Step 1 complete.", "Final log line."}
	var expectedContentBuilder strings.Builder
	for _, msgStr := range messagesToSend {
		expectedContentBuilder.WriteString(msgStr + "\n")
		logMsg := &pb.Log{Message: msgStr}
		err = stream.Send(logMsg)
		require.NoError(t, err, "Failed to send message '%s'", msgStr)
	}
	expectedContent := expectedContentBuilder.String()

	err = stream.CloseSend()
	require.NoError(t, err, "Failed to close send direction (CloseSend)")

	_, err = stream.CloseAndRecv()
	require.NoError(t, err, "CloseAndRecv returned an error after successful stream")

	require.FileExists(t, fp, "Log file was not created")
	contentBytes, err := os.ReadFile(fp)
	require.NoError(t, err, "Failed to read created log file")
	assert.Equal(t, expectedContent, string(contentBytes), "Log file content mismatch")
}

func TestStreamLogs_MissingMetadata(t *testing.T) {
	client, _, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	stream, err := client.StreamLogs(ctx)
	require.NoError(t, err, "Initial StreamLogs call should succeed without metadata")

	_, err = stream.CloseAndRecv()

	require.Error(t, err, "Expected an error due to missing metadata")
}

func TestStreamLogs_EmptyJobId(t *testing.T) {
	client, _, cleanup := setupTestServer(t)
	defer cleanup()

	md := metadata.New(map[string]string{"job_id": ""}) // Empty value
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	stream, err := client.StreamLogs(ctx)
	require.NoError(t, err)

	_, err = stream.CloseAndRecv()

	require.Error(t, err, "Expected an error due to empty job_id")
}
