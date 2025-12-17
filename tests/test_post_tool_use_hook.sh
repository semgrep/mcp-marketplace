#!/bin/bash
set -e

# Source common utilities
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/test_utils.sh"

# Test: PostToolUse Hook with Write tool
test_post_tool_use_write() {
    print_test_header "PostToolUse Hook with Write Tool"

    # Create a temporary directory for test files
    local TEST_DIR=$(mktemp -d)
    echo "Created test directory: ${TEST_DIR}"

    # Create a test Python file with a potential security issue
    local TEST_FILE="${TEST_DIR}/test_file.py"
    cat > "${TEST_FILE}" << 'EOF'
import subprocess

def execute_command(user_input):
    # This should trigger a security warning
    subprocess.call(user_input, shell=True)

def safe_function():
    return "safe"
EOF

    echo "Created test file: ${TEST_FILE}"

    # Create mock hook input JSON (simulating what Claude Code sends via stdin)
    local HOOK_INPUT=$(cat <<EOF
{
    "session_id": "test-session-id-12345",
    "transcript_path": "${TEST_DIR}/transcript.jsonl",
    "cwd": "${TEST_DIR}",
    "permission_mode": "default",
    "hook_event_name": "PostToolUse",
    "tool_name": "Write",
    "tool_input": {
        "file_path": "${TEST_FILE}",
        "content": "import subprocess\n\ndef execute_command(user_input):\n    subprocess.call(user_input, shell=True)\n"
    },
    "tool_response": {
        "filePath": "${TEST_FILE}",
        "content": "import subprocess\n\ndef execute_command(user_input):\n    subprocess.call(user_input, shell=True)\n"
    }
}
EOF
)

    echo "Prepared hook input JSON"
    echo ""

    # Test the semgrep mcp command with mock input
    echo "Running: semgrep mcp -k post-tool-cli-scan"
    echo "With mock hook input..."

    local EXIT_CODE=0
    if echo "${HOOK_INPUT}" | semgrep mcp -k post-tool-cli-scan; then
        print_success "PostToolUse hook command executed successfully"
    else
        EXIT_CODE=$?
        # Note: Semgrep might return non-zero if it finds issues, which is expected
        if [ $EXIT_CODE -eq 1 ]; then
            print_warning "PostToolUse hook found security issues (expected behavior)"
            EXIT_CODE=0
        else
            print_error "PostToolUse hook test failed with exit code: ${EXIT_CODE}"
            rm -rf "${TEST_DIR}"
            return 1
        fi
    fi

    # Cleanup
    rm -rf "${TEST_DIR}"
    echo "Cleaned up test directory"
    return 0
}

# Test: PostToolUse Hook with Edit tool
test_post_tool_use_edit() {
    print_test_header "PostToolUse Hook with Edit Tool"

    # Create a temporary directory for test files
    local TEST_DIR=$(mktemp -d)
    echo "Created test directory: ${TEST_DIR}"

    # Create a test Python file with a potential security issue
    local TEST_FILE="${TEST_DIR}/test_file.py"
    cat > "${TEST_FILE}" << 'EOF'
import subprocess

def execute_command(user_input):
    # This should trigger a security warning
    subprocess.call(user_input, shell=True)

def safe_function():
    return "safe"
EOF

    echo "Created test file: ${TEST_FILE}"

    local HOOK_INPUT_EDIT=$(cat <<EOF
{
    "session_id": "test-session-id-67890",
    "transcript_path": "${TEST_DIR}/transcript.jsonl",
    "cwd": "${TEST_DIR}",
    "permission_mode": "default",
    "hook_event_name": "PostToolUse",
    "tool_name": "Edit",
    "tool_input": {
        "file_path": "${TEST_FILE}",
        "old_string": "def safe_function():\n    return \"safe\"",
        "new_string": "def safe_function():\n    return \"modified\""
    },
    "tool_response": {
        "filePath": "${TEST_FILE}",
        "oldString": "def safe_function():\n    return \"safe\"",
        "newString": "def safe_function():\n    return \"modified\"",
        "structuredPatch": [
            {
                "oldStart": 7,
                "oldLines": 2,
                "newStart": 7,
                "newLines": 2,
                "lines": [" def safe_function():", "-    return \"safe\"", "+    return \"modified\""]
            }
        ]
    }
}
EOF
)

    echo "Running: semgrep mcp -k post-tool-cli-scan"
    echo "With mock hook input for Edit tool..."

    local EXIT_CODE=0
    if echo "${HOOK_INPUT_EDIT}" | semgrep mcp -k post-tool-cli-scan; then
        print_success "PostToolUse hook with Edit tool executed successfully"
    else
        EXIT_CODE=$?
        if [ $EXIT_CODE -eq 1 ]; then
            print_warning "PostToolUse hook found security issues (expected behavior)"
            EXIT_CODE=0
        else
            print_error "PostToolUse hook test with Edit failed with exit code: ${EXIT_CODE}"
            rm -rf "${TEST_DIR}"
            return 1
        fi
    fi

    # Cleanup
    rm -rf "${TEST_DIR}"
    echo "Cleaned up test directory"
    return 0
}

# Run all PostToolUse tests if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    print_separator
    echo "Testing PostToolUse Hook"
    print_separator

    ALL_PASSED=true

    if ! test_post_tool_use_write; then
        ALL_PASSED=false
    fi

    echo ""

    if ! test_post_tool_use_edit; then
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
