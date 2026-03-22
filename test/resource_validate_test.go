package test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dynatrace-oss/ai-config-manager/v3/pkg/repo"
	"gopkg.in/yaml.v3"
)

func TestCLIResourceValidate_PathStandaloneCommand(t *testing.T) {
	setupTestEnvironment(t)

	cmdPath := filepath.Join(t.TempDir(), "standalone.md")
	if err := os.WriteFile(cmdPath, []byte("---\ndescription: standalone command\n---\n# Cmd\n"), 0644); err != nil {
		t.Fatalf("write command: %v", err)
	}

	output, err := runAimgr(t, "resource", "validate", cmdPath)
	if err != nil {
		t.Fatalf("validate failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "valid") {
		t.Fatalf("expected valid output, got: %s", output)
	}
	if !strings.Contains(output, "command") {
		t.Fatalf("expected command type in output, got: %s", output)
	}
}

func TestCLIResourceValidate_CanonicalIDWithSourceRoot_JSON(t *testing.T) {
	setupTestEnvironment(t)

	sourceRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(sourceRoot, "skills", "demo-skill"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceRoot, "skills", "demo-skill", "SKILL.md"), []byte("---\nname: demo-skill\ndescription: demo\n---\n# Skill\n"), 0644); err != nil {
		t.Fatalf("write skill: %v", err)
	}

	output, err := runAimgr(t, "resource", "validate", "--format=json", "--source-root", sourceRoot, "skill/demo-skill")
	if err != nil {
		t.Fatalf("validate failed: %v\nOutput: %s", err, output)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("parse json output: %v\nOutput: %s", err, output)
	}

	if parsed["valid"] != true {
		t.Fatalf("expected valid=true, got: %v", parsed["valid"])
	}
	if parsed["resource_type"] != "skill" {
		t.Fatalf("expected resource_type=skill, got: %v", parsed["resource_type"])
	}
	if parsed["resolved_id"] != "skill/demo-skill" {
		t.Fatalf("expected resolved_id, got: %v", parsed["resolved_id"])
	}
}

func TestCLIResourceValidate_OutputYAML(t *testing.T) {
	setupTestEnvironment(t)

	agentPath := filepath.Join(t.TempDir(), "agent.md")
	if err := os.WriteFile(agentPath, []byte("---\ndescription: agent\n---\n# Agent\n"), 0644); err != nil {
		t.Fatalf("write agent: %v", err)
	}

	output, err := runAimgr(t, "resource", "validate", "--format=yaml", agentPath)
	if err != nil {
		t.Fatalf("validate failed: %v\nOutput: %s", err, output)
	}

	var parsed map[string]interface{}
	if err := yaml.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("parse yaml output: %v\nOutput: %s", err, output)
	}

	if parsed["valid"] != true {
		t.Fatalf("expected valid=true, got: %v", parsed["valid"])
	}
}

func TestCLIResourceValidate_ExitCodeValidationError(t *testing.T) {
	setupTestEnvironment(t)

	invalidSkill := filepath.Join(t.TempDir(), "bad-skill")
	if err := os.MkdirAll(invalidSkill, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(invalidSkill, "SKILL.md"), []byte("---\nname: bad-skill\n---\n# Skill\n"), 0644); err != nil {
		t.Fatalf("write skill: %v", err)
	}

	_, err := runAimgr(t, "resource", "validate", invalidSkill)
	if err == nil {
		t.Fatalf("expected non-zero exit")
	}

	if code := commandExitCode(err); code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
}

func TestCLIResourceValidate_ExitCodeUsageError(t *testing.T) {
	repoPath := filepath.Join(t.TempDir(), "missing-repo")
	t.Setenv("AIMGR_REPO_PATH", repoPath)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	_, err := runAimgr(t, "resource", "validate", "skill/without-context")
	if err == nil {
		t.Fatalf("expected non-zero exit")
	}

	if code := commandExitCode(err); code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
}

func TestCLIResourceValidate_DoesNotMutateRepoPathForPathValidation(t *testing.T) {
	repoPath := filepath.Join(t.TempDir(), "uncreated-repo")
	t.Setenv("AIMGR_REPO_PATH", repoPath)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	commandPath := filepath.Join(t.TempDir(), "cmd.md")
	if err := os.WriteFile(commandPath, []byte("---\ndescription: standalone\n---\n# Command\n"), 0644); err != nil {
		t.Fatalf("write command: %v", err)
	}

	if _, err := runAimgr(t, "resource", "validate", commandPath); err != nil {
		t.Fatalf("validate failed: %v", err)
	}

	if _, err := os.Stat(repoPath); !os.IsNotExist(err) {
		t.Fatalf("expected repo path to remain absent, got err=%v", err)
	}
}

