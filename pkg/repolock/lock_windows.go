//go:build windows

package repolock

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows"
)

// Lock represents a held OS-backed file lock on Windows.
//
// Uses LockFileEx/UnlockFileEx for exclusive file locking. Locks are
// automatically released by the OS if the process exits, including
// crash/kill scenarios.
type Lock struct {
	path string
	file *os.File
}

// Acquire acquires an exclusive lock for the provided path.
//
// Semantics:
//   - Blocking acquisition with periodic retries.
//   - Honors context cancellation.
//   - Returns AcquireTimeoutError if timeout expires before lock acquisition.
//   - Returns ErrNonReentrantLock if this process already holds the same lock.
func Acquire(ctx context.Context, path string, timeout time.Duration) (*Lock, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if err := tracker.claim(path); err != nil {
		return nil, err
	}

	// #nosec G703 -- lock file path is an internal repository lock path controlled by callers.
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		tracker.release(path)
		return nil, fmt.Errorf("failed to create lock directory: %w", err)
	}

	// #nosec G703 -- lock file path is an internal repository lock path controlled by callers.
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		tracker.release(path)
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}

	cleanup := func() {
		_ = file.Close()
		tracker.release(path)
	}

	deadline := time.Time{}
	if timeout > 0 {
		deadline = time.Now().Add(timeout)
	}

	for {
		acquired, lockErr := tryLockFileEx(file)
		if lockErr != nil {
			cleanup()
			return nil, fmt.Errorf("failed to acquire lock: %w", lockErr)
		}
		if acquired {
			return &Lock{path: path, file: file}, nil
		}

		if !deadline.IsZero() && time.Now().After(deadline) {
			cleanup()
			return nil, &AcquireTimeoutError{Path: path, Timeout: timeout}
		}

		select {
		case <-ctx.Done():
			cleanup()
			return nil, fmt.Errorf("lock acquisition canceled: %w", ctx.Err())
		case <-time.After(defaultPollInterval):
		}
	}
}

// TryAcquire attempts a non-blocking lock acquisition.
// Returns acquired=false when another process already holds the lock.
func TryAcquire(path string) (*Lock, bool, error) {
	if err := tracker.claim(path); err != nil {
		return nil, false, err
	}

	// #nosec G703 -- lock file path is an internal repository lock path controlled by callers.
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		tracker.release(path)
		return nil, false, fmt.Errorf("failed to create lock directory: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		tracker.release(path)
		return nil, false, fmt.Errorf("failed to open lock file: %w", err)
	}

	acquired, lockErr := tryLockFileEx(file)
	if lockErr != nil {
		_ = file.Close()
		tracker.release(path)
		return nil, false, fmt.Errorf("failed to acquire lock: %w", lockErr)
	}

	if !acquired {
		_ = file.Close()
		tracker.release(path)
		return nil, false, nil
	}

	return &Lock{path: path, file: file}, true, nil
}

// Unlock releases the lock.
func (l *Lock) Unlock() error {
	if l == nil || l.file == nil {
		return nil
	}

	defer tracker.release(l.path)

	handle := windows.Handle(l.file.Fd())
	var ol windows.Overlapped
	err := windows.UnlockFileEx(handle, 0, 1, 0, &ol)
	closeErr := l.file.Close()
	l.file = nil

	if err != nil {
		return fmt.Errorf("failed to unlock file: %w", err)
	}
	if closeErr != nil {
		return fmt.Errorf("failed to close lock file: %w", closeErr)
	}

	return nil
}

// tryLockFileEx attempts a non-blocking exclusive lock using LockFileEx.
// Returns (true, nil) on success, (false, nil) if already locked, or (false, err) on error.
func tryLockFileEx(file *os.File) (bool, error) {
	handle := windows.Handle(file.Fd())
	const lockfileExclusiveLock = 0x00000002
	const lockfileFailImmediately = 0x00000001

	var ol windows.Overlapped
	// LockFileEx with LOCKFILE_EXCLUSIVE_LOCK | LOCKFILE_FAIL_IMMEDIATELY
	err := windows.LockFileEx(handle, lockfileExclusiveLock|lockfileFailImmediately, 0, 1, 0, &ol)
	if err == nil {
		return true, nil
	}

	// ERROR_LOCK_VIOLATION means another process holds the lock — not an error, just contention.
	if err == windows.ERROR_LOCK_VIOLATION {
		return false, nil
	}

	// windows.Errno wrapping: check for the numeric code directly
	if windowsErr, ok := err.(windows.Errno); ok {
		// 33 = ERROR_LOCK_VIOLATION
		if uintptr(windowsErr) == 33 {
			return false, nil
		}
	}

	return false, err
}
