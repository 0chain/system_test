#!/bin/bash
# Pre-compile test binaries for faster test runs
# Usage: bash scripts/compile_tests.sh [suite...]
# Example: bash scripts/compile_tests.sh cli api tokenomics sdk

export PATH=$PATH:/usr/local/go/bin:/root/go/bin
cd /root/Code/system_test

compile_suite() {
    local suite=$1
    local dir=$2
    local outfile=$3
    echo "Compiling $suite tests..."
    cd $dir
    go test -c -o $outfile ./... 2>&1
    if [ $? -eq 0 ]; then
        echo "  -> $outfile compiled successfully ($(ls -lh $outfile | awk '{print $5}'))"
    else
        echo "  -> COMPILATION FAILED for $suite"
    fi
    cd /root/Code/system_test
}

suites=("${@:-cli api tokenomics sdk}")

for suite in "${suites[@]}"; do
    case $suite in
        cli)        compile_suite "CLI" "tests/cli_tests" "cli_tests.test" ;;
        api)        compile_suite "API" "tests/api_tests" "api_tests.test" ;;
        tokenomics) compile_suite "Tokenomics" "tests/tokenomics_tests" "tokenomics_tests.test" ;;
        sdk)        compile_suite "SDK" "tests/sdk_tests" "sdk_tests.test" ;;
        *) echo "Unknown suite: $suite" ;;
    esac
done

echo ""
echo "To run pre-compiled tests:"
echo "  cd tests/cli_tests && ./cli_tests.test -test.v -test.timeout 120m -test.count 1"
echo "  cd tests/api_tests && ./api_tests.test -test.v -test.timeout 60m -test.count 1"
echo "  cd tests/tokenomics_tests && ./tokenomics_tests.test -test.v -test.timeout 90m -test.count 1"
