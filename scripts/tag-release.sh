#!/bin/bash

# tag-release.sh - Automatically tag a new release based on semantic versioning
# Usage: ./scripts/tag-release.sh [major|minor|patch]
# Default: patch

set -e

# Get bump type from argument, default to patch
BUMP=${1:-patch}

# Validate bump type
case $BUMP in
    major|minor|patch)
        ;;
    *)
        echo "Error: Invalid bump type '$BUMP'"
        echo "Usage: $0 [major|minor|patch]"
        echo "  major - Increment major version (x.0.0)"
        echo "  minor - Increment minor version (x.y.0)"
        echo "  patch - Increment patch version (x.y.z) [default]"
        exit 1
        ;;
esac

# Get the latest tag, handling case where no tags exist
LATEST_TAG=$(git tag --sort=-version:refname | head -1 | sed 's/^v//' 2>/dev/null || echo "")

if [ -z "$LATEST_TAG" ]; then
    LATEST_TAG="0.0.0"
    echo "No existing tags found, starting from v0.0.0"
fi

echo "Current version: v$LATEST_TAG"

# Parse version components
MAJOR=$(echo $LATEST_TAG | cut -d. -f1)
MINOR=$(echo $LATEST_TAG | cut -d. -f2)
PATCH=$(echo $LATEST_TAG | cut -d. -f3)

# Calculate new version based on bump type
case $BUMP in
    major)
        NEW_VERSION="$((MAJOR + 1)).0.0"
        ;;
    minor)
        NEW_VERSION="$MAJOR.$((MINOR + 1)).0"
        ;;
    patch)
        NEW_VERSION="$MAJOR.$MINOR.$((PATCH + 1))"
        ;;
esac

NEW_TAG="v$NEW_VERSION"

echo "New version: $NEW_TAG ($BUMP bump)"
echo

# Confirm with user
read -p "Create and push tag $NEW_TAG? This will trigger the release workflow. (y/N): " confirm

if [ "$confirm" = "y" ] || [ "$confirm" = "Y" ]; then
    # Create annotated tag
    git tag -a "$NEW_TAG" -m "Release $NEW_TAG"
    echo "✓ Created tag: $NEW_TAG"
    
    # Push the tag to trigger release workflow
    git push origin "$NEW_TAG"
    echo "✓ Pushed tag: $NEW_TAG"
    echo
    echo "Release workflow should now be triggered automatically."
    echo "Check GitHub Actions for build status: https://github.com/YOUR_ORG/MakeMCP/actions"
else
    echo "Tag creation cancelled"
    exit 0
fi