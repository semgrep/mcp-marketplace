#!/bin/bash
set -e

# Source common utilities
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/test_utils.sh"

# Test: SessionStart Hook
test_session_start_hook() {
    print_test_header "SessionStart Hook"

    echo "CLAUDE_PLUGIN_ROOT: ${CLAUDE_PLUGIN_ROOT}"
    echo "Running check_version.sh..."

    chmod +x "${CLAUDE_PLUGIN_ROOT}/scripts/check_version.sh"
    if "${CLAUDE_PLUGIN_ROOT}/scripts/check_version.sh"; then
        print_success "SessionStart hook test passed"
        return 0
    else
        print_error "SessionStart hook test failed"
        return 1
    fi
}

# Run the test if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    print_separator
    echo "Testing SessionStart Hook"
    print_separator

    if test_session_start_hook; then
        exit 0
    else
        exit 1
    fi
fi
