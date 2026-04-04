# Pull Request Guide

## Branch setup

- Start from an up-to-date `main` branch.
- Create a focused branch such as `feature/your-feature-name`, `fix/your-bug-name`, or `docs/your-doc-change`.
- Keep each branch focused on one logical change.
- Push your branch to your fork or remote before opening a PR.

## Required local checks before opening a PR

- Run `make fmt` to format Go code.
- Run `make vet` for static analysis.
- Run `make test` for the baseline contributor test flow.
- Add `make e2e-test` when the change affects CLI entrypoints, installation flows, scripts, or other end-to-end user workflows.
- Update documentation when the change affects users, contributors, or AI agents.

## PR Description

Open the PR from your feature branch to `main` and include:

1. What changed
2. Why the change was needed
3. How to test it

Link related issues with GitHub keywords such as `Fixes #42` when applicable.

## Review Follow-up

- Wait for the checks in `.github/workflows/build.yml` to pass before asking for merge.
- Address review feedback promptly and respectfully.
- Maintainers merge after approval.

## Related Docs

- [CONTRIBUTING.md](../CONTRIBUTING.md) - Contribution workflow and commit message format
- [docs/CODING.md](CODING.md) - Build commands and code conventions
- [docs/TESTING.md](TESTING.md) - Test commands and isolation rules
