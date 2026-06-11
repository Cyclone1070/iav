#!/bin/bash
set -o pipefail

# Go to module root relative to script location
cd "$(dirname "$0")"

echo "=== Running Linters (golangci-lint run) ==="
golangci-lint run
lint_exit=$?

echo "=== Running Go Test Suite (go test ./...)==="
go test ./... 2>&1 | awk '!/^(ok|\?)/'
test_exit=$?
if [ $test_exit -eq 0 ]; then
  echo "0 failures."
fi

# Propagate failure if either step failed
if [ $lint_exit -ne 0 ] || [ $test_exit -ne 0 ]; then
  exit 1
fi

echo "=== All checks passed successfully ==="
