package repo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dynatrace-oss/ai-config-manager/v3/pkg/resource"
)

func TestBuildPackageReferenceIndexFromRoots_SupportsNestedResources(t *testing.T) {
	root := t.TempDir()

	if err := os.MkdirAll(filepath.Join(root, "commands", "team"), 0755); err != nil {
		t.Fatalf("mkdir commands: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "commands", "team", "deploy.md"), []byte("---\ndescription: deploy\n---\n# Deploy\n"), 0644); err != nil {
		t.Fatalf("write command: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(root, "skills", "skill-a"), 0755); err != nil {
		t.Fatalf("mkdir skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "skills", "skill-a", "SKILL.md"), []byte("---\nname: skill-a\ndescription: demo\n---\n# Skill\n"), 0644); err != nil {
		t.Fatalf("write skill: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(root, "agents"), 0755); err != nil {
		t.Fatalf("mkdir agents: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "agents", "reviewer.md"), []byte("---\ndescription: reviewer\n---\n# Agent\n"), 0644); err != nil {
		t.Fatalf("write agent: %v", err)
	}

	index, err := BuildPackageReferenceIndexFromRoots([]string{root})
	if err != nil {
		t.Fatalf("BuildPackageReferenceIndexFromRoots() error = %v", err)
	}

	if !index.Exists(resource.Command, "team/deploy") {
		t.Fatalf("expected nested command to be indexed")
	}
	if !index.Exists(resource.Skill, "skill-a") {
		t.Fatalf("expected skill to be indexed")
	}
	if !index.Exists(resource.Agent, "reviewer") {
		t.Fatalf("expected agent to be indexed")
	}
}

func TestValidatePackageReferences_SuggestsCanonicalIDs(t *testing.T) {
	index := NewPackageReferenceIndex()
	index.Add(resource.Command, "team/deploy")
	index.Add(resource.Skill, "python-helpers")

	pkg := &resource.Package{
		Name:        "test-pkg",
		Description: "test",
		Resources: []string{
			"command/team/deployy",
			"skill/python-helpres",
			"invalid",
		},
	}

	issues := ValidatePackageReferences(pkg, index)
	if len(issues) != 3 {
		t.Fatalf("ValidatePackageReferences() returned %d issues, want 3", len(issues))
	}

	if issues[0].Reference != "command/team/deployy" {
		t.Fatalf("unexpected first issue reference: %q", issues[0].Reference)
	}
	if !strings.Contains(issues[0].Suggestion, "command/team/deploy") {
		t.Fatalf("expected command suggestion, got %q", issues[0].Suggestion)
	}

	if issues[1].Reference != "skill/python-helpres" {
		t.Fatalf("unexpected second issue reference: %q", issues[1].Reference)
	}
	if !strings.Contains(issues[1].Suggestion, "skill/python-helpers") {
		t.Fatalf("expected skill suggestion, got %q", issues[1].Suggestion)
	}

	if issues[2].Reference != "invalid" {
		t.Fatalf("expected invalid reference passthrough, got %q", issues[2].Reference)
	}
	if issues[2].Suggestion != "" {
		t.Fatalf("expected no suggestion for invalid format, got %q", issues[2].Suggestion)
	}
}
