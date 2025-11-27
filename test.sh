#!/bin/bash

echo "=========================================="
echo "Running Tinydom WASM Tests..."
echo "=========================================="

# Check if wasmbrowsertest is installed
if ! command -v wasmbrowsertest &> /dev/null; then
    echo "⚠️  wasmbrowsertest not found. Install it with:"
    echo "   go install github.com/agnivade/wasmbrowsertest@latest"
    echo "   export PATH=\$PATH:\$(go env GOPATH)/bin"
    exit 1
fi

# Run WASM tests and filter out noise
GOOS=js GOARCH=wasm go test -v -tags wasm -cover 2>&1 | \
  grep -v "ERROR: could not unmarshal event" | \
  grep -v "call to released function"

# Capture the exit code from go test (before grep)
WASM_EXIT_CODE=${PIPESTATUS[0]}

if [ $WASM_EXIT_CODE -ne 0 ]; then
    echo ""
    echo "❌ WASM tests failed"
    exit 1
fi

echo ""
echo "✅ All tests passed!"
