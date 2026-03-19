# Coding Guide

Essential coding reference for ai-config-manager contributors.

## CRITICAL: Repository Safety for Testing

**NEVER run `aimgr repo` commands against the global repository during testing or bug reproduction!**

The default repository location is `~/.local/share/ai-config/repo/` which contains your real aimgr configuration. Testing against this will corrupt your development environment.

**Safe methods:**

| Method | Usage |
|--------|-------|
| Environment variable (recommended) | `export AIMGR_REPO_PATH=/tmp/test-repo-$(date +%s)` |
| Config file | `aimgr --config /tmp/test-config.yaml repo init` |
| Go tests | `repo.NewManagerWithPath(t.TempDir())` |

**Bottom line**: Every test operation MUST explicitly specify a temporary repository location. No exceptions.

## CRITICAL: Use Locally Built Binary

**ALWAYS use `./aimgr` (the locally built binary) when testing changes, NOT `aimgr` from PATH!**

Version managers (mise, asdf, etc.) may install older versions that are found first in PATH.

```bash
# CORRECT: Use local binary
./aimgr --version
./aimgr repo init

# WRONG: May use mise/asdf version
aimgr --version
```

## Concurrency and Locking Model (Repo/Workspace Mutations)

All repo mutations are coordinated with OS-backed advisory file locks under:

`<repo>/.workspace/locks/`

Lock files:

- Repo-wide lock: `<repo>/.workspace/locks/repo.lock`
- Workspace metadata lock: `<repo>/.workspace/locks/workspace-metadata.lock`
- Per-cache lock: `<repo>/.workspace/locks/caches/<cache-hash>.lock`

Lock ordering is strict and must never be reversed:

1. repo lock
2. cache lock
3. workspace metadata lock

Rules:

- Top-level mutating CLI commands are outermost repo-lock holders.
- Workspace cache operations may take cache lock and metadata lock, but must not try to take repo lock from inside those sections.
- Workspace metadata lock is only for short metadata read-modify-write sections (not long clone/fetch/pull work).

Scope / limitations:

- These locks are implemented with `flock` on Unix/POSIX builds.
- Windows lock implementation is currently not available in this release.
- Locks serialize *aimgr* mutation paths; they do not prevent arbitrary external tools from modifying files directly.

## Atomic Write Model (Repo-Managed State Files)

Repo-managed state files are written with **atomic replacement**, not in-place
truncating writes.

State files covered by this model include:

- `ai.repo.yaml`
- `.metadata/sources.json`
- Resource metadata files under `.metadata/...` (for example
  `.metadata/skills/<name>-metadata.json`)
- `.workspace/.cache-metadata.json`

Write sequence (via `pkg/fileutil.AtomicWrite`):

1. Create a temporary file in the **same directory** as the target file.
2. Write full file contents to the temp file.
3. `fsync` the temp file.
4. Rename the temp file over the destination path.
5. `fsync` the parent directory where supported.

Important limits:

- Parent directories must already exist before writing.
- Atomic replacement protects against partial-file writes during process crashes,
  but it does not by itself resolve concurrent read-modify-write races; locks
  still provide serialization for mutation paths.
- Parent-directory `fsync` is best-effort by platform; on Windows this layer
  currently does not perform directory `fsync`.

## Quick Commands

```bash
# Build
make build      # Build binary to ./aimgr
make install    # Build and install to ~/bin

# Test
make test             # All tests (vet -> unit -> integration)
make unit-test        # Fast unit tests only
make integration-test # Integration tests

# Code Quality
make fmt        # Format all Go code
make vet        # Run go vet
```

## Project Structure

```
cmd/    CLI command implementations (Cobra)
pkg/    Business logic (20 packages)
test/   Integration and E2E tests
docs/   Documentation
```

**Architecture**: CLI (Cobra) -> Business Logic (`pkg/`) -> Storage (XDG directories)

## Detailed Guides

- **[Code Style](contributor-guide/code-style.md)** -- Naming, imports, error handling, symlink handling, best practices
- **[Architecture](contributor-guide/architecture.md)** -- System overview, package responsibilities, 5 critical rules, data flows
- **[Development Environment](contributor-guide/development-environment.md)** -- IDE setup, mise, build tools

## Before Committing

1. `make fmt` -- Format code
2. `make test` -- All tests pass
3. Follow [code style guide](contributor-guide/code-style.md)
4. Git operations use `pkg/workspace` (see [architecture](contributor-guide/architecture.md))
5. Tests use `t.TempDir()` and `NewManagerWithPath()` (see [testing](contributor-guide/testing.md))
