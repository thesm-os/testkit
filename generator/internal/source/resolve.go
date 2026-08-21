// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package source

import (
	"errors"
	"fmt"
	"strings"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/sdk"
)

// Resolve splits a declared value into the package it names and the symbol
// within it, or reports an empty package for a plain literal.
//
// Two notations, told apart by whether the qualifier holds a slash:
//
//	time.Second                     -> resolved against file's import block
//	example.com/seed.DefaultRegion  -> a full import path, needing no import
//	gopkg.in/yaml.v3.Marshal        -> also a full path; the dots before the
//	                                   last one belong to the path
//
// The second notation exists because an import written only to feed a directive
// is an unused import, which does not compile. Without it, a value can only
// name a package the file already uses for real code.
//
// A literal — quoted, numeric, a keyword — has no qualifier and passes through
// untouched, checked only for the quoting fault that would swallow the rest of
// the generated line.
//
// A nil file resolves no qualifier at all. That is what a positionless
// declaration gets, and it reports [golang.ErrUnresolvedQualifier] rather than
// guessing against some other file's imports — the shape that made this
// function panic when a caller had no declaration to hand.
func Resolve(file *sdk.File, value string) (pkg, symbol string, err error) {
	if malformed := golang.IsWellFormedLiteral(value); malformed != nil {
		return "", "", fmt.Errorf("%q: %w", value, malformed)
	}
	if !qualified(value) {
		return "", value, nil
	}
	ref, err := resolveRef(file, value)
	if err != nil {
		return "", "", err
	}
	ext, ok := ref.(*sdk.ExternalRef)
	if !ok {
		// A bare identifier — a constant the declaring package owns. It renders
		// as itself and registers no import, which is what an empty source
		// package asks [golang.RefFor] for: the stamp is read by whichever file
		// later renders it, and that file is not known here.
		return "", value, nil
	}
	return ext.Package, ext.Name, nil
}

// resolveRef hands value to whichever of eidos's two rules its notation calls
// for.
//
// The two split on opposite dots, and that is the whole distinction. An import
// path may hold dots, so the full-path form splits from the right; a Go
// qualifier is one identifier and cannot hold one, so the source form splits
// from the left. Reading source text with the right-hand rule manufactures a
// qualifier that is not an identifier.
//
// A slash before the last dot is what picks the first: no Go qualifier holds
// one, and every import path worth writing in a directive does.
func resolveRef(file *sdk.File, value string) (sdk.Ref, error) {
	if dot := strings.LastIndex(value, "."); dot > 0 && strings.Contains(value[:dot], "/") {
		ref, err := golang.RefForQualified(value, "")
		if err != nil {
			return nil, fmt.Errorf("%q: %w", value, err)
		}
		return ref, nil
	}
	ref, err := golang.ResolveQualified(file, value, "")
	switch {
	case errors.Is(err, golang.ErrUnresolvedQualifier):
		// The full-path form is the way out of this, and an author who reached
		// for a qualifier has no reason to know it exists.
		_, symbol := golang.QualifierOf(value)
		return nil, fmt.Errorf(
			"%q: %w; write the full path as <import/path>.%s if the package is "+
				"imported only for this directive", value, err, symbol,
		)
	case err != nil:
		return nil, fmt.Errorf("%q: %w", value, err)
	}
	return ref, nil
}

// qualified reports whether value reads as a package-qualified symbol rather
// than as a literal. A quoted string or a number can hold a dot without naming
// anything.
//
// A leading dot is a number too: `.5` is a legal Go float literal and Go's own
// scanner reads it as one. Reading it as a qualifier splits it into an empty
// qualifier and the symbol `5`, which matches every un-aliased import — so the
// first import in the file wins and the generated builder says `http.5`.
func qualified(value string) bool {
	if value == "" || value[0] == '"' || value[0] == '`' {
		return false
	}
	c := value[0]
	return (c < '0' || c > '9') && c != '-' && c != '+' && c != '.'
}
