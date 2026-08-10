#!/usr/bin/env bash
# Concatenate every non-test source file under a directory, for reading a
# subsystem end to end.
#
# Usage:
#   ./scripts/concat.sh [--code|--outline] [--go] [dir]
#
#   (default)   everything, verbatim                     ~289k chars
#   --code      drop whole-line comments and blanks      ~133k  (-54%)
#   --outline   declarations only, no bodies             ~ 43k  (-85%)
#   --go        skip templates, Go only                        (-31%)
#
# --code keeps `//go:` and `//nolint` lines: those are compiler and linter
# input rather than prose, and dropping them changes what the file means.
#
# The docblocks this repo carries are where the reasoning lives, so --code
# trades away most of why the code is the way it is. Reach for it when the
# question is "what does this do", not "is this right".
set -euo pipefail

mode=full
templates=1

while [[ $# -gt 0 ]]; do
	case "$1" in
	--code) mode=code ;;
	--outline) mode=outline ;;
	--go) templates=0 ;;
	-*)
		echo "unknown flag: $1" >&2
		exit 2
		;;
	*) break ;;
	esac
	shift
done

root="${1:-generator}"

names=(-name '*.go')
if [[ $templates -eq 1 ]]; then
	names+=(-o -name '*.tmpl')
fi

find "$root" -type f \( "${names[@]}" \) \
	! -name '*_test.go' \
	! -path '*/testdata/*' |
	sort |
	while read -r file; do
		printf '\n===== %s =====\n\n' "$file"
		case "$mode" in
		full) cat "$file" ;;
		code)
			grep -vE '^[[:space:]]*//([^g]|g[^o]|$)' "$file" |
				grep -vE '^[[:space:]]*//$' |
				grep -vE '^[[:space:]]*$' || true
			;;
		outline)
			grep -E '^(package |func |type |const |var )' "$file" || true
			;;
		esac
	done
