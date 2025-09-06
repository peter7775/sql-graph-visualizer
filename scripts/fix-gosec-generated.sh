#!/bin/bash

# Fix gosec G104 issues in generated GraphQL files
# This script adds #nosec annotations to ignore unhandled Write() errors in generated code

set -e

GENERATED_FILE="internal/interfaces/graphql/generated/exec.go"

if [[ ! -f "$GENERATED_FILE" ]]; then
    echo "Generated file $GENERATED_FILE not found"
    exit 1
fi

echo "Fixing gosec issues in $GENERATED_FILE..."

# Fix w.Write() calls by adding nosec annotations and explicit error handling
sed -i.bak \
    -e 's/w\.Write(\[\]byte{.*})/_, _ = &/g; s/_, _ = &/_, _ = w.Write(\[\]byte{'"'"'{'"\'"'}) \/\/ #nosec G104/g' \
    -e 's/w\.Write(\[\]byte{.*:.*})/_, _ = w.Write(\[\]byte{'"'"':'"'"'}) \/\/ #nosec G104/g' \
    -e 's/w\.Write(\[\]byte{.*}.*})/_, _ = w.Write(\[\]byte{'"'"'}'"'"'}) \/\/ #nosec G104/g' \
    "$GENERATED_FILE"

# More specific patterns for the exact issues reported
sed -i \
    -e 's/\t\t\tw\.Write(\[\]byte{\x27{\x27})/\t\t\t_, _ = w.Write(\[\]byte{\x27{\x27}) \/\/ #nosec G104/g' \
    -e 's/\t\t\tw\.Write(\[\]byte{\x27:\x27})/\t\t\t_, _ = w.Write(\[\]byte{\x27:\x27}) \/\/ #nosec G104/g' \
    -e 's/\t\t\tw\.Write(\[\]byte{\x27}\x27})/\t\t\t_, _ = w.Write(\[\]byte{\x27}\x27}) \/\/ #nosec G104/g' \
    "$GENERATED_FILE"

echo "Fixed gosec issues in $GENERATED_FILE"

# Remove backup file
rm -f "$GENERATED_FILE.bak"

echo "Done!"
