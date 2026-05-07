#!/usr/bin/env bash
# Release script — creates a signed tag.
#
# Usage:
#   ./scripts/release.sh v0.8.0
#   ./scripts/release.sh v0.8.0 --push
#
# Opens $EDITOR for the tag message. Retries up to 3 times on
# GPG failure. Re-run safe: skips if tag exists.

set -euo pipefail

VERSION="${1:?Usage: $0 <version> [--push]}"
PUSH="${2:-}"

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m'

MAX_RETRIES=3

if git rev-parse "$VERSION" >/dev/null 2>&1; then
    echo -e "${GREEN}${VERSION}${NC} (exists)"
else
    MSGFILE=$(mktemp)
    printf '%s\n\n' "$VERSION" > "$MSGFILE"
    "${EDITOR:-vi}" "$MSGFILE"
    MSG=$(cat "$MSGFILE")
    rm -f "$MSGFILE"

    if [ -z "$(echo "$MSG" | tr -d '[:space:]')" ]; then
        echo -e "${RED}Empty tag message — aborting${NC}"
        exit 1
    fi

    for attempt in $(seq 1 "$MAX_RETRIES"); do
        if git tag -s "$VERSION" -m "$MSG"; then
            echo -e "${GREEN}${VERSION}${NC}"
            break
        fi
        echo -e "${RED}${VERSION} failed (attempt ${attempt}/${MAX_RETRIES})${NC}"
        if [ "$attempt" -eq "$MAX_RETRIES" ]; then
            echo -e "${RED}Aborting after ${MAX_RETRIES} attempts${NC}"
            exit 1
        fi
    done
fi

if [ "$PUSH" = "--push" ]; then
    echo -e "Pushing ${VERSION}..."
    git push origin "$VERSION"
    echo -e "${GREEN}Pushed${NC}"
else
    echo -e "${YELLOW}Push with: git push origin ${VERSION}${NC}"
fi
