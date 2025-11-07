# Publishing LazyHelm

Quick guide to publish your project and make it installable.

## 1. Push to GitHub

```bash
# Add all files
git add .

# Commit
git commit -m "feat: initial release with installation support"

# Push to GitHub
git push origin main
```

## 2. Create Your First Release

```bash
# Create and push a tag
git tag -a v0.1.0 -m "Initial release v0.1.0"
git push origin v0.1.0
```

GitHub Actions will automatically:
- Build binaries for Linux, macOS, Windows
- Create a GitHub release
- Upload all binaries

## 3. Users Can Now Install

Once the release is published, users can install via:

### Option 1: Go Install (Easiest)
```bash
go install github.com/alessandropitocchi/lazyhelm/cmd/lazyhelm@latest
```

### Option 2: Install Script
```bash
curl -sSL https://raw.githubusercontent.com/alessandropitocchi/lazyhelm/main/install.sh | bash
```

### Option 3: Manual Download
- Go to GitHub Releases
- Download the binary for their platform
- Move to `/usr/local/bin/`

## 4. (Optional) Homebrew Tap

For macOS users, you can create a Homebrew tap:

```bash
# Create a new repo: homebrew-tap
# The goreleaser will automatically update it on each release
```

Then users can install with:
```bash
brew tap alessandropitocchi/tap
brew install lazyhelm
```

## Testing Before Publishing

Test the release process locally:

```bash
# Install goreleaser
brew install goreleaser

# Test release without publishing
goreleaser release --snapshot --clean

# Check binaries in dist/
ls -la dist/
```

## Updating the Release

To create a new release:

```bash
# Make your changes
git add .
git commit -m "feat: add new feature"
git push

# Create new tag (bump version)
git tag -a v0.2.0 -m "Release v0.2.0 with new features"
git push origin v0.2.0
```

## Repository Settings

Make sure your GitHub repository has:
1. **Actions enabled** (Settings → Actions → Allow all actions)
2. **Write permissions for GITHUB_TOKEN** (Settings → Actions → Workflow permissions → Read and write)

## First Time Setup Checklist

- [x] README.md created
- [x] LICENSE file present
- [x] install.sh script created and executable
- [x] Makefile for easy builds
- [x] .goreleaser.yaml configured
- [x] GitHub Actions workflows (.github/workflows/)
- [x] .gitignore configured
- [ ] Push to GitHub
- [ ] Create first release tag
- [ ] Verify release on GitHub
- [ ] Test installation
- [ ] Share with the community!

## Support

After publishing:
- Monitor GitHub Issues for bug reports
- Update README with screenshots/GIFs
- Add to awesome-go or similar lists
- Share on Reddit, Twitter, etc.
