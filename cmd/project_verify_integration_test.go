//go:build integration

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dynatrace-oss/ai-config-manager/v3/pkg/manifest"
	"github.com/dynatrace-oss/ai-config-manager/v3/pkg/repo"
)

func TestProjectVerifyCommand(t *testing.T) {
	// Create temp directories
	projectDir := t.TempDir()
	repoDir := t.TempDir()

	// Set AIMGR_REPO_PATH
	oldEnv := os.Getenv("AIMGR_REPO_PATH")
	defer func() {
		if oldEnv != "" {
			_ = os.Setenv("AIMGR_REPO_PATH", oldEnv)
		} else {
			_ = os.Unsetenv("AIMGR_REPO_PATH")
		}
	}()
	_ = os.Setenv("AIMGR_REPO_PATH", repoDir)

	// Initialize repo
	manager := repo.NewManagerWithPath(repoDir)
	if err := manager.Init(); err != nil {
		t.Fatalf("Failed to initialize repo: %v", err)
	}

	// Create tool directory with valid symlink
	claudeDir := filepath.Join(projectDir, ".claude", "commands")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("Failed to create tool directory: %v", err)
	}

	// Create repo command
	repoCommand := filepath.Join(repoDir, "commands", "test-cmd")
	if err := os.WriteFile(repoCommand, []byte("#!/bin/bash\necho test"), 0755); err != nil {
		t.Fatalf("Failed to create repo command: %v", err)
	}

	// Create symlink
	symlinkPath := filepath.Join(claudeDir, "test-cmd")
	if err := os.Symlink(repoCommand, symlinkPath); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	// Change to project directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Failed to change to project directory: %v", err)
	}

	// Run verify command
	err = projectVerifyCmd.RunE(projectVerifyCmd, []string{})
	if err != nil {
		t.Errorf("Verify command failed: %v", err)
	}
}

// TestProjectVerifyFixUsesRepairReconcile verifies that verify --fix uses the
// repair/reconcile flow to remove undeclared resources while still warning that
// --fix is deprecated.
func TestProjectVerifyFixUsesRepairReconcile(t *testing.T) {
	projectDir := t.TempDir()
	repoDir := t.TempDir()

	oldEnv := os.Getenv("AIMGR_REPO_PATH")
	defer func() {
		if oldEnv != "" {
			_ = os.Setenv("AIMGR_REPO_PATH", oldEnv)
		} else {
			_ = os.Unsetenv("AIMGR_REPO_PATH")
		}
	}()
	_ = os.Setenv("AIMGR_REPO_PATH", repoDir)

	manager := repo.NewManagerWithPath(repoDir)
	if err := manager.Init(); err != nil {
		t.Fatalf("Failed to initialize repo: %v", err)
	}

	// Empty manifest means any installed content in owned dirs is undeclared.
	m := &manifest.Manifest{Resources: []string{}}
	if err := m.Save(filepath.Join(projectDir, manifest.ManifestFileName)); err != nil {
		t.Fatalf("Failed to save manifest: %v", err)
	}

	commandsDir := filepath.Join(projectDir, ".claude", "commands")
	if err := os.MkdirAll(commandsDir, 0755); err != nil {
		t.Fatalf("Failed to create commands directory: %v", err)
	}
	undeclaredPath := filepath.Join(commandsDir, "orphan.md")
	if err := os.WriteFile(undeclaredPath, []byte("orphan"), 0644); err != nil {
		t.Fatalf("Failed to write undeclared file: %v", err)
	}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Failed to change to project directory: %v", err)
	}

	oldFix := verifyFixFlag
	verifyFixFlag = true
	defer func() { verifyFixFlag = oldFix }()

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err = projectVerifyCmd.RunE(projectVerifyCmd, []string{})

	w.Close()
	os.Stderr = oldStderr
	captured := make([]byte, 4096)
	n, _ := r.Read(captured)
	stderrOutput := string(captured[:n])

	if err != nil {
		t.Fatalf("verify --fix failed: %v", err)
	}
	if !strings.Contains(stderrOutput, "Warning: --fix is deprecated. Running 'aimgr repair' reconciliation.") {
		t.Errorf("Expected deprecation warning on stderr, got: %q", stderrOutput)
	}
	if _, statErr := os.Stat(undeclaredPath); !os.IsNotExist(statErr) {
		t.Fatalf("Expected undeclared path to be removed by reconcile, stat err: %v", statErr)
	}
}

// TestVerifyFixDeprecationWarning verifies that using --fix with 'aimgr verify'
// prints a deprecation warning to stderr and still proceeds with fix behavior.
func TestVerifyFixDeprecationWarning(t *testing.T) {
	projectDir := t.TempDir()
	repoDir := t.TempDir()

	// Set AIMGR_REPO_PATH
	oldEnv := os.Getenv("AIMGR_REPO_PATH")
	defer func() {
		if oldEnv != "" {
			_ = os.Setenv("AIMGR_REPO_PATH", oldEnv)
		} else {
			_ = os.Unsetenv("AIMGR_REPO_PATH")
		}
	}()
	_ = os.Setenv("AIMGR_REPO_PATH", repoDir)

	// Initialize repo
	manager := repo.NewManagerWithPath(repoDir)
	if err := manager.Init(); err != nil {
		t.Fatalf("Failed to initialize repo: %v", err)
	}

	// Create tool directory with a broken symlink so there are issues to fix
	skillsDir := filepath.Join(projectDir, ".opencode", "skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatalf("Failed to create skills directory: %v", err)
	}
	brokenSymlink := filepath.Join(skillsDir, "broken-skill")
	if err := os.Symlink("/nonexistent/target/skills/broken-skill", brokenSymlink); err != nil {
		t.Fatalf("Failed to create broken symlink: %v", err)
	}

	// Change to project directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Failed to change to project directory: %v", err)
	}

	// Set the fix flag
	oldFix := verifyFixFlag
	verifyFixFlag = true
	defer func() { verifyFixFlag = oldFix }()

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Run verify command (ignore the error — the fix logic may fail since skill is gone from repo)
	_ = projectVerifyCmd.RunE(projectVerifyCmd, []string{})

	// Restore stderr and read captured output
	w.Close()
	os.Stderr = oldStderr
	captured := make([]byte, 4096)
	n, _ := r.Read(captured)
	stderrOutput := string(captured[:n])

	// Verify deprecation warning was printed
	if !strings.Contains(stderrOutput, "Warning: --fix is deprecated. Running 'aimgr repair' reconciliation.") {
		t.Errorf("Expected deprecation warning on stderr, got: %q", stderrOutput)
	}
}
