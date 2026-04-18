//go:build integration

package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dynatrace-oss/ai-config-manager/v3/pkg/resource"
	"github.com/dynatrace-oss/ai-config-manager/v3/pkg/tools"
)

func TestCleanOwnedResourceDirs_RemovesAllEntryTypesAndKeepsRoots(t *testing.T) {
	projectDir := t.TempDir()

	commandsDir := filepath.Join(projectDir, ".claude", "commands")
	skillsDir := filepath.Join(projectDir, ".claude", "skills")
	agentsDir := filepath.Join(projectDir, ".claude", "agents")
	for _, d := range []string{commandsDir, skillsDir, agentsDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}

	regularFile := filepath.Join(commandsDir, "manual.md")
	if err := os.WriteFile(regularFile, []byte("manual"), 0644); err != nil {
		t.Fatalf("write regular file: %v", err)
	}

	symlinkTarget := filepath.Join(projectDir, "elsewhere.txt")
	if err := os.WriteFile(symlinkTarget, []byte("outside"), 0644); err != nil {
		t.Fatalf("write symlink target: %v", err)
	}
	if err := os.Symlink(symlinkTarget, filepath.Join(skillsDir, "wrong-repo-link")); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	if err := os.Symlink(filepath.Join(projectDir, "missing-target"), filepath.Join(skillsDir, "broken-link")); err != nil {
		t.Fatalf("create broken symlink: %v", err)
	}

	nested := filepath.Join(agentsDir, "namespace", "inner.txt")
	if err := os.MkdirAll(filepath.Dir(nested), 0755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	if err := os.WriteFile(nested, []byte("nested"), 0644); err != nil {
		t.Fatalf("write nested file: %v", err)
	}

	owned := []OwnedResourceDir{
		{Path: commandsDir, Tool: tools.Claude, ResourceType: resource.Command},
		{Path: skillsDir, Tool: tools.Claude, ResourceType: resource.Skill},
		{Path: agentsDir, Tool: tools.Claude, ResourceType: resource.Agent},
	}

	removed, failed := cleanOwnedResourceDirs(owned)
	if len(failed) != 0 {
		t.Fatalf("expected no failures, got %v", failed)
	}
	if len(removed) != 4 {
		t.Fatalf("expected 4 removed top-level entries, got %d", len(removed))
	}

	for _, d := range []string{commandsDir, skillsDir, agentsDir} {
		entries, err := os.ReadDir(d)
		if err != nil {
			t.Fatalf("readdir %s: %v", d, err)
		}
		if len(entries) != 0 {
			t.Fatalf("expected owned root dir %s to remain empty, found %d entries", d, len(entries))
		}
	}
}
