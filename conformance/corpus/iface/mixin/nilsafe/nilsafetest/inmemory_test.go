// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package nilsafetest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/nilsafe"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/nilsafe/nilsafetest"
)

// The first classification check the generator derives, and the one that has to
// refuse the fixture to mean anything.
//
// Every other check is handed a derived value, which is well-formed by
// construction — so passing one to a nil-safety check would prove nothing. The
// generated check supplies its own zeros instead, which is why it takes no
// argument at all.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	nilsafetest.AssertMixedContract(t,
		nilsafetest.MixedModel(),
		nilsafetest.MixedSubject("in-memory", func() nilsafe.Mixed {
			return nilsafetest.NewInMemory()
		}),
		nilsafetest.MixedOnStore("stores a payload it was given", func(
			tb testing.TB, subject nilsafe.Mixed, v *nilsafe.Payload,
		) {
			tb.Helper()
			// The value is built here rather than taken from v, because v is
			// nil: eidos writes no literal for a pointer, so the fixture field
			// is declared and left at its zero.
			//
			// Which means for *this* signature the generated nilsafe check and
			// the smoke call make the same call. It earns its place on a method
			// whose parameters are derivable — a struct, a map, a slice — where
			// smoke passes a real value and only the nilsafe check passes zeros.
			testkit.NoError(tb, subject.Store(tb.Context(), &nilsafe.Payload{Key: "k", Body: "b"}),
				"a non-nil payload is accepted")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	nilsafetest.AssertMixedContract(t,
		nilsafetest.MixedSubject("in-memory", func() nilsafe.Mixed {
			return nilsafetest.NewInMemory()
		}),
		nilsafetest.MixedWithout("Store/smoke"),
		nilsafetest.MixedWithoutDouble(),
	)
}
