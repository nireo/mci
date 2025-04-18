package testutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testCiFileName = ".mci.yaml"
)

func execWithDirOut(dir string, command ...string) ([]byte, error) {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}

func execWithDir(dir string, command ...string) error {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir = dir
	return cmd.Run()
}

func CreateTestGitRepo(t *testing.T, repoDirName string) (string, func(), error) {
	t.Helper()
	baseTmpDir := t.TempDir()
	repoPath := filepath.Join(baseTmpDir, repoDirName)

	if err := os.Mkdir(repoPath, 0755); err != nil {
		return "", func() {}, fmt.Errorf("failed to create repo dir: %w", err)
	}

	if output, err := execWithDirOut(repoPath, "git", "init"); err != nil {
		return "", func() {}, fmt.Errorf("git init failed: %w\nOutput: %s", err, string(output))
	}

	if err := execWithDir(repoPath, "git", "config", "user.name", "Test User"); err != nil {
		return "", func() {}, fmt.Errorf("git config user.name failed: %w", err)
	}

	if err := execWithDir(repoPath, "git", "config", "user.email", "test@example.com"); err != nil {
		return "", func() {}, fmt.Errorf("git config user.email failed: %w", err)
	}
	cleanup := func() {}
	return repoPath, cleanup, nil
}

func AddAndCommitFile(t *testing.T, repoPath, filename, content, commitMessage string) (string, error) {
	t.Helper()
	filePath := filepath.Join(repoPath, filename)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file %s: %w", filename, err)
	}

	if output, err := execWithDirOut(repoPath, "git", "add", filename); err != nil {
		return "", fmt.Errorf("git add %s failed: %w\nOutput: %s", filename, err, string(output))
	}

	if output, err := execWithDirOut(repoPath, "git", "commit", "-m", commitMessage); err != nil {
		return "", fmt.Errorf("git commit failed: %w\nOutput: %s", err, string(output))
	}

	shaBytes, err := execWithDirOut(repoPath, "git", "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD failed: %w", err)
	}

	return strings.TrimSpace(string(shaBytes)), nil
}
