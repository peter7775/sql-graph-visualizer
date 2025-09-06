#!/bin/bash

# Test script for gosec installation methods
# This helps verify the CI installation approach locally

echo "Testing gosec installation methods..."

GOSEC_VERSION="2.18.2"
GOSEC_URL="https://github.com/securecodewarrior/gosec/releases/download/v${GOSEC_VERSION}/gosec_${GOSEC_VERSION}_linux_amd64.tar.gz"
TEST_DIR="/tmp/gosec-test"

# Clean up any existing test directory
rm -rf "$TEST_DIR"
mkdir -p "$TEST_DIR"

echo "Testing Method 1: curl + tar extraction..."
if curl -L "$GOSEC_URL" | tar -xzC "$TEST_DIR"; then
    if [ -f "$TEST_DIR/gosec" ]; then
        chmod +x "$TEST_DIR/gosec"
        echo "Method 1 successful!"
        echo "Version: $($TEST_DIR/gosec --version 2>/dev/null || echo 'version check failed')"
        
        # Test the actual security scan
        echo "Testing security scan..."
        if $TEST_DIR/gosec -exclude=G115 -exclude-dir=internal/application/services/graphql/generated ./... >/dev/null 2>&1; then
            echo "Security scan test passed!"
        else
            echo "WARNING: Security scan test had issues (might be expected without proper Go project)"
        fi
    else
        echo "Method 1 failed - binary not found after extraction"
    fi
else
    echo "Method 1 failed - download/extraction failed"
fi

echo "Testing Method 2: wget approach..."
TEST_DIR2="/tmp/gosec-test2"
rm -rf "$TEST_DIR2"
mkdir -p "$TEST_DIR2"

if wget "$GOSEC_URL" -O "$TEST_DIR2/gosec.tar.gz" && tar -xzf "$TEST_DIR2/gosec.tar.gz" -C "$TEST_DIR2"; then
    if [ -f "$TEST_DIR2/gosec" ]; then
        chmod +x "$TEST_DIR2/gosec"
        echo "Method 2 successful!"
        echo "Version: $($TEST_DIR2/gosec --version 2>/dev/null || echo 'version check failed')"
    else
        echo "Method 2 failed - binary not found after extraction"
    fi
else
    echo "Method 2 failed - download/extraction failed"
fi

# Cleanup
rm -rf "$TEST_DIR" "$TEST_DIR2"

echo "Test Summary:"
echo "  Method 1 (curl+tar): $([ -f "$TEST_DIR/gosec" ] && echo "Success" || echo "Failed")"
echo "  Method 2 (wget+tar): $([ -f "$TEST_DIR2/gosec" ] && echo "Success" || echo "Failed")"

echo ""
echo "If both methods work, the CI pipeline should be able to install gosec successfully!"
echo "If methods fail, check network connectivity and GitHub releases availability."
