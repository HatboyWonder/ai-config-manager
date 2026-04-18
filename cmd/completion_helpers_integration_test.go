//go:build integration

package cmd

import (
	"context"
	"testing"

	"github.com/dynatrace-oss/ai-config-manager/v3/pkg/repo"
	"github.com/spf13/cobra"
)

func TestCompletionHelpers_ReturnFastWhenRepoWriteLockHeld(t *testing.T) {
	repoPath := t.TempDir()
	t.Setenv("AIMGR_REPO_PATH", repoPath)

	manager := repo.NewManagerWithPath(repoPath)
	if err := manager.Init(); err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	writeLock, err := manager.AcquireRepoWriteLock(context.Background())
	if err != nil {
		t.Fatalf("failed to acquire repo write lock: %v", err)
	}
	defer func() {
		_ = writeLock.Unlock()
	}()

	cmd := &cobra.Command{}

	resourceSuggestions, resourceDirective := completeResourcesWithOptions(completionOptions{
		includePackages: true,
		multiArg:        true,
	})(cmd, nil, "skill/")
	if resourceDirective != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("expected NoFileComp directive for locked resource completion, got %v", resourceDirective)
	}
	if len(resourceSuggestions) != 0 {
		t.Fatalf("expected no dynamic resource suggestions while lock held, got %v", resourceSuggestions)
	}

	sourceSuggestions, sourceDirective := completeSourceNames(cmd, nil, "")
	if sourceDirective != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("expected NoFileComp directive for locked source completion, got %v", sourceDirective)
	}
	if len(sourceSuggestions) != 0 {
		t.Fatalf("expected no dynamic source suggestions while lock held, got %v", sourceSuggestions)
	}
}
