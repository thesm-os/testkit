// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package streamtest_test

import (
	"testing"

	"go.thesmos.sh/testkit/gen/suite/testdata/voidpure"
	"go.thesmos.sh/testkit/gen/suite/testdata/voidpure/streamtest"
	"go.thesmos.sh/testkit/suite"
)

func TestStreamContract(t *testing.T) {
	t.Parallel()
	factory := func() voidpure.Stream { return voidpure.NewInMemoryStream() }

	streamtest.AssertStreamContract(t, factory,
		// Sum is Pure-shaped — typed return works.
		streamtest.StreamOnSum(
			suite.AssertDeterministic[voidpure.Stream, voidpure.Digest](3),
		),
		// Close and Reset are Unknown-shaped — custom assertions.
		streamtest.StreamCustom("Reset is idempotent", func(t *testing.T, s voidpure.Stream) {
			s.Reset()
			s.Reset() // must not panic
		}),
		streamtest.StreamCustom("Close is idempotent", func(t *testing.T, s voidpure.Stream) {
			s.Close()
			s.Close() // must not panic
		}),
	)
}
