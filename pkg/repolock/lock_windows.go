//go:build windows

package repolock

import (
	"context"
	"fmt"
	"time"
)

// Lock is a placeholder for Windows build support.
type Lock struct{}

// Acquire is not implemented on Windows in this foundational task.
func Acquire(_ context.Context, _ string, _ time.Duration) (*Lock, error) {
	return nil, fmt.Errorf("repo locking is not implemented on windows in this release")
}

// TryAcquire is not implemented on Windows in this foundational task.
func TryAcquire(_ string) (*Lock, bool, error) {
	return nil, false, fmt.Errorf("repo locking is not implemented on windows in this release")
}

// Unlock is a no-op placeholder on Windows.
func (l *Lock) Unlock() error {
	_ = l
	return nil
}
