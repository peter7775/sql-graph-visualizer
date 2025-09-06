#!/bin/bash

# Script to update MIT license headers to Dual License (AGPL + Commercial) headers
# in all Go files

NEW_HEADER="/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under a Dual License:
 * - AGPL-3.0 for open source use (see LICENSE file)
 * - Commercial License for business use (contact: petrstepanek99@gmail.com)
 *
 * This software contains patent-pending innovations in database analysis
 * and graph visualization. Commercial use requires separate licensing.
 */"

# Find all Go files and update headers
find . -name "*.go" -type f | while read -r file; do
    echo "Processing: $file"
    
    # Check if file has old MIT header
    if grep -q "This source code is licensed under the MIT license" "$file"; then
        echo "  → Updating MIT header to Dual License"
        
        # Create temp file with new header
        temp_file=$(mktemp)
        
        # Add new header
        echo "$NEW_HEADER" > "$temp_file"
        echo "" >> "$temp_file"
        
        # Add everything after the old header (skip first 6 lines which contain old header)
        tail -n +7 "$file" >> "$temp_file"
        
        # Replace original file
        mv "$temp_file" "$file"
        
    elif head -n 6 "$file" | grep -q "Copyright.*Petr Miroslav Stepanek" && ! grep -q "Dual License" "$file"; then
        echo "  → Updating existing header to Dual License"
        
        # Create temp file with new header  
        temp_file=$(mktemp)
        
        # Add new header
        echo "$NEW_HEADER" > "$temp_file"
        echo "" >> "$temp_file"
        
        # Add everything after the old header (skip first 6 lines)
        tail -n +7 "$file" >> "$temp_file"
        
        # Replace original file
        mv "$temp_file" "$file"
        
    elif ! head -n 10 "$file" | grep -q "Copyright"; then
        echo "  → Adding new Dual License header"
        
        # Create temp file with new header
        temp_file=$(mktemp)
        
        # Add new header
        echo "$NEW_HEADER" > "$temp_file"
        echo "" >> "$temp_file"
        
        # Add original file content
        cat "$file" >> "$temp_file"
        
        # Replace original file
        mv "$temp_file" "$file"
        
    else
        echo "  → Already has correct header, skipping"
    fi
done

echo ""
echo "✅ License header update completed!"
echo ""
echo "Updated files with new Dual License header:"
echo "- AGPL-3.0 for open source use"
echo "- Commercial License for business use"
echo "- Patent-pending innovations notice"
echo ""
echo "Next step: git add . && git commit -m 'Update license headers to Dual License'"
