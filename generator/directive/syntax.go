// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package directive

import "strings"

// BundleKeyword is the meta-directive name that introduces a multi-
// directive line:
//
//	//testkit:directive conservative atomic idempotent writer=off timeout=1s
//
// Each whitespace-separated token after the keyword is one directive
// spec. Tokens take three forms:
//
//   - "name"             — directive with no args
//   - "name=value"       — directive with a single arg
//   - "name=v1,v2,v3"    — directive with comma-separated args
//
// The sentinel value "off" is special: "name=off" produces a [Token]
// with Off=true and no args, so consumers can detect opt-outs without
// pattern-matching on a magic string.
const BundleKeyword = "directive"

// OffSentinel is the magic value that flips a token into opt-out mode
// when used in a bundle (e.g. "writer=off" or "mutator=off").
const OffSentinel = "off"

// Token is the parsed form of one directive spec. A standalone
// `//testkit:errors ErrA ErrB` produces a single Token with
// Args=["ErrA","ErrB"]; a bundle line produces one Token per spec.
//
// The parser does not validate Name against the [Registry]; that is
// the caller's responsibility (typically the pipeline's directive
// validator step).
type Token struct {
	// Name is the directive name (no //testkit: prefix).
	Name string

	// Args carries the directive's arguments. For standalone lines the
	// args are space-split; for bundle tokens they are comma-split off
	// the value side of the "=". Empty when the token has no args or
	// when Off=true.
	Args []string

	// Off is true when the source declared "name=off" inside a bundle.
	// The opt-out semantics are directive-specific — the parser only
	// surfaces the flag.
	Off bool
}

// ParseLine parses one //testkit: comment body — that is, the text
// after the "testkit:" prefix. It handles both syntactic forms:
//
//   - Standalone:  "errors ErrA ErrB"   → [Token{Name:"errors", Args:["ErrA","ErrB"]}]
//   - Bundle:      "directive a b=c d=e,f"
//     → [Token{Name:"a"}, Token{Name:"b", Args:["c"]},
//     Token{Name:"d", Args:["e","f"]}]
//
// Whitespace handling matches [strings.Fields]. Returns an empty slice
// for blank input.
//
// The function is pure: same input → same output, no I/O, no
// allocation beyond the result slice.
func ParseLine(body string) []Token {
	fields := strings.Fields(body)
	if len(fields) == 0 {
		return nil
	}
	if fields[0] == BundleKeyword {
		return parseBundle(fields[1:])
	}
	return []Token{{
		Name: fields[0],
		Args: append([]string{}, fields[1:]...),
	}}
}

// parseBundle converts the tokens after the "directive" keyword into
// a slice of [Token]. Each input token is either a bare name or a
// name=value (with optional comma-separated values).
func parseBundle(specs []string) []Token {
	out := make([]Token, 0, len(specs))
	for _, spec := range specs {
		name, val, hasEq := strings.Cut(spec, "=")
		t := Token{Name: name}
		if hasEq {
			if val == OffSentinel {
				t.Off = true
			} else if val != "" {
				t.Args = strings.Split(val, ",")
			}
		}
		out = append(out, t)
	}
	return out
}
