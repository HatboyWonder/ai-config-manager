package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dynatrace-oss/ai-config-manager/v3/pkg/giturl"
	"github.com/dynatrace-oss/ai-config-manager/v3/pkg/repomanifest"
	"github.com/dynatrace-oss/ai-config-manager/v3/pkg/workspace"
)

func TestIsGitSource(t *testing.T) {
	tests := []struct {
		sourceType string
		expected   bool
	}{
		{"github", true},
		{"git-url", true},
		{"gitlab", true},
		{"local", false},
		{"file", false},
		{"", false},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.sourceType, func(t *testing.T) {
			result := isGitSource(tt.sourceType)
			if result != tt.expected {
				t.Errorf("isGitSource(%q) = %v, expected %v", tt.sourceType, result, tt.expected)
			}
		})
	}
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://github.com/test/repo", "https://github.com/test/repo"},
		{"HTTPS://GitHub.com/Test/Repo", "https://github.com/test/repo"},
		{"https://github.com/test/repo.git", "https://github.com/test/repo"},
		{"https://github.com/test/repo/", "https://github.com/test/repo"},
		{"https://github.com/test/repo.git/", "https://github.com/test/repo"},
		{"https://github.com/test/repo/.git", "https://github.com/test/repo"},
		{"https://github.com/test/repo///", "https://github.com/test/repo"},
		{"  https://github.com/test/repo  ", "https://github.com/test/repo"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeURL(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeURL(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeURL_ConsistencyWithCanonicalHelper(t *testing.T) {
	inputs := []string{
		"https://github.com/test/repo",
		"HTTPS://GitHub.com/Test/Repo",
		"https://github.com/test/repo/",
		"https://github.com/test/repo.git",
		"https://github.com/test/repo.git/",
		"https://github.com/test/repo/.git",
		"https://github.com/test/repo///",
		"  https://github.com/test/repo  ",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			got := normalizeURL(input)
			want := giturl.NormalizeURL(input)
			if got != want {
				t.Fatalf("normalizeURL(%q) = %q, want canonical %q", input, got, want)
			}
		})
	}
}

func TestURLNormalization_ConsistencyAcrossWorkspacePruneAndSourceID(t *testing.T) {
	input := "  HTTPS://github.com/Test/Repo.git/  "

	normalized := normalizeURL(input)
	workspaceHash := workspace.ComputeHash(input)
	canonicalHash := workspace.ComputeHash(normalized)
	if workspaceHash != canonicalHash {
		t.Fatalf("workspace hash mismatch: %s != %s", workspaceHash, canonicalHash)
	}

	canonicalSourceID := repomanifest.GenerateSourceID(&repomanifest.Source{URL: normalized})
	inputSourceID := repomanifest.GenerateSourceID(&repomanifest.Source{URL: input})
	if canonicalSourceID != inputSourceID {
		t.Fatalf("source ID mismatch for equivalent URLs: %s != %s", canonicalSourceID, inputSourceID)
	}

	if normalized != giturl.NormalizeURL(input) {
		t.Fatalf("prune normalization mismatch with canonical helper")
	}
}

func TestGetDirSize(t *testing.T) {
	// Create temp directory with some files
	tempDir := t.TempDir()

	// Create test files
	file1 := filepath.Join(tempDir, "file1.txt")
	file2 := filepath.Join(tempDir, "file2.txt")
	subDir := filepath.Join(tempDir, "subdir")
	file3 := filepath.Join(subDir, "file3.txt")

	if err := os.WriteFile(file1, []byte("12345"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, []byte("1234567890"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file3, []byte("123"), 0644); err != nil {
		t.Fatal(err)
	}

	// Calculate size
	size, err := getDirSize(tempDir)
	if err != nil {
		t.Fatalf("getDirSize failed: %v", err)
	}

	// Expected: 5 + 10 + 3 = 18 bytes
	expected := int64(18)
	if size != expected {
		t.Errorf("Expected size %d, got %d", expected, size)
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{100, "100 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatSize(tt.bytes)
			if result != tt.expected {
				t.Errorf("formatSize(%d) = %q, expected %q", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestRepositoryPlural(t *testing.T) {
	tests := []struct {
		count    int
		expected string
	}{
		{0, "repositories"},
		{1, "repository"},
		{2, "repositories"},
		{100, "repositories"},
	}

	for _, tt := range tests {
		result := repositoryPlural(tt.count)
		if result != tt.expected {
			t.Errorf("repositoryPlural(%d) = %q, expected %q", tt.count, result, tt.expected)
		}
	}
}
