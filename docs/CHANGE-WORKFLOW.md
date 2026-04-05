# Change Workflow

## Choose a landing path

| Situation | Path | Start with |
| --- | --- | --- |
| Small change landing without a PR | Direct push on `main` | `git checkout main && git pull --rebase` |
| Change needs review or the user asked for a PR | Branch + PR to `main` | `git checkout main && git pull --rebase && git checkout -b <type>/<name>` |

Use focused branch names such as `feature/<name>`, `fix/<name>`, or `docs/<name>`.

## Common requirements before push

- Track implementation work in `bd`; run `bd prime` when you need the full tracker workflow.
- Use the conventional commit format from [CONTRIBUTING.md](../CONTRIBUTING.md#commit-message-format).
- Run the minimum checks from [TESTING.md](TESTING.md) before committing.
- Update docs in the same change when commands, paths, workflows, or user-facing behavior changed.

## Direct push on `main`

1. `git checkout main`
2. `git pull --rebase`
3. Make the change and run the required checks from [TESTING.md](TESTING.md).
4. `git add <files>`
5. `git commit -m "<type(scope): summary>"`
6. `bd dolt push`
7. `git push origin main`
8. `git status --short --branch`

Finish only when `git status --short --branch` shows no uncommitted changes and `main` is up to date with `origin/main`.

## Branch + PR to `main`

1. `git checkout main`
2. `git pull --rebase`
3. `git checkout -b <type>/<name>`
4. Make the change and run the required checks from [TESTING.md](TESTING.md).
5. `git add <files>`
6. `git commit -m "<type(scope): summary>"`
7. `bd dolt push`
8. `git push -u origin <branch>`
9. Open a PR to `main` that includes:
   - what changed
   - why it changed
   - how to test it
   - related issue links such as `Fixes #42` when applicable
10. Wait for `.github/workflows/build.yml` checks to pass.
11. Maintainers merge after approval.
12. After merge, `git checkout main && git pull --rebase`

## Validation

- Use [TESTING.md](TESTING.md) for the authoritative change-to-check map before commit or PR creation.
- For docs-only edits, verify the touched links, paths, and commands in the affected files.

## Related docs

- [CONTRIBUTING.md](../CONTRIBUTING.md) — contributor setup and commit message format
- [TESTING.md](TESTING.md) — exact validation commands and minimum checks
- [RELEASING.md](RELEASING.md) — release tagging and publish workflow
