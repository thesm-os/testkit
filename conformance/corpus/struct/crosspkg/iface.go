// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package crosspkg is the fixture for a default that names a symbol outside
// the declaring package.
//
// Two notations, and both are needed. A qualifier resolves against the file's
// own imports, which reads naturally and is what an author writes for a package
// the file already uses. A full import path needs no import at all, which is
// the only thing that works when the package would otherwise be imported solely
// to feed a directive — an unused import does not compile.
//
// [go.thesmos.sh/testkit/conformance/corpus/struct/seed] holds the far side.
//
// Routing is declared once for the package rather than repeated on each
// type: every builder here lands in the same companion package, so a
// per-struct directive is the same statement written N times, and the Nth
// copy is the one that gets forgotten.
//
//testkit:out crosspkgtest/ pkg=crosspkgtest
package crosspkg

import (
	"time"

	"go.thesmos.sh/testkit/conformance/corpus/struct/seed"
)

// Settings draws every default from another package.
//
// Timeout and Window use the qualifier form against imports this file already
// has. Region uses the full-path form for a package this file does not import
// at all, which is the case the notation exists for.
//
//testkit:builder
type Settings struct {
	// Timeout names a constant in a package imported for real code below.
	Timeout time.Duration //testkit:default time.Second

	// Retries names a constant in a package this file imports.
	Retries int //testkit:default seed.Retries

	// Region names a constant by full import path. Nothing in this file
	// imports that package for anything else.
	Region string //testkit:default go.thesmos.sh/testkit/conformance/corpus/struct/seed.Region

	// Deadline exists so the file's `time` import is used by real code and not
	// only by a directive.
	Deadline time.Time
}

// Mirrored seeds from a companion in another package, named by full path.
//
//testkit:builder defaults=go.thesmos.sh/testkit/conformance/corpus/struct/seed.ConfigDefaults
type Mirrored = seed.Config

// Shorthand names the same companion by qualifier rather than by path, which
// is the notation an author reaches for first and the one every field in this
// file already uses.
//
// It exists because the two notations take different routes: a full path names
// itself, while a qualifier is only meaningful against the import block of the
// file that wrote it. The qualifier route was documented and unreachable — no
// fixture wrote one on a `defaults=` key, so nothing established that the
// generator could get from a struct to its file at all.
//
//testkit:builder defaults=seed.ConfigDefaults
type Shorthand = seed.Config

var _ = seed.Region
