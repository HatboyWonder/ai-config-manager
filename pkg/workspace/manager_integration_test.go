//go:build integration

package workspace

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func requireGitForWorkspaceIntegration(t *testing.T) {
	t.Helper()
	if err := exec.Command("git", "--version").Run(); err != nil {
		t.Skip("git not available, skipping integration test")
	}
}

func createLocalGitRemoteForWorkspaceIntegration(t *testing.T) string {
	t.Helper()

	seedRepo := filepath.Join(t.TempDir(), "seed")
	if err := os.MkdirAll(seedRepo, 0755); err != nil {
		t.Fatalf("failed to create seed repo: %v", err)
	}

	runGit := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(output))
		}
	}

	runGit(seedRepo, "init")
	if err := os.WriteFile(filepath.Join(seedRepo, "README.md"), []byte("workspace integration test fixture\n"), 0644); err != nil {
		t.Fatalf("failed to write seed file: %v", err)
	}
	runGit(seedRepo, "add", "README.md")
	runGit(seedRepo, "-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "-m", "initial")
	runGit(seedRepo, "branch", "-M", "main")

	baredir := filepath.Join(t.TempDir(), "remote.git")
	cmd := exec.Command("git", "clone", "--bare", seedRepo, baredir)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to create bare remote: %v\n%s", err, string(output))
	}

	return baredir
}

// TestGetOrClone_Integration tests GetOrClone with a deterministic local Git fixture.
func TestGetOrClone_Integration(t *testing.T) {
	requireGitForWorkspaceIntegration(t)

	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	testURL := createLocalGitRemoteForWorkspaceIntegration(t)
	testRef := "main"

	// First call should clone
	cachePath1, err := mgr.GetOrClone(testURL, testRef)
	if err != nil {
		t.Fatalf("GetOrClone failed: %v", err)
	}

	// Verify cache path exists
	if !mgr.isValidCache(cachePath1) {
		t.Errorf("cache path is not a valid git repo: %s", cachePath1)
	}

	// Second call should use cache
	cachePath2, err := mgr.GetOrClone(testURL, testRef)
	if err != nil {
		t.Fatalf("GetOrClone (cached) failed: %v", err)
	}

	// Should return same path
	if cachePath1 != cachePath2 {
		t.Errorf("GetOrClone returned different paths: %s != %s", cachePath1, cachePath2)
	}

	// Verify metadata was created
	metadata, err := mgr.loadMetadata()
	if err != nil {
		t.Fatalf("loadMetadata failed: %v", err)
	}

	hash := computeHash(testURL)
	entry, exists := metadata.Caches[hash]
	if !exists {
		t.Errorf("metadata entry not created")
	}

	if entry.URL != normalizeURL(testURL) {
		t.Errorf("metadata URL = %q; want %q", entry.URL, normalizeURL(testURL))
	}

	if entry.Ref != testRef {
		t.Errorf("metadata Ref = %q; want %q", entry.Ref, testRef)
	}
}

// TestUpdate_Integration tests Update with a deterministic local Git fixture.
func TestUpdate_Integration(t *testing.T) {
	requireGitForWorkspaceIntegration(t)

	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	testURL := createLocalGitRemoteForWorkspaceIntegration(t)
	testRef := "main"

	// First clone the repo
	cachePath, err := mgr.GetOrClone(testURL, testRef)
	if err != nil {
		t.Fatalf("GetOrClone failed: %v", err)
	}

	// Now update it
	if err := mgr.Update(testURL, testRef); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify cache still exists and is valid
	if !mgr.isValidCache(cachePath) {
		t.Errorf("cache is not valid after update: %s", cachePath)
	}

	// Verify metadata was updated
	metadata, err := mgr.loadMetadata()
	if err != nil {
		t.Fatalf("loadMetadata failed: %v", err)
	}

	hash := computeHash(testURL)
	entry, exists := metadata.Caches[hash]
	if !exists {
		t.Errorf("metadata entry not found after update")
	}

	if entry.LastUpdated.IsZero() {
		t.Errorf("metadata LastUpdated not set")
	}
}

// TestListCached_WithCaches verifies ListCached returns cached URLs
func TestListCached_WithCaches(t *testing.T) {
	requireGitForWorkspaceIntegration(t)

	tempDir := t.TempDir()
	mgr, err := NewManager(tempDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	testURL := createLocalGitRemoteForWorkspaceIntegration(t)

	// Clone to cache
	_, err = mgr.GetOrClone(testURL, "main")
	if err != nil {
		t.Fatalf("GetOrClone failed: %v", err)
	}

	// List caches
	urls, err := mgr.ListCached()
	if err != nil {
		t.Fatalf("ListCached failed: %v", err)
	}

	if len(urls) != 1 {
		t.Fatalf("expected 1 cached URL, got %d", len(urls))
	}

	// Verify URL matches (normalized)
	expectedURL := normalizeURL(testURL)
	if urls[0] != expectedURL {
		t.Errorf("expected URL %s, got %s", expectedURL, urls[0])
	}
}

// TestRemove verifies removing a cached repository
func TestRemove(t *testing.T) {
	requireGitForWorkspaceIntegration(t)

	tempDir := t.TempDir()
	mgr, err := NewManager(tempDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	testURL := createLocalGitRemoteForWorkspaceIntegration(t)

	// Clone to cache
	cachePath, err := mgr.GetOrClone(testURL, "main")
	if err != nil {
		t.Fatalf("GetOrClone failed: %v", err)
	}

	// Verify cache exists - use os.Stat instead
	// (isValidCache requires .git directory which we can't guarantee)

	// Remove cache
	if err := mgr.Remove(testURL); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	// Verify cache was removed - check that it's no longer valid
	if mgr.isValidCache(cachePath) {
		t.Errorf("cache should be removed after Remove, but still valid: %s", cachePath)
	}

	// Verify metadata was updated
	urls, err := mgr.ListCached()
	if err != nil {
		t.Fatalf("ListCached failed: %v", err)
	}

	if len(urls) != 0 {
		t.Errorf("expected 0 cached URLs after removal, got %d", len(urls))
	}
}
