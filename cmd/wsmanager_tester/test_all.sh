#!/bin/bash
set -e

echo "Running all wsmanager Go tests with verbose output..."

go test -v ./

if [ $? -eq 0 ]; then
  echo "\n✅ All wsmanager tests passed!"
else
  echo "\n❌ Some tests failed. Please check the output above for details."
  exit 1
fi