func TestCLIResourceValidate_PackagePathWithSourceRoot_JSON(t *testing.T) {
	setupTestEnvironment(t)

	root := t.TempDir()
	for _, dir := range []string{"commands/team", "skills/helper", "agents", "packages"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	if err := os.WriteFile(filepath.Join(root, "commands", "team", "deploy.md"), []byte("---\ndescription: deploy\n---\n# Deploy\n"), 0644); err != nil {
		t.Fatalf("write command: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "skills", "helper", "SKILL.md"), []byte("---\nname: helper\ndescription: helper\n---\n# Skill\n"), 0644); err != nil {
		t.Fatalf("write skill: %v", err)
	}

	pkgPath := filepath.Join(root, "packages", "team.package.json")
	pkgJSON := `{"name":"team-pkg","description":"team","resources":["command/team/deploy","skill/helper"]}`
	if err := os.WriteFile(pkgPath, []byte(pkgJSON), 0644); err != nil {
		t.Fatalf("write package: %v", err)
	}

	output, err := runAimgr(t, "resource", "validate", "--format=json", "--source-root", root, pkgPath)
	if err != nil {
		t.Fatalf("validate failed: %v\nOutput: %s", err, output)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("parse json output: %v\nOutput: %s", err, output)
	}

	if parsed["resource_type"] != "package" {
		t.Fatalf("expected resource_type=package, got: %v", parsed["resource_type"])
	}
	if parsed["valid"] != true {
		t.Fatalf("expected valid=true, got: %v", parsed["valid"])
	}
}

func TestCLIResourceValidate_PackageMissingRefExitCodeAndDiagnostic(t *testing.T) {
	setupTestEnvironment(t)

	root := t.TempDir()
	for _, dir := range []string{"commands/team", "skills", "agents", "packages"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "commands", "team", "deploy.md"), []byte("---\ndescription: deploy\n---\n# Deploy\n"), 0644); err != nil {
		t.Fatalf("write command: %v", err)
	}

	pkgPath := filepath.Join(root, "packages", "team.package.json")
	pkgJSON := `{"name":"team-pkg","description":"team","resources":["command/team/deployy"]}`
	if err := os.WriteFile(pkgPath, []byte(pkgJSON), 0644); err != nil {
		t.Fatalf("write package: %v", err)
	}

	output, err := runAimgr(t, "resource", "validate", "--format=json", "--source-root", root, pkgPath)
	if err == nil {
		t.Fatalf("expected non-zero exit for missing refs")
	}
	if code := commandExitCode(err); code != 1 {
		t.Fatalf("expected exit code 1, got %d\nOutput: %s", code, output)
	}
	if !strings.Contains(output, "missing_package_ref") {
		t.Fatalf("expected missing_package_ref diagnostic, got: %s", output)
	}
	if !strings.Contains(output, "command/team/deploy") {
		t.Fatalf("expected canonical suggestion in output, got: %s", output)
	}
}

func TestCLIResourceValidate_PackageValidationDoesNotMutateLiveRepo(t *testing.T) {
	setupTestEnvironment(t)

	liveRepo := t.TempDir()
	liveManager := repo.NewManagerWithPath(liveRepo)
	if err := liveManager.Init(); err != nil {
		t.Fatalf("init live repo: %v", err)
	}

	manifestPath := filepath.Join(liveRepo, "ai.repo.yaml")
	before, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest before: %v", err)
	}

	t.Setenv("AIMGR_REPO_PATH", liveRepo)

	validationRoot := t.TempDir()
	for _, dir := range []string{"commands/team", "skills", "agents", "packages"} {
		if err := os.MkdirAll(filepath.Join(validationRoot, dir), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(validationRoot, "commands", "team", "deploy.md"), []byte("---\ndescription: deploy\n---\n# Deploy\n"), 0644); err != nil {
		t.Fatalf("write command: %v", err)
	}
	pkgPath := filepath.Join(validationRoot, "packages", "team.package.json")
	pkgJSON := `{"name":"team-pkg","description":"team","resources":["command/team/deploy"]}`
	if err := os.WriteFile(pkgPath, []byte(pkgJSON), 0644); err != nil {
		t.Fatalf("write package: %v", err)
	}

	if output, err := runAimgr(t, "resource", "validate", "--source-root", validationRoot, pkgPath); err != nil {
		t.Fatalf("validate failed: %v\nOutput: %s", err, output)
	}

	after, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest after: %v", err)
	}
	if string(before) != string(after) {
		t.Fatalf("expected live repo manifest unchanged")
	}
}

