#!/bin/bash
set -e

# Source common utilities
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/test_utils.sh"


# Test: inject-secure-defaults-short
test_inject_secure_defaults() {
    print_test_header "Inject Secure Defaults ($1) Hook"

    local HOOK_INPUT=$(cat <<EOF
{
    "hook_event_name": "MOCK_EVENT"
}
EOF
)

    echo "Prepared hook input JSON"
    echo ""

    local SEMGREP_COMMAND=""

    # Test the semgrep mcp command with mock input
    if [ "$1" = "short" ]; then
        local SEMGREP_COMMAND="semgrep mcp -k inject-secure-defaults-short"
    else
        local SEMGREP_COMMAND="semgrep mcp -k inject-secure-defaults"
    fi
    echo "Running: ${SEMGREP_COMMAND}"

    local EXIT_CODE=0
    if echo "${HOOK_INPUT}" | ${SEMGREP_COMMAND}; then
        print_success "Inject Secure Defaults ($1) Hook executed successfully"
    else
        EXIT_CODE=$?
        print_error "Inject Secure Defaults ($1) Hook test failed with exit code: ${EXIT_CODE}"
        return 1
    fi

    return 0
}

# Run all PostToolUse tests if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    print_separator
    echo "Testing Inject Secure Default Hook"
    print_separator

    ALL_PASSED=true

    if ! test_inject_secure_defaults short; then
        ALL_PASSED=false
    fi

    echo ""

    if ! test_inject_secure_defaults full; then
        ALL_PASSED=false
    fi

    if [ "$ALL_PASSED" = true ]; then
        echo ""
        print_separator
        print_success "All PostToolUse hook tests passed!"
        print_separator
        exit 0
    else
        echo ""
        print_separator
        print_error "Some PostToolUse hook tests failed"
        print_separator
        exit 1
    fi
fi
