package repo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func (m *Manager) isGitRepo() bool {
	gitDir := filepath.Join(m.repoPath, ".git")
	info, err := os.Stat(gitDir)
	return err == nil && info.IsDir()
}

// CommitChanges commits already-staged changes with a message.
// Returns nil if successful, or an error if the commit fails.
// If not a git repo, returns nil (non-fatal - operations work without git).
//
// Deprecated for mutating operations: prefer CommitChangesForPaths so callers
// can scope staging to intended files only.
func (m *Manager) CommitChanges(message string) error {
	if !m.isGitRepo() {
		// Not a git repo - this is not an error, just skip
		return nil
	}

	// Legacy behavior: stage all tracked/untracked changes.
	addCmd := exec.Command("git", "add", "-A", ".")
	addCmd.Dir = m.repoPath
	if output, err := addCmd.CombinedOutput(); err != nil {
		if m.logger != nil {
			m.logger.Error("failed to stage changes",
				"path", m.repoPath,
				"error", err.Error(),
				"output", string(output),
			)
		}
		return fmt.Errorf("failed to stage changes: %w\nOutput: %s", err, output)
	}

	// Commit all currently staged changes.
	return m.commitStagedChanges(message, nil)
}

// CommitChangesForPaths stages the provided paths and commits only staged
// changes with the given message.
//
// Paths may be absolute or relative. Absolute paths must resolve under the
// repository root.
func (m *Manager) CommitChangesForPaths(message string, paths []string) error {
	if !m.isGitRepo() {
		// Not a git repo - this is not an error, just skip
		return nil
	}

	normalizedPaths, err := m.normalizeCommitPaths(paths)
	if err != nil {
		return err
	}

	if len(normalizedPaths) > 0 {
		if err := m.stageScopedPaths(normalizedPaths); err != nil {
			return err
		}
	}

	return m.commitStagedChanges(message, normalizedPaths)
}

func (m *Manager) commitStagedChanges(message string, scopedPaths []string) error {
	// Check staged changes only, so unrelated working-tree changes are ignored.
	statusArgs := []string{"diff", "--cached", "--name-only"}
	if len(scopedPaths) > 0 {
		statusArgs = append(statusArgs, "--")
		statusArgs = append(statusArgs, scopedPaths...)
	}
	statusCmd := exec.Command("git", statusArgs...)
	statusCmd.Dir = m.repoPath
	output, err := statusCmd.CombinedOutput()
	if err != nil {
		if m.logger != nil {
			m.logger.Error("failed to check staged git changes",
				"path", m.repoPath,
				"error", err.Error(),
				"output", string(output),
			)
		}
		return fmt.Errorf("failed to check staged git changes: %w\nOutput: %s", err, output)
	}

	// If no staged changes, nothing to commit.
	stagedPaths := parseGitNameOnlyOutput(output)
	if len(stagedPaths) == 0 {
		return nil
	}

	// Create commit. When scoped paths are provided, use pathspec to avoid
	// committing unrelated pre-staged changes.
	commitArgs := []string{"commit", "-m", message}
	if len(scopedPaths) > 0 {
		commitArgs = append(commitArgs, "--")
		commitArgs = append(commitArgs, stagedPaths...)
	}
	commitCmd := exec.Command("git", commitArgs...)
	commitCmd.Dir = m.repoPath
	if output, err := commitCmd.CombinedOutput(); err != nil {
		if m.logger != nil {
			m.logger.Error("failed to commit changes",
				"path", m.repoPath,
				"message", message,
				"error", err.Error(),
				"output", string(output),
			)
		}
		return fmt.Errorf("failed to commit changes: %w\nOutput: %s", err, output)
	}

	return nil
}

func (m *Manager) stageScopedPaths(paths []string) error {
	for _, path := range paths {
		addCmd := exec.Command("git", "add", "-A", "--", path)
		addCmd.Dir = m.repoPath
		output, err := addCmd.CombinedOutput()
		if err == nil {
			continue
		}

		outputText := string(output)
		if strings.Contains(outputText, "did not match any files") {
			continue
		}

		if m.logger != nil {
			m.logger.Error("failed to stage scoped change",
				"path", m.repoPath,
				"target", path,
				"error", err.Error(),
				"output", outputText,
			)
		}

		return fmt.Errorf("failed to stage scoped change %q: %w\nOutput: %s", path, err, output)
	}

	return nil
}

func (m *Manager) normalizeCommitPaths(paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, nil
	}

	seen := make(map[string]struct{}, len(paths))
	normalized := make([]string, 0, len(paths))

	for _, p := range paths {
		if strings.TrimSpace(p) == "" {
			continue
		}

		candidate := p
		if filepath.IsAbs(candidate) {
			rel, err := filepath.Rel(m.repoPath, candidate)
			if err != nil {
				return nil, fmt.Errorf("failed to normalize commit path %q: %w", p, err)
			}
			candidate = rel
		}

		candidate = filepath.Clean(candidate)
		if candidate == "." {
			continue
		}

		if strings.HasPrefix(candidate, "..") {
			return nil, fmt.Errorf("commit path %q escapes repository root", p)
		}

		candidate = filepath.ToSlash(candidate)
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		normalized = append(normalized, candidate)
	}

	sort.Strings(normalized)
	return normalized, nil
}

func parseGitNameOnlyOutput(output []byte) []string {
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	paths := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		paths = append(paths, line)
	}

	return paths
}

// copyFile copies a single file from src to dst
