// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package compositewritertest_test

import (
	"testing"

	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/compositewriter"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/compositewriter/compositewritertest"
)

// The generated contract, run against the in-memory subject. The fixture
// exists to pin the composite-writer detector — a key beside a value, an
// error and nothing else — and the identity gate holds the stamp to the
// directory's name, so the wiring here stays the minimal consumer's.
func TestCompositeWriterContract(t *testing.T) {
	t.Parallel()

	compositewritertest.AssertCompositeWriterContract(t,
		compositewritertest.CompositeWriterModel(),
		compositewritertest.CompositeWriterSubject("in-memory",
			func() compositewriter.CompositeWriter {
				return compositewritertest.NewInMemory()
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
		compositewritertest.CompositeWriterWithout("Set/smoke"),
		compositewritertest.CompositeWriterWithoutDouble(),
	)
}
