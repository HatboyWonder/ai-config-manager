package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"

	"github.com/dynatrace-oss/ai-config-manager/v3/pkg/repolock"
	"gopkg.in/yaml.v3"
)

const (
	commandExitCodeSuccess               = 0
	commandExitCodeCompletedWithFindings = 1
	commandExitCodeOperationalFailure    = 2
)

type commandErrorCategory string

const (
	commandErrorCategoryRepoBusy   commandErrorCategory = "repo_busy"
	commandErrorCategoryIOError    commandErrorCategory = "io_error"
	commandErrorCategoryParseError commandErrorCategory = "parse_error"
	commandErrorCategoryInternal   commandErrorCategory = "internal_error"
)

// commandExitError provides typed command-level exit propagation so command
// handlers can control process exit code without calling os.Exit in command
// bodies.
type commandExitError struct {
	ExitCode       int
	Category       commandErrorCategory
	SuppressStderr bool
	Cause          error
}

func (e *commandExitError) Error() string {
	if e == nil || e.Cause == nil {
		return "command failed"
	}

	return e.Cause.Error()
}

func (e *commandExitError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.Cause
}

func newCompletedWithFindingsError(message string) error {
	return &commandExitError{
		ExitCode:       commandExitCodeCompletedWithFindings,
		SuppressStderr: true,
		Cause:          errors.New(message),
	}
}

func newOperationalFailureError(err error) error {
	return &commandExitError{
		ExitCode:       commandExitCodeOperationalFailure,
		Category:       classifyOperationalError(err),
		SuppressStderr: false,
		Cause:          err,
	}
}

func newSuppressedOperationalFailureError(err error) error {
	return &commandExitError{
		ExitCode:       commandExitCodeOperationalFailure,
		Category:       classifyOperationalError(err),
		SuppressStderr: true,
		Cause:          err,
	}
}

func newOperationalFailureErrorWithCategory(err error, category commandErrorCategory) error {
	return &commandExitError{
		ExitCode:       commandExitCodeOperationalFailure,
		Category:       category,
		SuppressStderr: false,
		Cause:          err,
	}
}

func newSuppressedOperationalFailureErrorWithCategory(err error, category commandErrorCategory) error {
	return &commandExitError{
		ExitCode:       commandExitCodeOperationalFailure,
		Category:       category,
		SuppressStderr: true,
		Cause:          err,
	}
}

func getCommandExitCode(err error) int {
	if err == nil {
		return commandExitCodeSuccess
	}

	var commandErr *commandExitError
	if errors.As(err, &commandErr) {
		if commandErr.ExitCode >= 0 {
			return commandErr.ExitCode
		}
	}

	return 1
}

func getCommandErrorCategory(err error) commandErrorCategory {
	var commandErr *commandExitError
	if errors.As(err, &commandErr) {
		return commandErr.Category
	}

	return commandErrorCategoryInternal
}

func classifyOperationalError(err error) commandErrorCategory {
	if err == nil {
		return commandErrorCategoryInternal
	}

	var timeoutErr *repolock.AcquireTimeoutError
	if errors.As(err, &timeoutErr) || errors.Is(err, repolock.ErrNonReentrantLock) {
		return commandErrorCategoryRepoBusy
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return commandErrorCategoryRepoBusy
	}

	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		return commandErrorCategoryIOError
	}

	var syntaxErr *json.SyntaxError
	var unmarshalTypeErr *json.UnmarshalTypeError
	var yamlTypeErr *yaml.TypeError
	if errors.As(err, &syntaxErr) || errors.As(err, &unmarshalTypeErr) || errors.As(err, &yamlTypeErr) {
		return commandErrorCategoryParseError
	}

	return commandErrorCategoryInternal
}

func wrapLockAcquireError(lockPath string, err error) error {
	return newOperationalFailureErrorWithCategory(
		fmt.Errorf("failed to acquire repository lock at %s: %w", lockPath, err),
		commandErrorCategoryRepoBusy,
	)
}

func wrapReadLockAcquireError(lockPath string, err error) error {
	return newOperationalFailureErrorWithCategory(
		fmt.Errorf("failed to acquire repository read lock at %s: %w", lockPath, err),
		commandErrorCategoryRepoBusy,
	)
}
