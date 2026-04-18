package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseCleanFormat(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{name: "default table", raw: ""},
		{name: "table", raw: "table"},
		{name: "json", raw: "json"},
		{name: "yaml rejected", raw: "yaml", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseCleanFormat(tt.raw)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestSummarizeCleanResult_CountsByType(t *testing.T) {
	projectDir := t.TempDir()
	ownedPath := filepath.Join(projectDir, ".claude", "skills")
	if err := os.MkdirAll(ownedPath, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	summary := summarizeCleanResult(
		[]OwnedResourceDir{{Path: ownedPath}, {Path: filepath.Join(projectDir, ".claude", "agents")}},
		[]CleanRemovedEntry{
			{EntryType: "file"},
			{EntryType: "symlink"},
			{EntryType: "directory"},
		},
		[]CleanFailedEntry{{}, {}},
	)

	if summary.OwnedDirsDetected != 2 || summary.OwnedDirsExisting != 1 {
		t.Fatalf("unexpected owned dir counts: %+v", summary)
	}
	if summary.Removed != 3 || summary.RemovedFiles != 1 || summary.RemovedSymlinks != 1 || summary.RemovedDirs != 1 {
		t.Fatalf("unexpected removed counts: %+v", summary)
	}
	if summary.Failed != 2 {
		t.Fatalf("unexpected failed count: %+v", summary)
	}
}

func TestDisplayCleanResult_JSONIncludesDetails(t *testing.T) {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	result := CleanResult{
		Warnings: []string{"warn"},
		Removed:  []CleanRemovedEntry{{Tool: "claude", ResourceType: "skill", Path: "/tmp/p", EntryType: "symlink"}},
		Failed:   []CleanFailedEntry{{Tool: "claude", ResourceType: "skill", Path: "/tmp/f", EntryType: "file", Error: "boom"}},
		Summary:  CleanSummary{Removed: 1, Failed: 1},
	}

	err = displayCleanResult(result, "json")
	_ = w.Close()
	os.Stdout = old
	if err != nil {
		t.Fatalf("display json failed: %v", err)
	}

	buf := make([]byte, 8192)
	n, _ := r.Read(buf)

	var parsed CleanResult
	if err := json.Unmarshal(buf[:n], &parsed); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, string(buf[:n]))
	}
	if len(parsed.Removed) != 1 || len(parsed.Failed) != 1 || parsed.Summary.Removed != 1 {
		t.Fatalf("unexpected parsed result: %+v", parsed)
	}
}

func TestCollectCleanWarnings_MissingManifest(t *testing.T) {
	projectDir := t.TempDir()
	warnings := collectCleanWarnings(projectDir)
	if len(warnings) != 1 {
		t.Fatalf("expected one warning, got %d", len(warnings))
	}
	if !strings.Contains(warnings[0], "ai.package.yaml") || !strings.Contains(warnings[0], "will not be able to restore") {
		t.Fatalf("unexpected warning: %s", warnings[0])
	}
}

func TestCleanCommand_HelpDoesNotExposeYesFlag(t *testing.T) {
	if cleanCmd.Flags().Lookup("yes") != nil {
		t.Fatalf("clean command should not expose --yes")
	}
}
