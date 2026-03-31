# Workspace Caching

This document explains how aimgr's workspace caching system optimizes Git repository operations.

## Overview

Git repositories are cached in the `.workspace/` directory for efficient reuse across all Git operations. This significantly improves performance when working with remote repositories.

## Concurrency Guarantees

Workspace cache mutations are protected by OS-backed advisory file locks under
`<repo>/.workspace/locks/`.

Lock files:

- Repo-wide lock: `<repo>/.workspace/locks/repo.lock`
- Per-cache lock: `<repo>/.workspace/locks/caches/<cache-hash>.lock`
- Shared cache metadata lock: `<repo>/.workspace/locks/workspace-metadata.lock`

Required lock ordering is always:

1. repo lock
2. cache lock
3. workspace metadata lock

Notes:

- Top-level mutating repo commands hold the outer repo lock.
- Cache clone/update/remove operations are serialized per cache hash.
- `.workspace/.cache-metadata.json` updates are serialized with the metadata lock and written with atomic replacement.
- Workspace metadata lock is held only for short read-modify-write sections, not long git network operations.
- On Unix/POSIX builds this uses `flock` (`LOCK_SH` for repo read locks and `LOCK_EX` for repo write locks on the shared `repo.lock` path).
- On Windows, locks use `LockFileEx`/`UnlockFileEx`; repo read-lock APIs currently fall back to exclusive locking, so concurrent shared readers are not currently enabled on Windows.

### Atomic replacement for workspace cache metadata

`.workspace/.cache-metadata.json` uses the same atomic-write helper used for
other repo-managed state files (`ai.repo.yaml`, `.metadata/sources.json`, and
resource metadata files under `.metadata/...`).

For each write, aimgr:

1. creates a temp file in the same directory,
2. writes full JSON and `fsync`s the temp file,
3. renames the temp file over `.cache-metadata.json`,
4. `fsync`s the parent directory where supported.

This provides crash-safe replacement semantics for the file content. It does
not replace lock-based serialization for concurrent read-modify-write updates.

## Performance Benefits

- **First operation** (`repo add`, `repo sync`): Full git clone (creates cache)
- **Subsequent operations**: Reuse cached repository (10-50x faster)
- **Automatic cache management** with SHA256 hash-based storage
- **Shared across all resources** from the same source repository

## Commands Using Workspace Cache

### repo add
Adds resources using cached clone:
```bash
aimgr repo add gh:owner/repo
aimgr repo add https://github.com/owner/repo
```

### repo sync
Syncs from configured sources using cached clones:
```bash
aimgr repo sync
aimgr repo sync --format=json
```

## Batching Performance

Repository commands automatically batch resources from the same Git repository, cloning each unique source only once.

**Example**: 39 resources from one repository = 1 cached clone reused 39 times

This optimization significantly improves performance for bulk operations.

## Workspace Directory Structure

```
~/.local/share/ai-config/repo/    # Default location (configurable)
├── .workspace/                   # Git repository cache
│   ├── <hash-1>/                 # Cached repository 1 (by URL hash)
│   │   ├── .git/
│   │   ├── commands/
│   │   └── skills/
│   ├── <hash-2>/                 # Cached repository 2
│   │   └── ...
│   └── .cache-metadata.json      # Cache metadata (URLs, timestamps, refs)
├── .metadata/                    # Resource metadata
├── commands/                     # Command resources
├── skills/                       # Skill resources
└── agents/                       # Agent resources
```

**Note:** The repository path is configurable. See [configuration.md](../user-guide/configuration.md) for details on customizing the repository location via `repo.path` config or `AIMGR_REPO_PATH` environment variable.

### Hash-Based Storage

Each cached repository is stored in a directory named by the SHA256 hash of its URL. This ensures:
- **Unique storage** for each repository
- **Collision-free** caching
- **Efficient lookup** by URL

### Cache Metadata

The `.cache-metadata.json` file tracks:
- Repository URLs
- Clone timestamps
- Git refs (branches, tags, commits)
- Last access time

## Cache Management

### View Cache Status

