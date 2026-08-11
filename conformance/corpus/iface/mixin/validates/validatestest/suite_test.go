// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package validatestest_test

import (
	"testing"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/validates"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/validates/validatestest"
)

// The whole wiring a consumer writes: one subject, no options.
//
// That is the acceptance test for the generator. Every value the checks need is
// derived, so an option here would be a derivation that had not been done — and
// the run through the double is derived too, from the //testkit:stub the same
// interface declares.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	validatestest.AssertMixedContract(t,
		validatestest.MixedSubject("in-memory", func() validates.Mixed {
			return validates.NewInMemory()
		}),
	)
}
