package agent

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/nireo/mci"
	"github.com/nireo/mci/pb"
	"google.golang.org/grpc/metadata"
)

type grpcLogStream struct {
	client  pb.CoreClient
	streams map[string]pb.Core_StreamLogsClient
	mu      sync.Mutex
}

func (gl *grpcLogStream) registerStream(ctx context.Context, jobID string) error {
	gl.mu.Lock()
	defer gl.mu.Unlock()

	if _, exists := gl.streams[jobID]; exists {
		log.Printf("stream already registered %s", jobID)
		return fmt.Errorf("stream is already registered")
	}

	md := metadata.New(map[string]string{"job_id": jobID})
	// this will also overwrite any existing metadata meaning there is no need to worry about
	// duplicate/conflicting metadata.
	streamCtx := metadata.NewOutgoingContext(ctx, md)

	stream, err := gl.client.StreamLogs(streamCtx)
	if err != nil {
		return fmt.Errorf("failed to initiate log stream for job %s: %w", jobID, err)
	}

	gl.streams[jobID] = stream
	log.Printf("%s] stream registered successfully.", jobID)

	return nil
}

func (gl *grpcLogStream) unregisterStream(jobID string) {
	gl.mu.Lock()
	defer gl.mu.Unlock()

	stream, exists := gl.streams[jobID]
	if !exists {
		log.Printf("[%s] attempted to unregister a non-existent stream.", jobID)
		return
	}

	_, err := stream.CloseAndRecv()
	if err != nil {
		if err != io.EOF {
			log.Printf("[stream-%s] error closing log stream: %v", jobID, err)
		}
	} else {
		log.Printf("[stream-%s] stream closed successfully.", jobID)
	}

	delete(gl.streams, jobID)
}

func (gs *grpcLogStream) ReportLog(jobID string, stageName string, logLine string) {
	gs.mu.Lock()
	stream, exists := gs.streams[jobID]
	gs.mu.Unlock()

	if !exists {
		log.Printf("[stream-%s] ERROR: No active stream found for reporting log: %s", jobID, logLine)
		return
	}

	entry := &pb.Log{
		Message: logLine,
	}

	if err := stream.Send(entry); err != nil {
		log.Printf("[stream-%s] ERROR sending log entry: %v. Line: %s", jobID, err, logLine)
		if err == io.EOF {
			log.Printf("[LogStream-%s] Stream closed by server.", jobID)
		}

		go gs.unregisterStream(jobID)
	}
}

type LogReporter func(jobID string, stageName string, logLine string)
type StatusReporter func(jobID string, status pb.JobStatus, errMsg string)

// agent handles actually executing all of the jobs
func setupAgentWorkspace(repoURL, commitSHA string) (string, error) {
	workspaceDir, err := os.MkdirTemp("", "mci-agent-ws-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %v", err)
	}

	cmdClone := exec.Command("git", "clone", repoURL, workspaceDir)
	if output, err := cmdClone.CombinedOutput(); err != nil {
		os.RemoveAll(workspaceDir) // Clean up failed clone
		return "", fmt.Errorf("git clone failed: %w\nOutput: %s", err, string(output))
	}

	cmdCheckout := exec.Command("git", "checkout", commitSHA)
	cmdCheckout.Dir = workspaceDir
	if output, err := cmdCheckout.CombinedOutput(); err != nil {
		os.RemoveAll(workspaceDir)
		return "", fmt.Errorf("git checkout failed: %w\nOutput: %s", err, string(output))
	}

	return workspaceDir, nil
}

func readPipelineDefinition(filePath string) (*mci.Pipeline, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not open pipeline file '%s': %w", filePath, err)
	}
	defer f.Close()

	pipeline, err := mci.PipelineFromReader(f)
	if err != nil {
		return nil, fmt.Errorf("could not parse pipeline file '%s': %w", filePath, err)
	}
	if pipeline.Image == "" {
		return nil, fmt.Errorf("pipeline definition requires a top-level 'image' field")
	}
	return pipeline, nil
}

