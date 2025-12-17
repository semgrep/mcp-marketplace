#!/bin/bash

# Script to check if plugin files changed and version was bumped appropriately
# Used in CI to enforce version bumping when plugin code changes

set -e

BASE_REF=${1:-"main"}

echo "Comparing changes against origin/$BASE_REF..."

# Check if any files in plugin/ directory changed
PLUGIN_CHANGES=$(git diff --name-only origin/$BASE_REF...HEAD | grep '^plugin/' || true)

if [ -n "$PLUGIN_CHANGES" ]; then
  echo "✓ Plugin files have changed:"
  echo "$PLUGIN_CHANGES"
  echo ""

  # Check if plugin.json exists in base branch
  if git cat-file -e origin/$BASE_REF:plugin/.claude-plugin/plugin.json 2>/dev/null; then
    echo "Checking for version bump in existing plugin..."

    # Check if plugin.json version changed
    VERSION_CHANGED=$(git diff origin/$BASE_REF...HEAD -- plugin/.claude-plugin/plugin.json | grep '"version"' || true)

    if [ -z "$VERSION_CHANGED" ]; then
      echo "❌ ERROR: Plugin files were modified but version was not bumped!"
      echo ""
      echo "Please update the version in: plugin/.claude-plugin/plugin.json"
      echo ""
      echo "Changed files:"
      echo "$PLUGIN_CHANGES"
      exit 1
    else
      echo "✓ Plugin version has been updated:"
      echo "$VERSION_CHANGED"

      # Extract and display old and new versions
      OLD_VERSION=$(git show origin/$BASE_REF:plugin/.claude-plugin/plugin.json | grep '"version"' | sed 's/.*"version": "\(.*\)".*/\1/')
      NEW_VERSION=$(cat plugin/.claude-plugin/plugin.json | grep '"version"' | sed 's/.*"version": "\(.*\)".*/\1/')
      echo ""
      echo "Version change: $OLD_VERSION → $NEW_VERSION"
    fi
  else
    echo "✓ New plugin detected - checking that version is set..."

    # For new plugins, just verify a version exists
    NEW_VERSION=$(cat plugin/.claude-plugin/plugin.json | grep '"version"' | sed 's/.*"version": "\(.*\)".*/\1/' || true)

    if [ -z "$NEW_VERSION" ]; then
      echo "❌ ERROR: plugin.json must have a version field!"
      exit 1
    else
      echo "✓ Plugin version is set to: $NEW_VERSION"
    fi
  fi
else
  echo "✓ No plugin files changed, version bump not required"
fi
