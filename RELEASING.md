# Release Process

This document describes how to create a new release of LazyHelm.

## Prerequisites

1. You must have push access to the repository
2. All tests must pass
3. The `main` branch should be in a releasable state

## Creating a Release

### 1. Update Version

Make sure your changes are committed and pushed to `main`.

### 2. Create and Push a Tag

```bash
# Create a new tag (use semantic versioning)
git tag -a v1.0.0 -m "Release v1.0.0"

# Push the tag
git push origin v1.0.0
```

### 3. Automated Release

Once the tag is pushed, GitHub Actions will automatically:
- Build binaries for multiple platforms (Linux, macOS, Windows)
- Create a GitHub release with the binaries
- Generate a changelog
- Update Homebrew tap (if configured)

### 4. Verify the Release

1. Go to https://github.com/alessandropitocchi/lazyhelm/releases
2. Verify the release was created
3. Check that all binaries are present
4. Test installation using the install script

## Version Numbering

We follow [Semantic Versioning](https://semver.org/):

- `MAJOR.MINOR.PATCH`
- `MAJOR`: Breaking changes
- `MINOR`: New features (backward compatible)
- `PATCH`: Bug fixes (backward compatible)

Examples:
- `v1.0.0` - Initial stable release
- `v1.1.0` - New features added
- `v1.1.1` - Bug fixes
- `v2.0.0` - Breaking changes

## Manual Release (if needed)

If GitHub Actions fails, you can create a release manually:

```bash
# Install goreleaser (if not already installed)
brew install goreleaser

# Create a snapshot release (test without publishing)
goreleaser release --snapshot --clean

# Create an actual release (requires tag)
goreleaser release --clean
```

## Post-Release

1. Announce the release (optional)
2. Update documentation if needed
3. Close related issues/PRs
