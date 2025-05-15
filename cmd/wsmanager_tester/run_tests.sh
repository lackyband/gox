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
    
    echo "Running $test_name..." > "$output_file"
    
    # Run test with timeout
    set +e
    timeout --kill-after=30s "$TEST_TIMEOUT" go test -v -run "^${test_name}\$" "${TEST_PACKAGES[@]}" >> "$output_file" 2>&1
    local result=$?
    set -e
    
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

# Run tests in parallel
echo -e "\n${YELLOW}=== Running Tests (max $MAX_PARALLEL in parallel) ===${NC}\n"

PIDS=()
RESULTS=()
FAILED_TESTS=()
PASSED_TESTS=()

for test in "${TESTS[@]}"; do
    # Wait if we've reached max parallel
    while [ $(jobs -p | wc -l) -ge $MAX_PARALLEL ]; do
        sleep 0.1
    done
    
    # Run test in background
    output_file="${TEMP_DIR}/${test//[^[:alnum:]]/_}.log"
    run_test "$test" "$output_file" & 
    PIDS+=($!)
    RESULTS+=("$output_file")
    
    # Store test name and result
    if [ $? -eq 0 ]; then
        PASSED_TESTS+=("$test")
    else
        FAILED_TESTS+=("$test")
    fi
done

# Wait for all tests to complete
wait "${PIDS[@]}" 2>/dev/null || true

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
