# Release Process

## Start here

- Load the **github-releases** skill before doing release work.
- Use `docs/contributor-guide/release-process.md` for the detailed operator checklist after the skill confirms the version and execution path.

## Repo-local release facts

- Release automation lives in `.github/workflows/release.yml` and triggers on `git push` of tags matching `v*`.
- Packaging and version injection live in `.goreleaser.yaml`.
- Build-time version fields are injected into `github.com/dynatrace-oss/ai-config-manager/v3/pkg/version`.
- `CHANGELOG.md` is the release-history file that must be updated before tagging.
- Release work is destructive and externally visible; do not improvise from this file alone.

## Required local checks before tagging

- `make test`
- `make build`
- `git status --short --branch`
- Confirm the intended release notes update in `CHANGELOG.md`

## Operator references

- `.github/workflows/release.yml` — tag-triggered publishing workflow
- `.goreleaser.yaml` — archive, checksum, and ldflags configuration
- `docs/contributor-guide/release-process.md` — detailed tagging, monitoring, and rollback steps
