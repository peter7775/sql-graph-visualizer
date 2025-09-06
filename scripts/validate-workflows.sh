#!/bin/bash

# GitHub Actions workflow validation script
# This script validates all YAML workflow files for syntax errors

echo "🔍 Validating GitHub Actions workflows..."

WORKFLOWS_DIR=".github/workflows"
TOTAL_FILES=0
VALID_FILES=0
ERRORS=0

if [ ! -d "$WORKFLOWS_DIR" ]; then
    echo "❌ No workflows directory found at $WORKFLOWS_DIR"
    exit 1
fi

# Validate each workflow file
for file in "$WORKFLOWS_DIR"/*.yml "$WORKFLOWS_DIR"/*.yaml; do
    if [ -f "$file" ]; then
        TOTAL_FILES=$((TOTAL_FILES + 1))
        echo -n "Checking $(basename "$file")... "
        
        if python3 -c "
import yaml
try:
    with open('$file', 'r') as f:
        yaml.safe_load(f)
    exit(0)
except Exception as e:
    print('Error:', str(e))
    exit(1)
" 2>/dev/null; then
            echo "✅ Valid"
            VALID_FILES=$((VALID_FILES + 1))
        else
            echo "❌ Invalid YAML syntax"
            ERRORS=$((ERRORS + 1))
        fi
    fi
done

echo ""
echo "📊 Validation Summary:"
echo "  Total files checked: $TOTAL_FILES"
echo "  Valid files: $VALID_FILES"
echo "  Files with errors: $ERRORS"

if [ $ERRORS -eq 0 ]; then
    echo "✅ All workflow files are valid!"
    exit 0
else
    echo "❌ Found $ERRORS workflow files with syntax errors"
    exit 1
fi