func TestCLIResourceValidate_MissingRepoPath_ManifestBackedOutputIsMachineReadable(t *testing.T) {
	missingRepo := filepath.Join(t.TempDir(), "missing-repo")
	t.Setenv("AIMGR_REPO_PATH", missingRepo)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	root := t.TempDir()
	for _, dir := range []string{"commands/team", "skills/helper", "agents", "packages"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	if err := os.WriteFile(filepath.Join(root, "commands", "team", "deploy.md"), []byte("---\ndescription: deploy\n---\n# Deploy\n"), 0644); err != nil {
		t.Fatalf("write command: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "skills", "helper", "SKILL.md"), []byte("---\nname: helper\ndescription: helper\n---\n# Skill\n"), 0644); err != nil {
		t.Fatalf("write skill: %v", err)
	}

	pkgPath := filepath.Join(root, "packages", "release-bundle.package.json")
	pkgJSON := `{"name":"release-bundle","description":"bundle","resources":["command/team/deploy","skill/helper"]}`
	if err := os.WriteFile(pkgPath, []byte(pkgJSON), 0644); err != nil {
		t.Fatalf("write package: %v", err)
	}

	manifestPath := filepath.Join(t.TempDir(), "ai.repo.yaml")
	manifestYAML := "version: 1\nsources:\n  - name: local\n    path: " + root + "\n"
	if err := os.WriteFile(manifestPath, []byte(manifestYAML), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	t.Run("json", func(t *testing.T) {
		output, err := runAimgr(t, "resource", "validate", "--format=json", "--repo-manifest", manifestPath, "package/release-bundle")
		if err != nil {
			t.Fatalf("validate failed: %v\nOutput: %s", err, output)
		}
		if strings.Contains(output, "Warning: failed to initialize logger") {
			t.Fatalf("unexpected warning prefix in machine-readable output: %s", output)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(output), &parsed); err != nil {
			t.Fatalf("parse json output: %v\nOutput: %s", err, output)
		}
		if parsed["valid"] != true {
			t.Fatalf("expected valid=true, got: %v", parsed["valid"])
		}
		if parsed["resource_type"] != "package" {
			t.Fatalf("expected resource_type=package, got: %v", parsed["resource_type"])
		}

		ctx, ok := parsed["context"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected context object, got: %T", parsed["context"])
		}
		if ctx["kind"] != "repo-manifest" {
			t.Fatalf("expected context kind repo-manifest, got: %v", ctx["kind"])
		}
	})

	t.Run("yaml", func(t *testing.T) {
		output, err := runAimgr(t, "resource", "validate", "--format=yaml", "--repo-manifest", manifestPath, "package/release-bundle")
		if err != nil {
			t.Fatalf("validate failed: %v\nOutput: %s", err, output)
		}
		if strings.Contains(output, "Warning: failed to initialize logger") {
			t.Fatalf("unexpected warning prefix in machine-readable output: %s", output)
		}

		var parsed map[string]interface{}
		if err := yaml.Unmarshal([]byte(output), &parsed); err != nil {
			t.Fatalf("parse yaml output: %v\nOutput: %s", err, output)
		}
		if parsed["valid"] != true {
			t.Fatalf("expected valid=true, got: %v", parsed["valid"])
		}
		if parsed["resource_type"] != "package" {
			t.Fatalf("expected resource_type=package, got: %v", parsed["resource_type"])
		}

		ctx, ok := parsed["context"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected context object, got: %T", parsed["context"])
		}
		if ctx["kind"] != "repo-manifest" {
			t.Fatalf("expected context kind repo-manifest, got: %v", ctx["kind"])
		}
	})
}

func TestCLIResourceValidate_MissingRepoPath_MissingContextOutputIsMachineReadable(t *testing.T) {
	missingRepo := filepath.Join(t.TempDir(), "missing-repo")
	t.Setenv("AIMGR_REPO_PATH", missingRepo)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	t.Run("json", func(t *testing.T) {
		output, err := runAimgr(t, "resource", "validate", "--format=json", "skill/without-context")
		if err == nil {
			t.Fatalf("expected non-zero exit")
		}
		if code := commandExitCode(err); code != 2 {
			t.Fatalf("expected exit code 2, got %d\nOutput: %s", code, output)
		}
		if strings.Contains(output, "Warning: failed to initialize logger") {
			t.Fatalf("unexpected warning prefix in machine-readable output: %s", output)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(output), &parsed); err != nil {
			t.Fatalf("parse json output: %v\nOutput: %s", err, output)
		}
		if parsed["valid"] != false {
			t.Fatalf("expected valid=false, got: %v", parsed["valid"])
		}
	})

	t.Run("yaml", func(t *testing.T) {
		output, err := runAimgr(t, "resource", "validate", "--format=yaml", "skill/without-context")
		if err == nil {
			t.Fatalf("expected non-zero exit")
		}
		if code := commandExitCode(err); code != 2 {
			t.Fatalf("expected exit code 2, got %d\nOutput: %s", code, output)
		}
		if strings.Contains(output, "Warning: failed to initialize logger") {
			t.Fatalf("unexpected warning prefix in machine-readable output: %s", output)
		}

		var parsed map[string]interface{}
		if err := yaml.Unmarshal([]byte(output), &parsed); err != nil {
			t.Fatalf("parse yaml output: %v\nOutput: %s", err, output)
		}
		if parsed["valid"] != false {
			t.Fatalf("expected valid=false, got: %v", parsed["valid"])
		}
	})
}

func commandExitCode(err error) int {
	if err == nil {
		return 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}
	return -1
}
