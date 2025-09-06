#!/bin/bash

# Security check script for mysql-graph-visualizer
# This script runs gosec with appropriate exclusions for generated code

echo "🔒 Running security analysis with gosec..."

# Run gosec with exclusions for generated code and false positives
gosec -exclude=G115 -exclude-dir=internal/application/services/graphql/generated ./...

EXIT_CODE=$?

if [ $EXIT_CODE -eq 0 ]; then
    echo "✅ Security scan completed successfully - No issues found!"
else
    echo "⚠️  Security scan found issues. Please review and fix before deployment."
fi

exit $EXIT_CODE
