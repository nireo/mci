package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/docker/docker/client"
	"github.com/google/uuid"
	"github.com/nireo/mci/pb"
	"github.com/nireo/mci/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testRepoDirName = "test-repo"
	testCiFileName  = ".mci.yaml"
)

// test that the repository is correctly cloned etc.
func TestSetupWorkspace_Integration(t *testing.T) {
	repoPath, _, err := testutil.CreateTestGitRepo(t, "testrepo1")
	require.NoError(t, err)

	initialContent := "Initial content"
	initialFile := "readme.md"
	commitSHA1, err := testutil.AddAndCommitFile(t, repoPath, initialFile, initialContent, "Initial commit")
	require.NoError(t, err)
	require.NotEmpty(t, commitSHA1)

	secondContent := "Second version"
	secondFile := "data.txt"
	commitSHA2, err := testutil.AddAndCommitFile(t, repoPath, secondFile, secondContent, "Second commit")
	require.NoError(t, err)
	require.NotEmpty(t, commitSHA2)
	require.NotEqual(t, commitSHA1, commitSHA2)

	t.Run("CheckoutFirstCommit", func(t *testing.T) {
		repoURL := "file://" + filepath.ToSlash(repoPath)
		workspaceDir, err := setupAgentWorkspace(repoURL, commitSHA1)
		require.NoError(t, err)
		defer os.RemoveAll(workspaceDir) // Cleanup workspace for this subtest

		_, errInitial := os.Stat(filepath.Join(workspaceDir, initialFile))
		assert.NoError(t, errInitial, "Initial file should exist")

		_, errSecond := os.Stat(filepath.Join(workspaceDir, secondFile))
		assert.Error(t, errSecond, "Second file should not exist")
		assert.True(t, os.IsNotExist(errSecond))

		cmdSha := exec.Command("git", "rev-parse", "HEAD")
		cmdSha.Dir = workspaceDir
		shaBytes, err := cmdSha.Output()
		require.NoError(t, err)
		assert.Equal(t, commitSHA1, strings.TrimSpace(string(shaBytes)))
	})

	t.Run("CheckoutSecondCommit", func(t *testing.T) {
		repoURL := "file://" + filepath.ToSlash(repoPath)
		workspaceDir, err := setupAgentWorkspace(repoURL, commitSHA2)
		require.NoError(t, err)
		defer os.RemoveAll(workspaceDir)

		_, errInitial := os.Stat(filepath.Join(workspaceDir, initialFile))
		assert.NoError(t, errInitial, "Initial file should exist")

		_, errSecond := os.Stat(filepath.Join(workspaceDir, secondFile))
		assert.NoError(t, errSecond, "Second file should exist")

		cmdSha := exec.Command("git", "rev-parse", "HEAD")
		cmdSha.Dir = workspaceDir
		shaBytes, err := cmdSha.Output()
		require.NoError(t, err)
		assert.Equal(t, commitSHA2, strings.TrimSpace(string(shaBytes)))
	})
}

func TestReadPipelineDefinition_Integration(t *testing.T) {
	repoPath, _, err := testutil.CreateTestGitRepo(t, "testrepo2")
	require.NoError(t, err)

	pipelineContent := `
image: alpine:latest
stages:
  - name: build
    commands:
      - echo "Building..."
      - touch build_artifact
  - name: test
    commands:
      - echo "Testing..."
`
	commitSHA, err := testutil.AddAndCommitFile(t, repoPath, testCiFileName, pipelineContent, "Add pipeline file")
	require.NoError(t, err)

	repoURL := "file://" + filepath.ToSlash(repoPath)
	workspaceDir, err := setupAgentWorkspace(repoURL, commitSHA)
	require.NoError(t, err)
	defer os.RemoveAll(workspaceDir)

	pipelineFilePath := filepath.Join(workspaceDir, testCiFileName)
	pipeline, err := readPipelineDefinition(pipelineFilePath)
	require.NoError(t, err)

	assert.Equal(t, "alpine:latest", pipeline.Image)
	require.Len(t, pipeline.Stages, 2)
	assert.Equal(t, "build", pipeline.Stages[0].Name)
	require.Len(t, pipeline.Stages[0].Commands, 2)
	assert.Equal(t, `echo "Building..."`, pipeline.Stages[0].Commands[0])
	assert.Equal(t, `touch build_artifact`, pipeline.Stages[0].Commands[1])
	assert.Equal(t, "test", pipeline.Stages[1].Name)
	require.Len(t, pipeline.Stages[1].Commands, 1)
	assert.Equal(t, `echo "Testing..."`, pipeline.Stages[1].Commands[0])
}

