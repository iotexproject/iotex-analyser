#!/usr/bin/env bash
# Cut a release: tag main, push the tag, create a GitHub Release with auto-generated notes.
# The existing GitHub Actions workflow builds and pushes the Docker image on tag push.
#
# Usage: ./release.sh <version>     e.g. ./release.sh v1.18.0
set -euo pipefail

NEW_VERSION="${1:-}"

if [ -z "$NEW_VERSION" ]; then
  echo "Usage: $0 <version>   e.g. $0 v1.18.0"
  echo "Last tag: $(git describe --tags --abbrev=0 2>/dev/null || echo '(none)')"
  exit 1
fi

if ! [[ "$NEW_VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-.+)?$ ]]; then
  echo "Error: version must match v*.*.* or v*.*.*-* (got: $NEW_VERSION)"
  exit 1
fi

for cmd in git gh; do
  command -v "$cmd" >/dev/null || { echo "Error: '$cmd' not found in PATH"; exit 1; }
done

if git rev-parse "$NEW_VERSION" >/dev/null 2>&1; then
  echo "Error: tag $NEW_VERSION already exists"
  exit 1
fi

BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [ "$BRANCH" != "main" ]; then
  echo "Error: must be on main (currently on $BRANCH)"
  exit 1
fi

if ! git diff-index --quiet HEAD --; then
  echo "Error: working tree has uncommitted changes"
  git status --short
  exit 1
fi

echo "Fetching origin..."
git fetch origin --tags --quiet

if [ "$(git rev-parse HEAD)" != "$(git rev-parse origin/main)" ]; then
  echo "Error: local main is not in sync with origin/main"
  echo "  local:  $(git rev-parse HEAD)"
  echo "  remote: $(git rev-parse origin/main)"
  exit 1
fi

LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")

echo
echo "Releasing $NEW_VERSION"
if [ -n "$LAST_TAG" ]; then
  echo "Previous release: $LAST_TAG"
  COMMIT_COUNT=$(git rev-list --count "$LAST_TAG..HEAD")
  echo "Commits to include ($COMMIT_COUNT):"
  git log --oneline "$LAST_TAG..HEAD"
else
  echo "(no previous tag found — full history will be included)"
fi
echo

read -r -p "Proceed with release? [y/N] " REPLY
[[ "$REPLY" =~ ^[Yy]$ ]] || { echo "Aborted."; exit 1; }

echo "Creating tag $NEW_VERSION..."
git tag -a "$NEW_VERSION" -m "Release $NEW_VERSION"

echo "Pushing tag to origin..."
git push origin "$NEW_VERSION"

echo "Creating GitHub Release..."
if [ -n "$LAST_TAG" ]; then
  gh release create "$NEW_VERSION" \
    --title "$NEW_VERSION" \
    --generate-notes \
    --notes-start-tag "$LAST_TAG"
else
  gh release create "$NEW_VERSION" \
    --title "$NEW_VERSION" \
    --generate-notes
fi

echo
echo "Done. Released $NEW_VERSION."
echo "Docker image will be built and pushed by GitHub Actions:"
echo "  ghcr.io/iotexproject/iotex-analyser:$NEW_VERSION"
echo "  ghcr.io/iotexproject/iotex-analyser:latest"
