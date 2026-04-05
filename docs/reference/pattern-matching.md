# Pattern Matching

This document explains how to use glob patterns to filter and match resources in aimgr.

## Overview

The `pkg/pattern` package provides glob pattern matching for resources. Patterns allow you to select multiple resources without specifying each one individually.

## Pattern Syntax

### Basic Wildcards

- `*` - Matches any sequence of characters, including `/` in nested resource names such as `api/deploy`
- `?` - Matches any single character
- `[abc]` - Matches any character in the set (a, b, or c)
- `[a-z]` - Matches any character in the range (a through z)
- `{a,b}` - Matches either a or b

### Type Prefix

Patterns can optionally include a resource type prefix:

- `type/pattern` - Matches only resources of the specified type
- `pattern` - Matches resources of any type

**Valid types**: `command`, `skill`, `agent`, `package`

## Pattern Examples

### Simple Patterns

**Match all resources:**
```bash
aimgr repo list "*"
```

**Match resources starting with "test":**
```bash
aimgr repo list "test*"
```

**Match resources ending with "helper":**
```bash
aimgr repo list "*helper"
```

**Match resources containing "web":**
```bash
aimgr repo list "*web*"
```

### Type-Specific Patterns

**Match all skills:**
```bash
aimgr repo list "skill/*"
```

**Match all packages:**
```bash
aimgr repo list "package/*"
```

**Match all agents:**
```bash
aimgr repo list "agent/*"
```

**Match all commands:**
```bash
aimgr repo list "command/*"
```

### Combined Patterns

**Match skills starting with "pdf":**
```bash
aimgr repo list "skill/pdf*"
```

**Match packages with "tools" in the name:**
```bash
aimgr repo list "package/*tools*"
```

**Match commands starting with "test" or "build":**
```bash
aimgr repo list "command/{test,build}*"
```

**Match skills with version suffix:**
```bash
aimgr repo list "skill/*-v?"
```

## CLI Usage

### List Command

List resources matching a pattern:

```bash
# List all skills
aimgr repo list "skill/*"

# List skills starting with "web"
aimgr repo list "skill/web*"

# List all resources containing "test"
aimgr repo list "*test*"
```

### Install Command

Install resources matching a pattern:

```bash
# Install all skills
aimgr install "skill/*"

# Install commands starting with "test"
aimgr install "command/test*"
```

### Uninstall Command

Uninstall resources matching a pattern:

```bash
# Uninstall all agents
aimgr uninstall "agent/*"

# Uninstall skills ending with "helper"
aimgr uninstall "skill/*helper"
```

### Repository Commands

**Add with filter:**
```bash
# Add only skills from a repository
aimgr repo add gh:owner/repo --filter "skill/*"

# Add only packages from a local directory
aimgr repo add local:./resources --filter "package/*"

# Add resources matching pattern
aimgr repo add gh:owner/repo --filter "web-*"
```

The same pattern syntax is used in shared `ai.repo.yaml` manifests via `sources[].include`
(for `aimgr repo apply-manifest <path-or-url>`):

```yaml
version: 1
sources:
  - name: community
    url: https://github.com/example/ai-tools
    include:
      - skill/*
      - command/release-*
```

`include` semantics match repeated `--filter` flags (`OR` logic): a resource is included if any
pattern matches.

After applying and syncing:

```bash
aimgr repo apply-manifest ./ai.repo.yaml
aimgr repo sync
```

`repo sync` continues using the `sources[].include` filters persisted by `repo apply-manifest`.

`aimgr repo remove` removes sources, not individual resources. Use pattern matching with `repo list`, `install`, `uninstall`, or `repo add --filter`.

## Code Examples

### Basic Pattern Matching

```go
import "github.com/dynatrace-oss/ai-config-manager/v3/pkg/pattern"

// Parse pattern to extract type and check if it's a pattern
resourceType, patternStr, isPattern := pattern.ParsePattern("skill/pdf*")
// Returns: resource.Skill, "pdf*", true
```

### Create a Matcher

```go
// Create a matcher for a pattern
matcher, err := pattern.NewMatcher("skill/pdf*")
if err != nil {
    return err
}

// Match against resources
res := &resource.Resource{Type: resource.Skill, Name: "pdf-processing"}
if matcher.Match(res) {
    fmt.Println("Matched!")
}
```

### Check Pattern Type

```go
// Check if pattern (vs exact name)
if matcher.IsPattern() {
    fmt.Println("This is a glob pattern")
}

// Get resource type filter (if specified)
resType := matcher.GetResourceType()  // Returns resource.Skill or ""
```

### Match by Name Only

```go
// Match by name only (useful when you already know the type)
if matcher.MatchName("pdf-processing") {
    fmt.Println("Name matches!")
}
```

## Pattern Features

### Type Filtering

When you specify a type prefix, the pattern only matches resources of that type:

```go
// Match all packages
matcher, _ := pattern.NewMatcher("package/*")
// Only matches packages, not skills/commands/agents

// Match specific package pattern
matcher, _ := pattern.NewMatcher("package/web-*")
// Only matches packages starting with "web-"
```

### Cross-Type Matching

Without a type prefix, patterns match across all resource types:

```go
// Match "test" in any resource type
matcher, _ := pattern.NewMatcher("*test*")
// Matches: command/test, skill/testing, agent/tester, package/test-suite
```

