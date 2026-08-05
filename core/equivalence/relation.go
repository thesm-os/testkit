// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package equivalence

import "github.com/google/go-cmp/cmp"

// Relation is one equivalence rule. Each Relation contributes a
// slice of [cmp.Option] values that the [Chain] composes into one
// comparison run. Relations identify themselves via [Relation.Name]
// for diagnostic and presets-registry use.
//
// Relations are stateless and safe for concurrent use; consumers
// share a single instance across any number of Chains.
type Relation interface {
	// Name returns a stable identifier ("strict", "id-field",
	// "timestamp", "custom:my-rule"). Used in diff messages and
	// preset lookups.
	Name() string

	// Options returns the cmp options this relation contributes.
	// May return nil for relations that rely solely on go-cmp's
	// default deep equality (e.g., [Strict]).
	Options() []cmp.Option
}
