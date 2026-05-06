// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package buffertest_test

import (
	"testing"

	"go.thesmos.sh/testkit/gen/model/testdata/eventually"
	"go.thesmos.sh/testkit/gen/model/testdata/eventually/buffertest"
	"go.thesmos.sh/testkit/model/law"
)

func TestBufferModel(t *testing.T) {
	t.Parallel()

	factory := func() eventually.Buffer { return eventually.NewInMemoryBuffer() }

	t.Run("tier 1 with reference", func(t *testing.T) {
		t.Parallel()
		buffertest.AssertBufferModel(t, factory,
			buffertest.BufferModelReference(factory),
		)
	})

	t.Run("EventuallyAfter Write then Read within 30 steps", func(t *testing.T) {
		t.Parallel()
		// With 2 actions (Read + Write), the probability of not
		// seeing Read in 30 consecutive draws is (1/2)^30 < 1e-9.
		// This proves EventuallyAfter works end-to-end through
		// the generated API and TraceBinder wiring.
		buffertest.AssertBufferModel(t, factory,
			buffertest.BufferModelReference(factory),
			buffertest.BufferModelLaw(&law.EventuallyAfter[eventually.Buffer]{
				Trigger:     "Write",
				Response:    "Read",
				WithinSteps: 30,
			}),
		)
	})
}
