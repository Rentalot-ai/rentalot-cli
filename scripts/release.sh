#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-}"
if [[ -z "$VERSION" ]]; then
  echo "Usage: $0 <version>"
  echo "  e.g. $0 v0.1.0"
  exit 1
fi

# Strip leading v for bare semver
BARE_VERSION="${VERSION#v}"
# Ensure tag has v prefix
TAG="v${BARE_VERSION}"

echo "==> Pre-flight checks..."
if [[ -n "$(git status --porcelain)" ]]; then
  echo "Error: working tree is dirty"
  exit 1
fi

echo "==> Running tests..."
make test

echo "==> Running lint..."
make lint

echo "==> Building..."
make build

# If chlog is installed, use it for changelog management
if command -v chlog >/dev/null 2>&1; then
  echo "==> Checking unreleased entries..."
  if chlog show unreleased 2>/dev/null | grep -q .; then
    echo "==> Stamping changelog: ${BARE_VERSION}..."
    chlog release "${BARE_VERSION}"

    echo "==> Syncing CHANGELOG.md..."
    chlog sync

    echo "==> Committing changelog..."
    git add CHANGELOG.yaml CHANGELOG.md
    git commit -m "release: ${TAG}"
  else
    echo "==> No unreleased entries, skipping changelog stamp"
  fi
else
  echo "==> chlog not installed, skipping changelog management"
fi

echo "==> Tagging ${TAG}..."
git tag -a "${TAG}" -m "Release ${TAG}"

echo "==> Pushing to origin..."
git push origin main
git push origin "${TAG}"

echo ""
echo "Done! ${TAG} tagged and pushed."
echo ""
echo "Next steps:"
echo "  Watch the release:   gh run watch"
echo "  View release:        gh release view ${TAG}"
