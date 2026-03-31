package repolock

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestLockPaths(t *testing.T) {
	repoPath := filepath.Join(t.TempDir(), "repo")
	m := NewManager(repoPath)

	if got, want := m.RepoLockPath(), filepath.Join(repoPath, ".workspace", "locks", "repo.lock"); got != want {
		t.Fatalf("RepoLockPath() = %q, want %q", got, want)
	}

	if got, want := m.WorkspaceMetadataLockPath(), filepath.Join(repoPath, ".workspace", "locks", "workspace-metadata.lock"); got != want {
		t.Fatalf("WorkspaceMetadataLockPath() = %q, want %q", got, want)
	}

	if got, want := m.CacheLockPath("abc123"), filepath.Join(repoPath, ".workspace", "locks", "caches", "abc123.lock"); got != want {
		t.Fatalf("CacheLockPath() = %q, want %q", got, want)
	}
}

func TestAcquireCreatesLockDirectoryBeforeRepoInit(t *testing.T) {
	repoPath := filepath.Join(t.TempDir(), "repo")
	m := NewManager(repoPath)

	lock, err := m.AcquireRepoLock(context.Background())
	if err != nil {
		t.Fatalf("AcquireRepoLock() failed: %v", err)
	}
	t.Cleanup(func() {
		_ = lock.Unlock()
	})

	if _, err := os.Stat(filepath.Join(repoPath, ".workspace", "locks")); err != nil {
		t.Fatalf("lock directory not created: %v", err)
	}
}

func TestAcquireRepoReadSharedSharedBehavior(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repo", ".workspace", "locks", "repo.lock")

	cmd := startLockHelperWithMode(t, path, "shared")
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	lock, ok, err := TryAcquireShared(path)
	if err != nil {
		t.Fatalf("TryAcquireShared() returned error: %v", err)
	}

	if runtime.GOOS == "windows" {
		if ok {
			_ = lock.Unlock()
			t.Fatalf("TryAcquireShared() unexpectedly acquired lock on windows exclusive-only fallback")
		}
		return
	}

	if !ok {
		t.Fatalf("TryAcquireShared() expected to acquire while another shared reader holds lock")
	}
	if err := lock.Unlock(); err != nil {
		t.Fatalf("Unlock() failed: %v", err)
	}
}

func TestAcquireSharedTimesOutWhileExclusiveHeld(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repo", ".workspace", "locks", "repo.lock")

	cmd := startLockHelperWithMode(t, path, "exclusive")
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	_, err := AcquireShared(context.Background(), path, 120*time.Millisecond)
	if err == nil {
		t.Fatalf("AcquireShared() expected timeout error")
	}
	var timeoutErr *AcquireTimeoutError
	if !errors.As(err, &timeoutErr) {
		t.Fatalf("AcquireShared() expected AcquireTimeoutError, got: %v", err)
	}
}

func TestAcquireExclusiveTimesOutWhileSharedHeld(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repo", ".workspace", "locks", "repo.lock")

	cmd := startLockHelperWithMode(t, path, "shared")
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	_, err := AcquireExclusive(context.Background(), path, 120*time.Millisecond)
	if err == nil {
		t.Fatalf("AcquireExclusive() expected timeout error")
	}
	var timeoutErr *AcquireTimeoutError
	if !errors.As(err, &timeoutErr) {
		t.Fatalf("AcquireExclusive() expected AcquireTimeoutError, got: %v", err)
	}
}

func TestTryAcquireAndTimeout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repo", ".workspace", "locks", "repo.lock")

	cmd := startLockHelper(t, path)
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	if _, ok, err := TryAcquire(path); err != nil {
		t.Fatalf("TryAcquire() returned error: %v", err)
	} else if ok {
		t.Fatalf("TryAcquire() unexpectedly acquired held lock")
	}

	_, err := Acquire(context.Background(), path, 120*time.Millisecond)
	if err == nil {
		t.Fatalf("Acquire() expected timeout error")
	}
	var timeoutErr *AcquireTimeoutError
	if !errors.As(err, &timeoutErr) {
		t.Fatalf("Acquire() expected AcquireTimeoutError, got: %v", err)
	}
}

func TestAcquireCancellation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repo", ".workspace", "locks", "repo.lock")

	cmd := startLockHelper(t, path)
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()

	_, err := Acquire(ctx, path, time.Second)
	if err == nil {
		t.Fatalf("Acquire() expected cancellation error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Acquire() expected context deadline error, got %v", err)
	}
}

