#!/bin/bash

# Check if the installed Semgrep version meets the minimum requirement
# This script is shared between Claude and Cursor plugins

set -e

# Determine the script's directory and find the semgrep-version file
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Look for semgrep-version in different locations depending on context
if [ -f "${SCRIPT_DIR}/../../semgrep-version" ]; then
    # Running from template repo
    MIN_VERSION_FILE="${SCRIPT_DIR}/../../semgrep-version"
elif [ -n "${CLAUDE_PLUGIN_ROOT}" ] && [ -f "${CLAUDE_PLUGIN_ROOT}/semgrep-version" ]; then
    # Running from Claude plugin
    MIN_VERSION_FILE="${CLAUDE_PLUGIN_ROOT}/semgrep-version"
elif [ -n "${CURSOR_PLUGIN_ROOT}" ] && [ -f "${CURSOR_PLUGIN_ROOT}/semgrep-version" ]; then
    # Running from Cursor plugin
    MIN_VERSION_FILE="${CURSOR_PLUGIN_ROOT}/semgrep-version"
else
    echo "Error: Could not find semgrep-version file"
    exit 1
fi

MIN_VERSION=$(cat "$MIN_VERSION_FILE")

# Get installed Semgrep version
if ! command -v semgrep &> /dev/null; then
    echo "Error: Semgrep is not installed"
    exit 1
fi

INSTALLED_VERSION=$(semgrep --version 2>/dev/null | head -n1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -n1)

if [ -z "$INSTALLED_VERSION" ]; then
    echo "Error: Could not determine installed Semgrep version"
    exit 1
fi

# Compare versions using sort -V
version_gte() {
    [ "$1" = "$(echo -e "$1\n$2" | sort -V | tail -n1)" ]
}

if version_gte "$INSTALLED_VERSION" "$MIN_VERSION"; then
    echo "Success: Semgrep $INSTALLED_VERSION >= $MIN_VERSION (minimum required)"
    exit 0
else
    echo "Error: Semgrep $INSTALLED_VERSION < $MIN_VERSION (minimum required)"
    echo "Please update Semgrep: brew upgrade semgrep"
    exit 1
fi
