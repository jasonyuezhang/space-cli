# Semantic Versioning Guide

Space CLI uses [Semantic Versioning 2.0.0](https://semver.org/) for version management.

## Version Format

Versions follow the format: **MAJOR.MINOR.PATCH**

- **MAJOR** version for incompatible API changes
- **MINOR** version for new functionality in a backward-compatible manner
- **PATCH** version for backward-compatible bug fixes

## Automatic Version Bumping

A pre-commit hook automatically bumps the version based on your commit message:

### Keywords in Commit Messages

- **`[major]`** or **`[breaking]`** - Bumps major version (x.0.0)
  ```bash
  git commit -m "[major] Remove deprecated APIs"
  # 1.2.3 → 2.0.0
  ```

- **`[minor]`** or **`[feature]`** or **`[feat]`** - Bumps minor version (0.x.0)
  ```bash
  git commit -m "[feature] Add new DNS server"
  # 1.2.3 → 1.3.0
  ```

- **`[patch]`** or **default** - Bumps patch version (0.0.x)
  ```bash
  git commit -m "Fix bug in config loader"
  # 1.2.3 → 1.2.4
  ```

### How It Works

1. The pre-commit hook runs before each commit
2. It reads the current version from the `VERSION` file
3. It analyzes your commit message for keywords
4. It bumps the appropriate version number
5. It updates the `VERSION` file and stages it
6. The commit proceeds with the updated version

## Manual Version Management

You can also manage versions manually using the helper script or Make targets:

### Using Make Targets

```bash
# Show current version
make version

# Bump patch version (0.0.x)
make version-patch

# Bump minor version (0.x.0)
make version-minor

# Bump major version (x.0.0)
make version-major
```

### Using the Version Script Directly

```bash
# Show current version
./scripts/version.sh get

# Bump versions
./scripts/version.sh patch
./scripts/version.sh minor
./scripts/version.sh major

# Set specific version
./scripts/version.sh set 2.0.0

# Show help
./scripts/version.sh help
```

## Building with Version

The version is automatically embedded into the binary during build:

```bash
make build    # Builds with version from VERSION file
make install  # Installs with version from VERSION file
```

Check the version:

```bash
space --version
# Output: space version 0.1.0
```

## VERSION File

The `VERSION` file at the project root contains the current semantic version:

```
0.1.0
```

This file is:
- **Tracked in git** - Version history is preserved
- **Auto-updated** - By pre-commit hook on each commit
- **Read by Makefile** - Used during build process
- **Simple text** - Easy to read and edit manually if needed

## Examples

### Regular Development Flow

```bash
# Work on bug fix
git add .
git commit -m "Fix port binding issue"
# → Version bumps: 0.1.0 → 0.1.1

# Work on new feature
git add .
git commit -m "[feature] Add VM management commands"
# → Version bumps: 0.1.1 → 0.2.0

# Make breaking change
git add .
git commit -m "[breaking] Remove deprecated config format"
# → Version bumps: 0.2.0 → 1.0.0
```

### Manual Version Control

```bash
# Set version for release
./scripts/version.sh set 1.0.0

# Build and tag
make install
git tag v1.0.0
git push --tags
```

## Release Workflow

1. **Development**: Work on features/fixes
2. **Commit**: Use appropriate keywords in commit messages
3. **Build**: `make install` to build with new version
4. **Tag**: Create git tag matching version (optional)
   ```bash
   VERSION=$(cat VERSION)
   git tag "v$VERSION"
   git push origin "v$VERSION"
   ```
5. **Release**: Distribute the binary

## Best Practices

1. **Use Keywords**: Be explicit about version bumps in commit messages
2. **Semantic Meaning**: Follow semantic versioning principles
   - MAJOR: Breaking changes (incompatible API changes)
   - MINOR: New features (backward-compatible)
   - PATCH: Bug fixes (backward-compatible)
3. **Manual Overrides**: Use manual version commands sparingly
4. **Git Tags**: Tag releases with matching versions
5. **Changelog**: Keep a CHANGELOG.md documenting changes per version

## Version in Code

The version is embedded at build time using linker flags:

```go
package cli

var (
    // Version is set at build time
    Version = "dev"
    // BuildTime is set at build time
    BuildTime string
    // GitCommit is set at build time
    GitCommit string
)
```

## Troubleshooting

### Pre-commit hook not running

Ensure the hook is executable:
```bash
chmod +x .git/hooks/pre-commit
```

### VERSION file not updating

Check if the hook has errors:
```bash
.git/hooks/pre-commit
```

### Wrong version bump

Manually fix the VERSION file:
```bash
./scripts/version.sh set X.Y.Z
git add VERSION
git commit --amend --no-edit
```

### Disable auto-versioning

To skip the pre-commit hook for a single commit:
```bash
git commit --no-verify -m "Your message"
```

To disable permanently:
```bash
rm .git/hooks/pre-commit
```

## Migration from `dev`

The project started with version `dev`. The first commit with the new system will bump to `0.1.0`.

All subsequent commits will follow semantic versioning automatically.