### Exact Matching

Patterns without wildcards match exact names:

```go
// Exact match (no wildcards)
matcher, _ := pattern.NewMatcher("skill/pdf-processing")
// Only matches skill named "pdf-processing"
```

## Common Use Cases

### Testing Resources

**Find all test-related resources:**
```bash
aimgr repo list "*test*"
```

**Install test tools:**
```bash
aimgr install "command/test*"
aimgr install "skill/test*"
```

**Uninstall test packages from a project:**
```bash
aimgr uninstall "package/test-*"
```

### Development Workflows

**Add development packages:**
```bash
aimgr repo add gh:owner/repo --filter "package/dev-*"
```

**Install web development tools:**
```bash
aimgr install "skill/web-*"
aimgr install "command/web-*"
```

**Uninstall old versions from a project:**
```bash
aimgr uninstall "*-v1"
```

### Bulk Operations

**List all resources by type:**
```bash
aimgr repo list "command/*" --format=json
aimgr repo list "skill/*" --format=json
aimgr repo list "agent/*" --format=json
aimgr repo list "package/*" --format=json
```

**Import only specific resource types from a local source:**
```bash
aimgr repo add local:./backup --filter "skill/*"
```

**List deprecated resources before cleanup:**
```bash
aimgr repo list "deprecated-*"
```

## Advanced Patterns

### Character Classes

**Match version suffixes:**
```bash
# Match v1, v2, v3, etc.
aimgr repo list "*-v[0-9]"

# Match va, vb, vc
aimgr repo list "*-v[a-c]"
```

### Brace Expansion

**Match multiple alternatives:**
```bash
# Match build or test commands
aimgr repo list "command/{build,test}"

# Match dev or prod packages
aimgr repo list "package/{dev,prod}-*"
```

### Complex Patterns

**Match specific naming patterns:**
```bash
# Match resources with namespace prefix
aimgr repo list "company-*"

# Match resources with category and version
aimgr repo list "*-web-v?"

# Match resources in specific categories
aimgr repo list "{dev,test,prod}-*"
```

## Pattern Matching Implementation

### ParsePattern Function

```go
func ParsePattern(arg string) (resource.ResourceType, string, bool) {
    parts := strings.SplitN(arg, "/", 2)

    var resourceType resource.ResourceType
    var pattern string

    if len(parts) == 2 {
        typeStr := parts[0]
        pattern = parts[1]

        switch typeStr {
        case "command":
            resourceType = resource.Command
        case "skill":
            resourceType = resource.Skill
        case "agent":
            resourceType = resource.Agent
        case "package":
            resourceType = resource.PackageType
        default:
            pattern = arg
            resourceType = ""
        }
    } else {
        pattern = arg
    }

    isPattern := IsPattern(pattern)
    return resourceType, pattern, isPattern
}
```

### Matcher Structure

```go
type Matcher struct {
    resourceType ResourceType  // Optional type filter
    pattern      glob.Glob     // Compiled glob pattern
    isPattern    bool          // Is this a pattern or exact match?
}

func (m *Matcher) Match(res *resource.Resource) bool {
    // Check type filter first
    if m.resourceType != "" && res.Type != m.resourceType {
        return false
    }
    
    // Then check name pattern
    return m.MatchName(res.Name)
}

func (m *Matcher) MatchName(name string) bool {
    return m.pattern.Match(name)
}
```

## Best Practices

1. **Use type prefixes** when filtering by resource type for efficiency
2. **Quote patterns** in shell commands to prevent shell expansion
3. **Test patterns** with `repo list` before using with `install` or `uninstall`
4. **Use `--dry-run`** when available for destructive operations
5. **Be specific** to avoid unintended matches

## Troubleshooting

### Pattern Not Matching

If your pattern doesn't match expected resources:

1. **Check quoting**: Use quotes to prevent shell expansion
   ```bash
   aimgr repo list "skill/*"  # Correct
   aimgr repo list skill/*    # May expand in shell
   ```

2. **Test incrementally**: Build patterns step by step
   ```bash
   aimgr repo list "*"           # All resources
   aimgr repo list "web*"        # Resources starting with "web"
   aimgr repo list "skill/web*"  # Skills starting with "web"
   ```

3. **Use exact names first**: Verify resources exist
   ```bash
aimgr repo list "skill/web-helper"  # Exact match
aimgr repo list "skill/web*"        # Pattern match
   ```

### Unexpected Matches

If your pattern matches too many resources:

1. **Add type prefix**: Narrow by resource type
   ```bash
aimgr repo list "test*"          # All resources starting with "test"
aimgr repo list "command/test*" # Only commands starting with "test"
   ```

2. **Be more specific**: Use longer patterns
   ```bash
   aimgr repo list "*test*"         # Too broad
   aimgr repo list "test-*"         # Better
   aimgr repo list "test-helper-*"  # Most specific
   ```

3. **Use character classes**: Match specific patterns
   ```bash
   aimgr repo list "*-v?"           # Only -v followed by one char
   aimgr repo list "*-v[0-9]"       # Only -v followed by digit
   ```

## Related Documentation

- [Supported Tools](./supported-tools.md) - Tool support and resource format documentation
- [Output Formats](./output-formats.md) - CLI output formats
