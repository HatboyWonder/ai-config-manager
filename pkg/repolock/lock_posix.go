//go:build unix

package repolock

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/unix"
)

// Lock represents a held OS-backed file lock.
//
// On supported POSIX systems this uses flock(2) against an open file
// descriptor. Locks are automatically released by the OS if the process exits,
// including crash/kill scenarios.
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

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		tracker.release(path)
		return nil, fmt.Errorf("failed to create lock directory: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
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
		err = unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
		if err == nil {
			return &Lock{path: path, file: file}, nil
		}

		if !errors.Is(err, unix.EWOULDBLOCK) && !errors.Is(err, unix.EAGAIN) {
			cleanup()
			return nil, fmt.Errorf("failed to acquire lock: %w", err)
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

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		tracker.release(path)
		return nil, false, fmt.Errorf("failed to create lock directory: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		tracker.release(path)
		return nil, false, fmt.Errorf("failed to open lock file: %w", err)
	}

	err = unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if err == nil {
		return &Lock{path: path, file: file}, true, nil
	}

	_ = file.Close()
	tracker.release(path)

	if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
		return nil, false, nil
	}

	return nil, false, fmt.Errorf("failed to acquire lock: %w", err)
}

// Unlock releases the lock.
func (l *Lock) Unlock() error {
	if l == nil || l.file == nil {
		return nil
	}

	defer tracker.release(l.path)
	errUnlock := unix.Flock(int(l.file.Fd()), unix.LOCK_UN)
	errClose := l.file.Close()
	l.file = nil

	if errUnlock != nil {
		return fmt.Errorf("failed to unlock file: %w", errUnlock)
	}
	if errClose != nil {
		return fmt.Errorf("failed to close lock file: %w", errClose)
	}

	return nil
}
