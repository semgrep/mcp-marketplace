#!/bin/bash
set -e

# Main test runner for all hook tests

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Source common utilities
source "${SCRIPT_DIR}/test_utils.sh"

# Print overall test header
print_separator
echo "Testing Claude Code Hooks"
print_separator
echo "CLAUDE_PLUGIN_ROOT: ${CLAUDE_PLUGIN_ROOT}"

# Track overall test status
ALL_TESTS_PASSED=true

# Run SessionStart hook test
echo ""
if bash "${SCRIPT_DIR}/test_session_start_hook.sh"; then
    :
else
    ALL_TESTS_PASSED=false
fi

# Run PostToolUse hook test
echo ""
if bash "${SCRIPT_DIR}/test_post_tool_use_hook.sh"; then
    :
else
    ALL_TESTS_PASSED=false
fi

# Print final summary
echo ""
print_separator
if [ "$ALL_TESTS_PASSED" = true ]; then
    print_success "All hook tests passed!"
    print_separator
    exit 0
else
    print_error "Some hook tests failed"
    print_separator
    exit 1
fi
