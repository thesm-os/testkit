// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package writertest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/writer"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/writer/writertest"
)

// The one shape that needs no seed option: Put is classified writer, so the
// harness populates every subject through the interface's own method before each
// check runs.
//
// No fixture either. What Put is handed is derived from Value's own fields, and
// the whole wiring is one Subject.
func TestWriterContract(t *testing.T) {
	t.Parallel()

	writertest.AssertWriterContract(t,
		writertest.WriterSubject("in-memory", func() writer.Writer {
			return writertest.NewInMemory()
		}),
	)
}

// That the write lands is the shape's own law and no part of its signature: Put
// reports whether it failed, never what it stored, so observing the effect needs
// a method the interface does not declare.
func TestPutStoresUnderTheValuesOwnKey(t *testing.T) {
	t.Parallel()

	s := writertest.NewInMemory()
	want := writer.Value{Key: "k", Body: "b"}
	testkit.NoError(t, s.Put(t.Context(), want), "storing a value succeeds")

	got, ok := s.Stored("k")
	testkit.True(t, ok, "the value is found under its own key")
	testkit.Equal(t, got, want, "and comes back whole")
}

// Declining the double is separate from dropping a check.
func TestWriterContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	writertest.AssertWriterContract(t,
		writertest.WriterSubject("in-memory", func() writer.Writer {
			return writertest.NewInMemory()
		}),
		writertest.WriterWithout("Put/smoke"),
		writertest.WriterWithoutDouble(),
	)
}
