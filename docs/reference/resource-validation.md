# Resource Validation for Creators

`aimgr resource validate` is the creator-focused validation entry point for all resource types:

- `skill`
- `agent`
- `command`
- `package`

Use it before `repo add`, `repo sync`, or sharing resources with other teams.

## Command Syntax

```bash
aimgr resource validate <resource-id-or-path> [flags]
```

Target resolution order:

1. If `<resource-id-or-path>` exists on disk, aimgr validates it as a path target.
2. Otherwise aimgr expects a canonical ID in `type/name` form (for example `skill/my-skill`).

## Static vs Contextual Validation

### Static validation (no repository context required)

Used for:

- skills
- agents
- commands

Static validation checks resource structure/content only.

### Static + contextual validation (packages)

Package validation always includes:

1. **Static package schema validation** (`name`, `description`, `resources`, JSON shape)
2. **Contextual reference validation** for each package resource reference

Contextual validation resolves package references against a validation context (source root, local repo, or repo manifest).

## Context and Resolution Flags

### `--source-root`

Validate canonical IDs and package references against a specific local source tree.

```bash
aimgr resource validate --source-root ./my-resources skill/my-skill
aimgr resource validate --source-root ./my-resources package/release-bundle
```

### `--repo-manifest`

Load source context from an `ai.repo.yaml` (local path or supported URL form).

```bash
aimgr resource validate --repo-manifest ./ai.repo.yaml package/release-bundle
```

For canonical ID resolution, aimgr checks local path sources in the manifest.

### Context precedence for canonical IDs

When validating canonical IDs, aimgr resolves context in this order:

1. `--source-root`
2. local repo (if available)
3. `--repo-manifest`

If none are available, canonical-ID validation fails with a context error.

## Output Format (`--format`)

Supported formats:

- `table` (default)
- `json`
- `yaml`

Examples:

```bash
aimgr resource validate ./skills/my-skill --format=table
aimgr resource validate skill/my-skill --source-root ./resources --format=json
aimgr resource validate package/release-bundle --repo-manifest ./ai.repo.yaml --format=yaml
```

JSON/YAML include structured fields such as:

- `target`, `resolved_path`, `resolved_id`
- `resource_type`, `mode`, `context`, `valid`
- `diagnostics[]`, `summary`

## Exit Codes

`aimgr resource validate` uses three exit codes:

- `0`: validation passed
- `1`: validation failed (resource/schema/reference problems)
- `2`: usage/context/setup error (invalid target/flags, missing context, manifest/context load failure)

## Examples by Resource Type

### Skill

Path-based:

```bash
aimgr resource validate ./skills/skill-creator
```

Canonical ID:

```bash
aimgr resource validate --source-root ./resources skill/skill-creator
```

### Agent

Path-based:

```bash
aimgr resource validate ./agents/code-reviewer.md
```

Canonical ID:

```bash
aimgr resource validate --source-root ./resources agent/code-reviewer
```

### Command

Path-based:

```bash
aimgr resource validate ./commands/release/deploy.md
```

Canonical ID:

```bash
aimgr resource validate --source-root ./resources command/release/deploy
```

### Package

Path-based (explicit context):

```bash
aimgr resource validate ./packages/release-bundle.package.json --source-root ./resources
```

Canonical ID (manifest-backed context):

```bash
aimgr resource validate package/release-bundle --repo-manifest ./ai.repo.yaml
```

## Safety and Isolation

Validation is read-only and does not mutate your live repository state.

- Static validation reads only the target resource.
- Contextual package validation builds transient validation state from provided roots/manifest sources.
- Validation does not run repository mutation operations (`repo add`, `repo sync`, metadata writes, etc.).

This makes `resource validate` safe for local preflight checks and CI validation.

## Troubleshooting

### Missing package references (`missing_package_ref`)

Symptom:

- package validation fails with unresolved resource references.

Typical causes:

- typo in package resource ID
- referenced resource not present in selected context roots

Fixes:

1. Check canonical resource IDs in `resources[]`.
2. Ensure the right context is provided (`--source-root` or `--repo-manifest`).
3. Use the diagnostic suggestion text (includes likely canonical IDs when available).

### Mismatched canonical IDs

Symptom:

- canonical target resolves to a path that does not exist, or package refs do not match actual canonical names.

Typical causes:

- wrong resource type/name (`skill/foo` vs `command/foo`)
- nested command ID mismatch (for example `command/team/deploy` expected, `command/deploy` provided)

Fixes:

1. Verify the ID follows the canonical `type/name` scheme.
2. For nested resources, include the full canonical name.
3. Re-run with `--format=json` to inspect `resolved_id`, diagnostics, and suggestions.

### Canonical-ID validation without context (`context_required`)

Symptom:

- validating `type/name` fails with a context-required error.

Fixes:

1. Provide `--source-root <path>`; or
2. Provide `--repo-manifest <path-or-url>`; or
3. Run where a local aimgr repo is available and initialized.

Without any context, canonical IDs cannot be mapped to actual files.