func executeJob(ctx context.Context, job *pb.Job, statusReporter StatusReporter, logReporter LogReporter) error {
	workspaceDir, err := setupAgentWorkspace(job.RepoUrl, job.CommitSha)
	if err != nil {
		log.Printf("[%s] error: %v", job.Id, err)
		return err
	}
	defer os.RemoveAll(workspaceDir)

	pipelinePath := filepath.Join(workspaceDir, ".mci.yaml")
	pipeline, err := readPipelineDefinition(pipelinePath)
	if err != nil {
		log.Printf("[%s] error: %v", job.Id, err)
		return err
	}
	log.Printf("[%s] pipeline definition loaded. image: %s", job.Id, pipeline.Image)

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Printf("[%s] error: %v", job.Id, err)
		statusReporter(job.Id, pb.JobStatus_FAILURE, fmt.Sprintf("failed to create client: %s", err))
		return err
	}
	defer cli.Close()

	status, err := runPipelineSteps(ctx, cli, job, pipeline, workspaceDir, logReporter)
	if err != nil {
		statusReporter(job.Id, pb.JobStatus_FAILURE, fmt.Sprintf("failed to run pipeline: %s", err))
		return err
	}

	statusReporter(job.Id, status, "")
	log.Printf("[%s] job finished.", job.Id)
	return nil
}

type logStreamer struct {
	jobID       string
	stageName   string
	prefix      string
	logReporter LogReporter
}

func (ls *logStreamer) Write(p []byte) (n int, err error) {
	for line := range strings.SplitSeq(string(p), "\n") {
		if line != "" {
			ls.logReporter(ls.jobID, ls.stageName, ls.prefix+line)
		}
	}
	return len(p), nil
}

