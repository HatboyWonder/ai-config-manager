package cmd

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dynatrace-oss/ai-config-manager/v3/pkg/repolock"
)

func TestGetCommandExitCode(t *testing.T) {
	if got := getCommandExitCode(nil); got != 0 {
		t.Fatalf("expected exit code 0 for nil error, got %d", got)
	}

	if got := getCommandExitCode(newCompletedWithFindingsError("findings")); got != 1 {
		t.Fatalf("expected exit code 1 for completed findings, got %d", got)
	}

	if got := getCommandExitCode(newOperationalFailureError(errors.New("boom"))); got != 2 {
		t.Fatalf("expected exit code 2 for operational failure, got %d", got)
	}
}

func TestClassifyOperationalError(t *testing.T) {
	t.Run("lock timeout maps to repo_busy", func(t *testing.T) {
		err := &repolock.AcquireTimeoutError{Path: "/tmp/repo.lock", Timeout: time.Second}
		if got := classifyOperationalError(err); got != commandErrorCategoryRepoBusy {
			t.Fatalf("expected repo_busy, got %q", got)
		}
	})

	t.Run("context cancellation maps to repo_busy", func(t *testing.T) {
		if got := classifyOperationalError(context.Canceled); got != commandErrorCategoryRepoBusy {
			t.Fatalf("expected repo_busy, got %q", got)
		}
	})

	t.Run("path error maps to io_error", func(t *testing.T) {
		err := &fs.PathError{Op: "open", Path: filepath.Join("missing", "file"), Err: os.ErrNotExist}
		if got := classifyOperationalError(err); got != commandErrorCategoryIOError {
			t.Fatalf("expected io_error, got %q", got)
		}
	})
}
