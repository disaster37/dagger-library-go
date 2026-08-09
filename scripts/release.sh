#!/bin/bash
set -euo pipefail

NEW_VERSION="${1:-}"
if [ -z "$NEW_VERSION" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 2.0.4"
    exit 1
fi

if ! echo "$NEW_VERSION" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$'; then
    echo "ERROR: version must match X.Y.Z format (e.g. 2.0.4)"
    exit 1
fi

OLD_VERSION=$(head -1 helm/VERSION | tr -d '[:space:]')
echo "Bumping from $OLD_VERSION to $NEW_VERSION"

for module in helm golang image operator-sdk kwok; do
    echo "$NEW_VERSION" > "$module/VERSION"
done

for module in helm golang image operator-sdk kwok; do
    if [ -f "$module/README.md" ]; then
        sed -i "s/@${OLD_VERSION}/@${NEW_VERSION}/g" "$module/README.md"
    fi
done

echo "Done. Review changes, then run:"
echo "  git add -A && git commit -m 'Release $NEW_VERSION' && git tag $NEW_VERSION"
