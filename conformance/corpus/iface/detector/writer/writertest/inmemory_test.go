// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package writertest_test

import (
	"testing"

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
