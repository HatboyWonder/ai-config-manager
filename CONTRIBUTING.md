# Contributing to aimgr

## Quick Start

### Prerequisites

- **Go 1.25.6+** (or use [mise](https://mise.jdx.dev/) for automatic version management)
- **Make** (build automation)
- **Git** (version control)

### Setup

```bash
# Clone the repository
git clone https://github.com/dynatrace-oss/ai-config-manager.git
cd ai-config-manager

# Build the binary
make build

# Run the baseline contributor test suite
make test
```

### Installation Paths by Operating System

The `make install` target automatically detects your OS and installs the `aimgr` binary to the appropriate location:

#### macOS

- **Install Path**: `/usr/local/bin`
- **Why this location**: 
  - Already in your system PATH (no manual PATH configuration needed)
  - Used by Homebrew and other package managers
  - Works with shell completion out of the box
  - No `sudo` required (if `/usr/local/bin` exists)
- **Shell Completion**: Install the binary first, then set up completions separately with `aimgr completion <shell>`
- **Command**: `make install` → binary at `/usr/local/bin/aimgr`

#### Linux (Ubuntu, Arch, etc.)

- **Install Path**: `~/.local/bin` (XDG Base Directory standard)
- **Why this location**:
  - User-specific installation (no `sudo` required)
  - Follows XDG Base Directory Specification
  - Keeps system directories clean
- **Note**: You may need to add `~/.local/bin` to your PATH if not already present:
  ```bash
  export PATH="$HOME/.local/bin:$PATH"
  ```
  Add this line to your `~/.bashrc`, `~/.zshrc`, or equivalent shell config file.
- **Command**: `make install` → binary at `~/.local/bin/aimgr`

#### Windows (experimental support)

- **Install Path**: `%USERPROFILE%\AppData\Local\bin`
- **Note**: Windows support is prepared but may require additional testing

#### Checking Your Installation

After running `make install`, verify the installation:

```bash
# Show where the binary was installed
make os-info

# Verify the binary works
aimgr --version
```

If `aimgr` is not found, add the install path to your PATH environment variable or run the binary with its full path.

### Verify Your Setup

```bash
# Check Go version
go version  # Should show 1.25.6 or higher

# Build and test
make build
make test   # Should pass the baseline contributor checks
```

## Development Workflow

### 1. Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork locally
3. Add upstream remote:
   ```bash
   git remote add upstream https://github.com/dynatrace-oss/ai-config-manager.git
   ```

### 2. Create a Feature Branch

```bash
git checkout -b feature/your-feature-name
```

### 3. Make Changes

1. Write your code
2. Add tests for new functionality
3. Follow [docs/CODING.md](docs/CODING.md) for repository-specific implementation rules
4. Run tests frequently: `make test`

### 4. Commit Your Changes

Follow conventional commit format (see below)

### 5. Land the Change

Use [docs/CHANGE-WORKFLOW.md](docs/CHANGE-WORKFLOW.md) to choose the right landing path for this change.

If you are publishing a feature branch, push it with:

```bash
git push -u origin feature/your-feature-name
```

## Submitting Changes

### Before Submitting

- [ ] Minimum checks from [docs/TESTING.md](docs/TESTING.md) passed for the change type
- [ ] New code has tests
- [ ] Documentation updated when commands, workflows, or user-facing behavior changed
- [ ] Commit messages follow the format below

### Commit Message Format

Use conventional commits:

```
type(scope): short description

Longer description if needed, explaining what and why.

Fixes #issue-number
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `style`: Formatting, missing semicolons, etc.
- `refactor`: Code restructuring
- `test`: Adding tests
- `chore`: Maintenance, dependencies, etc.

**Examples:**

```
feat(repo): add bulk import support for plugins

Add ability to import multiple commands and skills from Claude plugins
in a single operation.

Fixes #42
```

```
fix(install): handle symlink creation on Windows

Use junction points instead of symlinks for Windows compatibility.
```

### Change Landing

Read **[docs/CHANGE-WORKFLOW.md](docs/CHANGE-WORKFLOW.md)** for commit, push, branch, PR, review, and merge expectations.

## Deeper Docs

Use the focused project docs when you need repository-local detail:

- [docs/OVERVIEW.md](docs/OVERVIEW.md) - architecture map and where common work starts
- [docs/CODING.md](docs/CODING.md) - implementation constraints, build commands, and safety rules
- [docs/TESTING.md](docs/TESTING.md) - test selection, isolation rules, and minimum checks
- [docs/CHANGE-WORKFLOW.md](docs/CHANGE-WORKFLOW.md) - commit, push, branch, PR, and merge expectations
- [docs/contributor-guide/README.md](docs/contributor-guide/README.md) - deeper contributor references for setup, architecture, and test authoring

## Getting Help

- **Issues**: [GitHub Issues](https://github.com/dynatrace-oss/ai-config-manager/issues)
- **Discussions**: [GitHub Discussions](https://github.com/dynatrace-oss/ai-config-manager/discussions)
- **Documentation**: 
  - User docs: [README.md](README.md)
  - AI agent guide: [AGENTS.md](AGENTS.md)
  - Contributor docs: [docs/contributor-guide/](docs/contributor-guide/)

---

Thank you for contributing to aimgr! 🎉
