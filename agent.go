package mci

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

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

func readPipelineDefinition(filePath string) (*Pipeline, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not open pipeline file '%s': %w", filePath, err)
	}
	defer f.Close()

	pipeline, err := PipelineFromReader(f)
	if err != nil {
		return nil, fmt.Errorf("could not parse pipeline file '%s': %w", filePath, err)
	}
	if pipeline.Image == "" {
		return nil, fmt.Errorf("pipeline definition requires a top-level 'image' field")
	}
	return pipeline, nil
}

func executeJob(ctx context.Context, job Job) error {
	workspaceDir, err := setupAgentWorkspace(job.RepoCloneURL, job.CommitSHA)
	if err != nil {
		log.Printf("[%s] error: %v", job.ID, err)
		return err
	}
	defer os.RemoveAll(workspaceDir)

	pipelinePath := filepath.Join(workspaceDir, ".mci.yaml")
	pipeline, err := readPipelineDefinition(pipelinePath)
	if err != nil {
		log.Printf("[%s] error: %v", job.ID, err)
		return err
	}
	log.Printf("[%s] pipeline definition loaded. image: %s", job.ID, pipeline.Image)

	// TODO: init docker

	log.Printf("[%s] Job finished.", job.ID)
	return nil
}
