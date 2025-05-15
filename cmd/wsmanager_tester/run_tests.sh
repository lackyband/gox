#!/bin/bash
set -eo pipefail

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
TEST_TIMEOUT="15m"  # Increased timeout for long-running tests
MAX_PARALLEL=4      # Maximum parallel test runs

# Change to the directory where this script is located
cd "$(dirname "$0")"

echo -e "${YELLOW}=== Starting wsmanager_tester Test Suite ===${NC}"
echo -e "${YELLOW}Running in: $(pwd)${NC}"

# Get list of all test packages
TEST_PACKAGES=(
    "./..."
)

# Get list of all test functions
echo -e "${YELLOW}Discovering tests...${NC}"
TEST_LIST=$(go test -list . "${TEST_PACKAGES[@]}" 2>&1 | grep -v "^ok" | grep -v "^?" | grep -v "^Benchmark" | sort)

if [ -z "$TEST_LIST" ]; then
    echo -e "${RED}No tests found in packages: ${TEST_PACKAGES[*]}${NC}"
    exit 1
fi

# Convert to array
readarray -t TESTS <<< "$TEST_LIST"

echo -e "\n${YELLOW}Found ${#TESTS[@]} test functions:${NC}"
printf '  %s\n' "${TESTS[@]}"

# Function to run a single test
run_test() {
    local test_name=$1
    local output_file=$2
    
    echo -e "${YELLOW}=== RUN   $test_name${NC}"
    echo "Running $test_name..." > "$output_file"
    
    # Create a named pipe for real-time output
    local pipe="${output_file}.pipe"
    mkfifo "$pipe"
    
    # Run test with timeout, tee output to both console and file
    set +e
    (
        timeout --kill-after=30s "$TEST_TIMEOUT" go test -v -run "^${test_name}\\$" "${TEST_PACKAGES[@]}" 2>&1 | \
        tee -a "$output_file" "$pipe" & 
    ) & local test_pid=$!
    
    # Display output in real-time
    cat "$pipe" | while IFS= read -r line; do
        echo "    $line"
    done
    
    # Wait for the test to complete and get the result
    wait $test_pid
    local result=$?
    set -e
    
    # Clean up
    rm -f "$pipe"
    
    if [ $result -eq 0 ]; then
        echo -e "${GREEN}✅ PASSED: $test_name${NC}"
        return 0
    elif [ $result -eq 124 ]; then
        echo -e "${RED}⏱️  TIMEOUT ($TEST_TIMEOUT): $test_name${NC}"
        return 1
    else
        echo -e "${RED}❌ FAILED: $test_name${NC}"
        return 1
    fi
}

# Create temporary directory for test outputs
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

# Run tests sequentially
echo -e "\n${YELLOW}=== Running Tests Sequentially ===${NC}\n"

FAILED_TESTS=()
PASSED_TESTS=()

for test in "${TESTS[@]}"; do
    output_file="${TEMP_DIR}/${test//[^[:alnum:]]/_}.log"
    
    # Run test and capture exit status
    if run_test "$test" "$output_file"; then
        PASSED_TESTS+=("$test")
    else
        FAILED_TESTS+=("$test")
    fi
done

# Print summary
echo -e "\n${YELLOW}=== Test Summary ===${NC}"
echo -e "Total:    ${#TESTS[@]}"
echo -e "${GREEN}Passed:   ${#PASSED_TESTS[@]}${NC}"
echo -e "${RED}Failed:   ${#FAILED_TESTS[@]}${NC}"

# Print failed tests
if [ ${#FAILED_TESTS[@]} -gt 0 ]; then
    echo -e "${RED}Failed tests:${NC}"
    for test in "${FAILED_TESTS[@]}"; do
        echo -e "  - ${RED}$test${NC}"
        echo -e "    ${YELLOW}Output:${NC}"
        cat "${TEMP_DIR}/${test//[^[:alnum:]]/_}.log" | sed 's/^/      /'
        echo
    done
    
    # Save full logs
    echo -e "${YELLOW}Full test logs saved to: $TEMP_DIR${NC}"
    echo -e "${RED}❌ Some tests failed. See above for details.${NC}"
    exit 1
else
    echo -e "${GREEN}✅ All tests passed!${NC}"
    exit 0
fi
