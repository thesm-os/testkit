// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package nilsafetest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/nilsafe"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/nilsafe/nilsafetest"
)

// The fixture that derives nothing at all, and says so.
//
// Store's only parameter is a pointer, and eidos writes no literal for one —
// so every family the rules reached for it was refused and the header lists
// both refusals with what would close them. There is no drop test below
// because there is no generated check to drop.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	nilsafetest.RunMixed(t,
		nilsafetest.MixedHarness[*nilsafetest.InMemory]{Name: "in-memory", New: nilsafetest.NewInMemory},
		nilsafetest.MixedChecks{
			{
				Method: "Store",
				Name:   "accepts-a-payload-it-was-given",
				Claim:  "Store stores a payload it was given",
				Run: func(tb testing.TB, s nilsafe.Mixed, fx nilsafetest.MixedFixture) {
					tb.Helper()
					// The value is built here rather than drawn from the
					// fixture, because the fixture's is nil: eidos writes no
					// literal for a pointer, so the field is declared and left
					// at its zero.
					//
					// Which means for *this* signature the generated nilsafe
					// check and the smoke call make the same call. It earns its
					// place on a method whose parameters are derivable — a
					// struct, a map, a slice — where smoke passes a real value
					// and only the nilsafe check passes zeros.
					testkit.NoError(tb, s.Store(tb.Context(), &nilsafe.Payload{Key: "k", Body: "b"}),
						"a non-nil payload is accepted")
				},
			},
		},
	)
}
