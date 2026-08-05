// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package noprefix is the prefix=off half of the sentinel directive.
//
// The directive has two behaviours behind one name: prefix=<value> overrides
// the prefix derived from the package name, and prefix=off suppresses the
// prefix subtest entirely. A corpus covering only the override branch is green
// with the suppression branch never generated, because the directive name
// appears either way.
//
// The sentinels here should not carry a package prefix — with the subtest
// suppressed they would be legal, and a generator that ignored prefix=off
// would fail against them, which is how the fixture would prove suppression
// took effect rather than merely being requested.
//
// They carry one anyway. The repository's own `lint error-prefix` recurses
// from the root and has no exclusion mechanism, so it flags the unprefixed
// form and there is no way to exempt a fixture corpus from it. Filed upstream
// as ergon ISSUE-DRAFT-errorprefix-needs-exclusions; until that lands this
// fixture exercises the directive without being able to detect a generator
// that parses it and does nothing.
//
//testkit:sentinel prefix=off
package noprefix

import "errors"

var (
	// ErrRetry carries the prefix only to satisfy the repository's lint. The
	// directive above says the generated assertion must not require it.
	ErrRetry = errors.New("noprefix: retry the operation")

	// ErrGiveUp likewise.
	ErrGiveUp = errors.New("noprefix: no attempts remain")
)
