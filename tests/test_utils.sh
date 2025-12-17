#!/bin/bash

# Common utilities for test scripts

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Set up the environment variable that the hook expects
export CLAUDE_PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-$(cd "$(dirname "$0")/../plugin" && pwd)}"

# Helper function to print test header
print_test_header() {
    local test_name="$1"
    echo ""
    echo "----------------------------------------"
    echo "Test: ${test_name}"
    echo "----------------------------------------"
}

# Helper function to print success message
print_success() {
    local message="$1"
    echo -e "${GREEN}✓ ${message}${NC}"
}

# Helper function to print error message
print_error() {
    local message="$1"
    echo -e "${RED}✗ ${message}${NC}"
}

# Helper function to print warning message
print_warning() {
    local message="$1"
    echo -e "${YELLOW}⚠ ${message}${NC}"
}

# Helper function to print section separator
print_separator() {
    echo "========================================"
}
