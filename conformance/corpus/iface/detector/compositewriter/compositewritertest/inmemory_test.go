// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package compositewritertest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/compositewriter"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/compositewriter/compositewritertest"
)

// A write returning a receipt earns the zero-value check that a plain write
// cannot: there is a result beside the error, so the two can disagree.
//
// Reaching that check needs an input the subject refuses, and the derivation
// cannot invent one — every value it writes down is well-formed by construction.
// This is the case <Iface>WithFixture exists for, and the whole of what it is
// asked to supply.
func TestCompositeWriterContract(t *testing.T) {
	t.Parallel()

	compositewritertest.AssertCompositeWriterContract(t,
		compositewritertest.CompositeWriterModel(),
		compositewritertest.CompositeWriterSubject("in-memory",
			func() compositewriter.CompositeWriter {
				return compositewritertest.NewInMemory()
			}),
		compositewritertest.CompositeWriterWithFixture(refusedFixture()),
		compositewritertest.CompositeWriterOnStore("returns the receipt it stored", func(
			tb testing.TB, subject compositewriter.CompositeWriter, v compositewriter.Value,
		) {
			tb.Helper()
			got, err := subject.Store(tb.Context(), v)
			testkit.NoError(tb, err, "a value with a key is accepted")
			testkit.Equal(tb, got.Key, v.Key, "the receipt names what was written")
			testkit.False(tb, got.Rev == "", "and carries the revision the store assigned")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestCompositeWriterContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	compositewritertest.AssertCompositeWriterContract(t,
		compositewritertest.CompositeWriterSubject("in-memory",
			func() compositewriter.CompositeWriter {
				return compositewritertest.NewInMemory()
			}),
		compositewritertest.CompositeWriterWithFixture(refusedFixture()),
		compositewritertest.CompositeWriterWithout("Store/smoke"),
		compositewritertest.CompositeWriterWithoutDouble(),
	)
}

// refusedFixture supplies the value this subject rejects.
//
// A generator derives plausible strings and every plausible key is a key, so the
// zero-value check has no way to reach the error path and says so by name rather
// than passing. The empty key is the one value this store refuses.
func refusedFixture() compositewritertest.CompositeWriterFixture {
	f := compositewritertest.DefaultCompositeWriterFixture()
	f.VOther = compositewriter.Value{}
	return f
}