func TestRunPipelineSteps_Integration_Success(t *testing.T) {
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
	commitSHA, err := testutil.AddAndCommitFile(t, repoPath, testCiFileName, pipelineContent, "Add successful pipeline")
	require.NoError(t, err)

	repoURL := "file://" + filepath.ToSlash(repoPath)
	workspaceDir, err := setupAgentWorkspace(repoURL, commitSHA)
	require.NoError(t, err)
	defer os.RemoveAll(workspaceDir)

	pipeline, err := readPipelineDefinition(filepath.Join(workspaceDir, testCiFileName))
	require.NoError(t, err)

	job := &pb.Job{
		Id:        uuid.NewString(),
		CommitSha: commitSHA,
	}

	var logs []string
	var logsMu sync.Mutex
	mockLogReporter := func(jobID string, stageName string, logLine string) {
		logsMu.Lock()
		defer logsMu.Unlock()
		logs = append(logs, fmt.Sprintf("[%s] %s", stageName, logLine))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	require.NoError(t, err)
	defer cli.Close()

	finalStatus, finalErr := runPipelineSteps(ctx, cli, job, pipeline, workspaceDir, mockLogReporter)

	require.NoError(t, finalErr)
	assert.Equal(t, pb.JobStatus_SUCCESS, finalStatus)

	logsMu.Lock()
	defer logsMu.Unlock()

	t.Log("Captured Logs:\n", strings.Join(logs, "\n")) // Log output for debugging

	assert.Contains(t, strings.Join(logs, "\n"), "[setup] === entering Stage: setup ===")
	assert.Contains(t, strings.Join(logs, "\n"), "[setup] $ echo \"Setup phase\"")
	assert.Contains(t, strings.Join(logs, "\n"), "Setup phase")
	assert.Contains(t, strings.Join(logs, "\n"), "[setup] $ mkdir data")
	assert.Contains(t, strings.Join(logs, "\n"), "[setup] Exit code: 0")
	assert.Contains(t, strings.Join(logs, "\n"), "[setup] === Exiting Stage: setup ===")

	assert.Contains(t, strings.Join(logs, "\n"), "[process] === entering Stage: process ===")
	assert.Contains(t, strings.Join(logs, "\n"), "[process] $ echo \"Processing...\" > data/output.txt")
	assert.Contains(t, strings.Join(logs, "\n"), "[process] $ ls -l /workspace/data")
	assert.Contains(t, strings.Join(logs, "\n"), "output.txt")
	assert.Contains(t, strings.Join(logs, "\n"), "[process] Exit code: 0")
	assert.Contains(t, strings.Join(logs, "\n"), "[process] === Exiting Stage: process ===")

	assert.Contains(t, strings.Join(logs, "\n"), "[check] === entering Stage: check ===")
	assert.Contains(t, strings.Join(logs, "\n"), "[check] $ cat data/output.txt")
	assert.Contains(t, strings.Join(logs, "\n"), "Processing...") // Output of cat
	assert.Contains(t, strings.Join(logs, "\n"), "[check] $ exit 0")
	assert.Contains(t, strings.Join(logs, "\n"), "[check] Exit code: 0")
	assert.Contains(t, strings.Join(logs, "\n"), "[check] === Exiting Stage: check ===")

	// Check if the file was actually created in the host workspace via the volume mount
	hostOutputPath := filepath.Join(workspaceDir, "data", "output.txt")
	_, err = os.Stat(hostOutputPath)
	assert.NoError(t, err, "Output file should exist on host via volume mount")
}

func TestRunPipelineSteps_Integration_Failure(t *testing.T) {
	repoPath, _, err := testutil.CreateTestGitRepo(t, "testrepo4")
	require.NoError(t, err)

	pipelineContent := `
image: alpine:latest
stages:
  - name: build
    commands:
      - echo "Building..."
      - exit 1 # Deliberate failure
  - name: test
    commands:
      - echo "This should not run"
`
	commitSHA, err := testutil.AddAndCommitFile(t, repoPath, testCiFileName, pipelineContent, "Add failing pipeline")
	require.NoError(t, err)

	repoURL := "file://" + filepath.ToSlash(repoPath)
	workspaceDir, err := setupAgentWorkspace(repoURL, commitSHA)
	require.NoError(t, err)
	defer os.RemoveAll(workspaceDir)

	pipeline, err := readPipelineDefinition(filepath.Join(workspaceDir, testCiFileName))
	require.NoError(t, err)

	job := &pb.Job{
		Id:        uuid.NewString(),
		CommitSha: commitSHA,
	}

	var logs []string
	var logsMu sync.Mutex
	mockLogReporter := func(jobID string, stageName string, logLine string) {
		logsMu.Lock()
		defer logsMu.Unlock()
		logs = append(logs, fmt.Sprintf("[%s] %s", stageName, logLine))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	require.NoError(t, err)
	defer cli.Close()

	finalStatus, finalErr := runPipelineSteps(ctx, cli, job, pipeline, workspaceDir, mockLogReporter)

	require.Error(t, finalErr)
	assert.Equal(t, pb.JobStatus_FAILURE, finalStatus)
	assert.Contains(t, finalErr.Error(), "failed with exit code 1")
	assert.Contains(t, finalErr.Error(), "stage 'build'")

	logsMu.Lock()
	defer logsMu.Unlock()

	t.Log("Captured Logs (Failure Test):\n", strings.Join(logs, "\n"))

	assert.Contains(t, strings.Join(logs, "\n"), "[build] === entering Stage: build ===")
	assert.Contains(t, strings.Join(logs, "\n"), "[build] $ echo \"Building...\"")
	assert.Contains(t, strings.Join(logs, "\n"), "Building...")
	assert.Contains(t, strings.Join(logs, "\n"), "[build] $ exit 1")
	assert.Contains(t, strings.Join(logs, "\n"), "[build] Exit code: 1")
	assert.Contains(t, strings.Join(logs, "\n"), "command 'exit 1' failed with exit code 1 in stage 'build'")

	assert.NotContains(t, strings.Join(logs, "\n"), "=== entering Stage: test ===")
	assert.NotContains(t, strings.Join(logs, "\n"), "This should not run")
}
