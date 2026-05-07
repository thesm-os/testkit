#!/usr/bin/env bash
# Release script — tags root + all submodules with GPG signing.
#
# Usage:
#   ./scripts/release.sh v0.7.0
#   ./scripts/release.sh v0.7.0 --push
#
# Opens $EDITOR once for the tag message. All tags (root +
# submodules) share the same message. Failed GPG signing retries
# up to 3 times per tag. Re-run safe: skips existing tags.

set -euo pipefail

VERSION="${1:?Usage: $0 <version> [--push]}"
PUSH="${2:-}"

SUBMODULES=(clitest cmd container gen httptest model oteltest)

BLUE='\033[0;36m'
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m'

MAX_RETRIES=3

tag_with_retry() {
    local tag="$1"
    local msg="$2"
    for attempt in $(seq 1 "$MAX_RETRIES"); do
        if git tag -s "$tag" -m "$msg"; then
            echo -e "  ${GREEN}${tag}${NC}"
            return 0
        fi
        echo -e "  ${RED}${tag} failed (attempt ${attempt}/${MAX_RETRIES})${NC}"
    done
    echo -e "${RED}${tag} failed after ${MAX_RETRIES} attempts — aborting${NC}"
    exit 1
}

echo -e "${BLUE}Releasing ${VERSION}...${NC}"

# Collect the tag message.
if git rev-parse "$VERSION" >/dev/null 2>&1; then
    echo -e "  ${GREEN}${VERSION}${NC} (exists)"
    MSG=$(git tag -l --format='%(contents)' "$VERSION")
else
    # Write template, open editor, read back.
    MSGFILE=$(mktemp)
    printf '%s\n\n' "$VERSION" > "$MSGFILE"
    "${EDITOR:-vi}" "$MSGFILE"
    MSG=$(cat "$MSGFILE")
    rm -f "$MSGFILE"

    if [ -z "$(echo "$MSG" | tr -d '[:space:]')" ]; then
        echo -e "${RED}Empty tag message — aborting${NC}"
        exit 1
    fi

    tag_with_retry "$VERSION" "$MSG"
fi

# Submodule tags — reuse the same message.
for mod in "${SUBMODULES[@]}"; do
    tag="${mod}/${VERSION}"
    if git rev-parse "$tag" >/dev/null 2>&1; then
        echo -e "  ${GREEN}${tag}${NC} (exists)"
    else
        tag_with_retry "$tag" "$MSG"
    fi
done

echo -e "${GREEN}Tagged ${VERSION} + ${#SUBMODULES[@]} submodule tags${NC}"

if [ "$PUSH" = "--push" ]; then
    tags=("$VERSION")
    for mod in "${SUBMODULES[@]}"; do
        tags+=("${mod}/${VERSION}")
    done
    echo -e "${BLUE}Pushing ${#tags[@]} tags...${NC}"
    git push origin "${tags[@]}"
    echo -e "${GREEN}Pushed${NC}"
else
    echo -ne "${YELLOW}Push with: git push origin ${VERSION}"
    for mod in "${SUBMODULES[@]}"; do
        echo -n " ${mod}/${VERSION}"
    done
    echo -e "${NC}"
fi
