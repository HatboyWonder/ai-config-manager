package repolock

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"
)

const (
	defaultAcquireTimeout = 30 * time.Second
	defaultPollInterval   = 50 * time.Millisecond
)

var (
	// ErrNonReentrantLock indicates the same process attempted to acquire the same
	// lock path multiple times. This package intentionally treats locks as
	// non-reentrant to prevent nested lock deadlocks (for example: top-level
	// operation acquires repo lock, then an internal helper tries to acquire it
	// again).
	ErrNonReentrantLock = errors.New("lock is non-reentrant")
)

// AcquireTimeoutError is returned when a lock could not be acquired before the
// configured timeout elapsed.
type AcquireTimeoutError struct {
	Path    string
	Timeout time.Duration
}

func (e *AcquireTimeoutError) Error() string {
	return fmt.Sprintf("timed out acquiring lock %q after %s", e.Path, e.Timeout)
}

// Is lets callers match timeout lock acquisition errors with errors.Is.
func (e *AcquireTimeoutError) Is(target error) bool {
	_, ok := target.(*AcquireTimeoutError)
	return ok
}

// Manager builds explicit lock paths and acquires OS-backed file locks rooted at
// <repo>/.workspace/locks/.
//
// Lock ordering invariant for operations that need multiple locks:
//  1. repo lock
//  2. cache lock
//  3. workspace metadata lock
//
// Never acquire in reverse order.
type Manager struct {
	repoPath string
}

// NewManager creates a lock manager for a repository path.
func NewManager(repoPath string) *Manager {
	return &Manager{repoPath: repoPath}
}

// RepoLockPath returns <repo>/.workspace/locks/repo.lock.
func (m *Manager) RepoLockPath() string {
	return filepath.Join(m.repoPath, ".workspace", "locks", "repo.lock")
}

// WorkspaceMetadataLockPath returns
// <repo>/.workspace/locks/workspace-metadata.lock.
func (m *Manager) WorkspaceMetadataLockPath() string {
	return filepath.Join(m.repoPath, ".workspace", "locks", "workspace-metadata.lock")
}

// CacheLockPath returns <repo>/.workspace/locks/caches/<cache-hash>.lock.
func (m *Manager) CacheLockPath(cacheHash string) string {
	return filepath.Join(m.repoPath, ".workspace", "locks", "caches", cacheHash+".lock")
}

// AcquireRepoLock acquires the repo-wide lock using default CLI semantics:
// block until available, up to 30 seconds, honoring context cancellation.
func (m *Manager) AcquireRepoLock(ctx context.Context) (*Lock, error) {
	return Acquire(ctx, m.RepoLockPath(), defaultAcquireTimeout)
}

// AcquireWorkspaceMetadataLock acquires the shared workspace metadata lock.
func (m *Manager) AcquireWorkspaceMetadataLock(ctx context.Context) (*Lock, error) {
	return Acquire(ctx, m.WorkspaceMetadataLockPath(), defaultAcquireTimeout)
}

// AcquireCacheLock acquires a per-cache lock for the provided cache hash.
func (m *Manager) AcquireCacheLock(ctx context.Context, cacheHash string) (*Lock, error) {
	return Acquire(ctx, m.CacheLockPath(cacheHash), defaultAcquireTimeout)
}

// TryAcquireRepoLock attempts to acquire the repo lock without waiting.
func (m *Manager) TryAcquireRepoLock() (*Lock, bool, error) {
	return TryAcquire(m.RepoLockPath())
}

type heldLockTracker struct {
	mu   sync.Mutex
	held map[string]int
}

var tracker = &heldLockTracker{held: make(map[string]int)}

func (t *heldLockTracker) claim(path string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.held[path] > 0 {
		return fmt.Errorf("%w: %s", ErrNonReentrantLock, path)
	}
	t.held[path]++
	return nil
}

func (t *heldLockTracker) release(path string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.held[path] <= 1 {
		delete(t.held, path)
		return
	}
	t.held[path]--
}
