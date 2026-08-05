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
// The sentinels here deliberately carry no package prefix. That is what makes
// the fixture able to detect a failure rather than merely exercise a path: with
// the subtest suppressed they are legal, and a generator that parsed prefix=off
// and ignored it would emit the assertion and fail against them.
//
// The repository's own `lint error-prefix` would flag them, correctly by its
// own rules. The corpus is exempted from it in .ergon.yaml, because a fixture
// corpus contains violating patterns on purpose.
//
//testkit:sentinel prefix=off
package noprefix

import "errors"

var (
	// ErrRetry has no "noprefix: " prefix, and must not be required to.
	ErrRetry = errors.New("retry the operation")

	// ErrGiveUp likewise.
	ErrGiveUp = errors.New("no attempts remain")
)