Check workspace cache usage:
```bash
ls -lh ~/.local/share/ai-config/repo/.workspace/
```

### Prune Unreferenced Caches

Remove caches that are no longer referenced by any resources:

```bash
# Preview what would be removed
aimgr repo prune --dry-run

# Remove unreferenced caches
aimgr repo prune

# Force remove without confirmation
aimgr repo prune --force
```

**When to prune**:
- After removing many resources
- When `.workspace/` grows too large
- To free up disk space

### Manual Cache Cleanup

If needed, you can manually remove the workspace cache:
```bash
rm -rf ~/.local/share/ai-config/repo/.workspace/
```

The cache will be recreated on the next Git operation.

## How It Works

### 1. First Clone

When you add a resource from a Git repository for the first time:

```bash
aimgr repo add gh:owner/repo
```

1. Calculate SHA256 hash of the repository URL
2. Clone the repository to `.workspace/<hash>/`
3. Extract resources from the cached repository
4. Save metadata about the cache
5. Copy resources to the main repository

### 2. Subsequent Operations

When you add more resources from the same repository:

```bash
aimgr repo add gh:owner/repo --filter "skill/*"
```

1. Calculate SHA256 hash of the repository URL
2. Check if `.workspace/<hash>/` exists
3. Reuse existing cached repository (no clone needed)
4. Extract resources from the cache
5. Copy resources to the main repository

**Result**: 10-50x faster than re-cloning

## Cache Lifecycle

### Creation
- Cache created on first `repo add` from a Git source
- Full clone operation

### Reuse
- Cache reused for all subsequent operations on the same source
- No network operations for resource extraction

### Pruning
- Cache removed if no resources reference it
- Triggered by `repo prune` command

## Implementation Details

### Cache Key Generation

```go
import "crypto/sha256"

// Generate cache key from repository URL
func getCacheKey(repoURL string) string {
    hash := sha256.Sum256([]byte(repoURL))
    return hex.EncodeToString(hash[:])
}
```

### Cache Lookup

```go
// Check if cache exists
func cacheExists(repoURL string) bool {
    key := getCacheKey(repoURL)
    cachePath := filepath.Join(workspaceDir, key)
    _, err := os.Stat(cachePath)
    return err == nil
}
```

### Cache Operations

The `pkg/workspace/` package provides:
- `GetOrClone()` - Clone repository to cache if missing, otherwise reuse existing cache
- `Update()` - Refresh cached repository state
- `ListCached()` - Enumerate cached source URLs
- `Prune()` - Remove unreferenced caches
- `Remove()` - Remove one cached repository

## Best Practices

1. **Let the cache build naturally**: Don't manually populate `.workspace/`
2. **Prune periodically**: Run `repo prune` after removing many resources
3. **Don't edit cached repos**: Cached repositories are read-only for aimgr
4. **Use patterns for bulk ops**: `--filter` flags work with cached repos
5. **Monitor disk usage**: Large repositories consume space even when cached

## Troubleshooting

### Cache Corruption

If a cached repository becomes corrupted:

```bash
# Remove the entire workspace cache
rm -rf ~/.local/share/ai-config/repo/.workspace/

# Or remove a specific cache
rm -rf ~/.local/share/ai-config/repo/.workspace/<hash>/
```

The cache will be recreated on the next operation.

### Stale Cache

If a cached repository has outdated content:

```bash
# Force a fresh clone by removing the cache
rm -rf ~/.local/share/ai-config/repo/.workspace/
```

### Disk Space Issues

If `.workspace/` is consuming too much space:

```bash
# Check workspace size
du -sh ~/.local/share/ai-config/repo/.workspace/

# Prune unreferenced caches
aimgr repo prune

# Or remove entire workspace (will be recreated)
rm -rf ~/.local/share/ai-config/repo/.workspace/
```

## Related Documentation

- [Repository Layout](./repository-layout.md) - Repository folder structure
- [Git Tracking](./git-tracking.md) - How aimgr uses Git
- [Supported Tools](../reference/supported-tools.md) - Tool support and resource format documentation