func runPipelineSteps(
	ctx context.Context,
	cli *client.Client, job *pb.Job,
	pipeline *mci.Pipeline,
	workspaceDir string,
	logReporter LogReporter,
) (pb.JobStatus, error) {
	log.Printf("[%s] pulling image: %s", job.Id, pipeline.Image)
	pullReader, err := cli.ImagePull(ctx, pipeline.Image, image.PullOptions{})
	if err != nil {
		return pb.JobStatus_FAILURE, fmt.Errorf("failed to pull image %s: %w", pipeline.Image, err)
	}
	defer pullReader.Close()
	io.Copy(io.Discard, pullReader)
	log.Printf("[%s] image %s ready.", job.Id, pipeline.Image)

	containerConfig := &container.Config{
		Image:        pipeline.Image,
		WorkingDir:   "/workspace",
		Tty:          false,
		AttachStdout: true,
		AttachStderr: true,
		OpenStdin:    false,
		Cmd:          []string{"sleep", "infinity"},
	}

	absWorkspaceDir, _ := filepath.Abs(workspaceDir)
	hostConfig := &container.HostConfig{
		Binds: []string{fmt.Sprintf("%s:/workspace", absWorkspaceDir)},
	}

	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, fmt.Sprintf("mci-agent-%s", job.Id))
	if err != nil {
		return pb.JobStatus_FAILURE, fmt.Errorf("failed to create container: %w", err)
	}
	containerID := resp.ID
	log.Printf("[%s] created container: %s", job.Id, containerID[:12])

	defer func() {
		log.Printf("[%s] stopping container %s", job.Id, containerID[:12])
		timeout := 10 * time.Second
		stopCtx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		if err := cli.ContainerStop(stopCtx, containerID, container.StopOptions{}); err != nil {
			log.Printf("[%s] Warning: Failed to stop container %s: %v", job.Id, containerID[:12], err)
		}

		log.Printf("[%s] removing container %s", job.Id, containerID[:12])
		rmCtx, cancelRm := context.WithTimeout(context.Background(), timeout)
		defer cancelRm()
		if err := cli.ContainerRemove(rmCtx, containerID, container.RemoveOptions{Force: true}); err != nil {
			log.Printf("[%s] Warning: Failed to remove container %s: %v", job.Id, containerID[:12], err)
		}
	}()

	if err := cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return pb.JobStatus_FAILURE, fmt.Errorf("failed to start container: %w", err)
	}
	log.Printf("[%s] started container: %s", job.Id, containerID[:12])

	for _, stage := range pipeline.Stages {
		log.Printf("[%s] === Running Stage: %s ===", job.Id, stage.Name)
		logReporter(job.Id, stage.Name, fmt.Sprintf("=== entering Stage: %s ===", stage.Name))

		for _, command := range stage.Commands {
			log.Printf("[%s] executing command: %s", job.Id, command)
			logReporter(job.Id, stage.Name, fmt.Sprintf("$ %s", command))

			execConfig := container.ExecOptions{
				AttachStdout: true,
				AttachStderr: true,
				Cmd:          []string{"sh", "-c", command},
				WorkingDir:   "/workspace",
			}
			execIDResp, err := cli.ContainerExecCreate(ctx, containerID, execConfig)
			if err != nil {
				return pb.JobStatus_FAILURE, fmt.Errorf("failed to create exec for command '%s': %w", command, err)
			}

			hijackedResp, err := cli.ContainerExecAttach(ctx, execIDResp.ID, container.ExecAttachOptions{})
			if err != nil {
				return pb.JobStatus_FAILURE, fmt.Errorf("failed to attach to exec for command '%s': %w", command, err)
			}
			defer hijackedResp.Close()

			logWriter := &logStreamer{jobID: job.Id, stageName: stage.Name, prefix: "", logReporter: logReporter}
			_, err = io.Copy(logWriter, hijackedResp.Reader)
			if err != nil {
				log.Printf("[%s] warning: Error copying log stream: %v", job.Id, err)
			}

			inspectResp, err := cli.ContainerExecInspect(ctx, execIDResp.ID)
			if err != nil {
				return pb.JobStatus_FAILURE, fmt.Errorf("failed to inspect exec for command '%s': %w", command, err)
			}

			log.Printf("[%s] command '%s' finished with exit code %d", job.Id, command, inspectResp.ExitCode)
			logReporter(job.Id, stage.Name, fmt.Sprintf("Exit code: %d", inspectResp.ExitCode))

			if inspectResp.ExitCode != 0 {
				finalErr := fmt.Errorf("command '%s' failed with exit code %d in stage '%s'", command, inspectResp.ExitCode, stage.Name)
				logReporter(job.Id, stage.Name, finalErr.Error())
				return pb.JobStatus_FAILURE, finalErr // Stop pipeline execution on failure
			}
		}
		logReporter(job.Id, stage.Name, fmt.Sprintf("=== Exiting Stage: %s ===", stage.Name))
	}

	return pb.JobStatus_SUCCESS, nil
}

type Agent struct {
	logStreams grpcLogStream
}

func NewAgent() *Agent {
	return &Agent{
		logStreams: grpcLogStream{
			streams: make(map[string]pb.Core_StreamLogsClient),
		},
	}
}

func (a *Agent) handleJob(job *pb.Job) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	err := a.logStreams.registerStream(ctx, job.Id)
	if err != nil {
		return err
	}
	defer a.logStreams.unregisterStream(job.Id)

	statusReporter := func(jobID string, status pb.JobStatus, errMsg string) {
		log.Printf("StatusReporter: Job %s, Status %s, Error: %s", jobID, status.String(), errMsg)
	}

	go func() {
		log.Printf("[%s] starting job execution", job.Id)
		err := executeJob(ctx, job, statusReporter, a.logStreams.ReportLog)
		if err != nil {
			log.Printf("[%s] job execution returned error: %v", job.Id, err)
		} else {
			log.Printf("[%s] job execution completed.", job.Id)
		}
	}()

	return nil
}
