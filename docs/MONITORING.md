# Monitoring

## Start here

- Load the **observability-triage** skill for log analysis and issue triage.
- This repository is log-first; it does not have a production metrics/dashboard backend wired into the repo.
- Prioritize local health signals for `aimgr`, `bd`, and OpenCode/opencode-coder.

## Fast evidence path

1. `bd doctor` / `bd version` / `./aimgr --version` (always available commands)
2. `.beads/dolt-server.log` (when beads services have run)
3. `~/.local/share/opencode/log/*.log` (user-level OpenCode runtime logs)
4. `.beads/daemon-error` (if present)
5. `logs/operations.log` (if aimgr file logging is enabled)
6. `.beads/daemon.log` (optional; environment/version-dependent)

## Available Data

### Local tool logs

Expected local signals:

- `.beads/dolt-server.log`
  - Dolt SQL server startup, connections, and database-level errors used by beads (after running beads commands that start/use services).
- `~/.local/share/opencode/log/*.log`
  - OpenCode runtime logs, plugin startup, config warnings, and session errors.

Optional / runtime-created / historical signals:

- `.beads/daemon-error` (optional)
  - Last fatal daemon/bootstrap issue when the beads daemon records one. Treat it as high-signal when present.
- `logs/operations.log` (runtime-created)
  - Aimgr structured repo-operation log when aimgr file logging is active.
- `.beads/daemon.log` (optional, environment/version-dependent)
  - Daemon lifecycle/import-export detail when a beads daemon log is emitted in the local setup.
- `debug.log` (historical)
  - Historical local debug output from related tooling in this workspace.

### Health commands

- `./aimgr --version`
- `bd version`
- `bd doctor`
- `git status --short --branch`

Use the local `./aimgr` binary for validation, not a globally installed `aimgr` from PATH.

## What To Look For

### Critical

- Beads repository/database mismatch warnings
- `bd doctor` hangs or never returns
- Repeated database errors in `.beads/dolt-server.log`
- Machine-readable command output being polluted by warnings/noise

### Important

- Structured logging corruption such as `!BADKEY`
- Repeated import/export validation failures in beads logs
- Plugin load failures or missing command/skill files in OpenCode logs
- Unexpected stderr output on otherwise successful commands

### Usually Safe To Ignore

- Historical warnings from unrelated repositories in shared OpenCode logs
- One-off config deprecation warnings unless they are blocking behavior
- Old failures that cannot be reproduced in the current checkout

## Context And Meaning

- `.beads/daemon-error` is high priority when present because it can indicate tracker-state corruption risk.
- `!BADKEY` in slog-style logs usually means formatted logging is being used incorrectly and the resulting logs are less trustworthy for diagnosis.
- `fatal: couldn't find remote ref main` in old daemon logs may be historical; confirm it still reproduces before treating it as an active bug.
- Empty `logs/operations.log` is not automatically a bug; it may simply mean no aimgr operations were run with file logging enabled.

## Current Triage Approach

When asked to analyze monitoring data for this repo:

1. Run lightweight health commands (`bd version`, `bd doctor`, `./aimgr --version`)
2. Review `.beads/dolt-server.log` when beads services have run
3. Review current OpenCode logs in `~/.local/share/opencode/log/`
4. Check `.beads/daemon-error` and `.beads/daemon.log` only when present
5. Include `logs/operations.log` only when aimgr file logging is enabled
6. Group findings by root cause and create/update beads issues for clear tooling bugs