func TestLockNonReentrant(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repo", ".workspace", "locks", "repo.lock")

	lock, err := Acquire(context.Background(), path, time.Second)
	if err != nil {
		t.Fatalf("Acquire() failed: %v", err)
	}
	t.Cleanup(func() {
		_ = lock.Unlock()
	})

	_, _, err = TryAcquire(path)
	if err == nil {
		t.Fatalf("expected non-reentrant error")
	}
	if !errors.Is(err, ErrNonReentrantLock) {
		t.Fatalf("expected ErrNonReentrantLock, got %v", err)
	}
}

func TestLockReadWriteTransitionsNonReentrant(t *testing.T) {
	t.Run("read to write transition", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "repo", ".workspace", "locks", "repo.lock")

		readLock, err := AcquireShared(context.Background(), path, time.Second)
		if err != nil {
			t.Fatalf("AcquireShared() failed: %v", err)
		}
		t.Cleanup(func() {
			_ = readLock.Unlock()
		})

		_, err = AcquireExclusive(context.Background(), path, 100*time.Millisecond)
		if err == nil {
			t.Fatalf("expected read->write transition to fail")
		}
		if !errors.Is(err, ErrNonReentrantLock) {
			t.Fatalf("expected ErrNonReentrantLock, got %v", err)
		}
	})

	t.Run("write to read transition", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "repo", ".workspace", "locks", "repo.lock")

		writeLock, err := AcquireExclusive(context.Background(), path, time.Second)
		if err != nil {
			t.Fatalf("AcquireExclusive() failed: %v", err)
		}
		t.Cleanup(func() {
			_ = writeLock.Unlock()
		})

		_, err = AcquireShared(context.Background(), path, 100*time.Millisecond)
		if err == nil {
			t.Fatalf("expected write->read transition to fail")
		}
		if !errors.Is(err, ErrNonReentrantLock) {
			t.Fatalf("expected ErrNonReentrantLock, got %v", err)
		}
	})
}

func TestLockReleasedAfterSubprocessKill(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repo", ".workspace", "locks", "repo.lock")

	cmd := startLockHelper(t, path)

	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("failed to kill helper process: %v", err)
	}
	_ = cmd.Wait()

	lock, err := Acquire(context.Background(), path, time.Second)
	if err != nil {
		t.Fatalf("failed to acquire lock after helper kill: %v", err)
	}
	if err := lock.Unlock(); err != nil {
		t.Fatalf("failed to unlock lock: %v", err)
	}
}

func TestHelperAcquireLockAndWait(t *testing.T) {
	if os.Getenv("AIMGR_TEST_HELPER_MODE") != "acquire-wait" {
		return
	}

	path := os.Getenv("AIMGR_TEST_HELPER_LOCK_PATH")
	if path == "" {
		os.Exit(2)
	}
	readyPath := os.Getenv("AIMGR_TEST_HELPER_READY_PATH")
	if readyPath == "" {
		os.Exit(4)
	}

	lockMode := os.Getenv("AIMGR_TEST_HELPER_LOCK_MODE")
	var (
		lock *Lock
		err  error
	)
	if lockMode == "shared" {
		lock, err = AcquireShared(context.Background(), path, time.Second)
	} else {
		lock, err = AcquireExclusive(context.Background(), path, time.Second)
	}
	if err != nil {
		os.Exit(3)
	}
	// #nosec G703 -- readyPath is a test-only marker path controlled by this test process.
	if err := os.WriteFile(readyPath, []byte("ready"), 0644); err != nil {
		os.Exit(5)
	}

	defer func() {
		_ = lock.Unlock()
	}()
	for {
		time.Sleep(time.Second)
	}
}

func startLockHelper(t *testing.T, lockPath string) *exec.Cmd {
	t.Helper()
	return startLockHelperWithMode(t, lockPath, "exclusive")
}

func startLockHelperWithMode(t *testing.T, lockPath, lockMode string) *exec.Cmd {
	t.Helper()

	readyPath := filepath.Join(t.TempDir(), "ready")
	// #nosec G702 -- os.Args[0] is the current test binary path.
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperAcquireLockAndWait")
	cmd.Env = append(
		os.Environ(),
		"AIMGR_TEST_HELPER_LOCK_PATH="+lockPath,
		"AIMGR_TEST_HELPER_READY_PATH="+readyPath,
		"AIMGR_TEST_HELPER_LOCK_MODE="+lockMode,
		"AIMGR_TEST_HELPER_MODE=acquire-wait",
	)

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start helper process: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, err := os.Stat(readyPath); err == nil {
			return cmd
		}
		if time.Now().After(deadline) {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			t.Fatalf("helper did not signal readiness")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
