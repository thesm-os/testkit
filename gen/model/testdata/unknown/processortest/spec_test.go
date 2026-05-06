// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package processortest_test

import (
	"fmt"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/gen/model/testdata/unknown"
	"go.thesmos.sh/testkit/gen/model/testdata/unknown/processortest"
	"go.thesmos.sh/testkit/model"
	"go.thesmos.sh/testkit/model/action"
)

func TestInMemoryProcessorModel(t *testing.T) {
	t.Parallel()

	factory := func() unknown.Processor { return unknown.NewInMemoryProcessor() }

	t.Run("auto-derived actions only", func(t *testing.T) {
		t.Parallel()
		// CRUD core with Process(Unknown) skipped by the generator.
		// The auto-derived Reader+Writer actions still run.
		processortest.AssertProcessorModel(t, factory,
			processortest.ProcessorModelReference(factory),
		)
	})

	t.Run("extra action covers Unknown method", func(t *testing.T) {
		t.Parallel()
		// Use ExtraActions to cover the Unknown-shaped Process method.
		// The consumer writes the comparison logic since the generator
		// can't infer it from the signature.
		processortest.AssertProcessorModel(t, factory,
			processortest.ProcessorModelReference(factory),
			processortest.ProcessorModelExtraActions(
				action.Unknown("Process",
					func(rt *rapid.T, sut, ref unknown.Processor) model.ActionResult {
						input := rapid.StringMatching(`[a-z]{1,5}`).Draw(rt, "input")
						count := rapid.IntRange(0, 10).Draw(rt, "count")

						sutOut, sutOK, sutErr := sut.Process(rt.Context(), input, count)
						refOut, refOK, refErr := ref.Process(rt.Context(), input, count)

						if sutErr != refErr {
							return model.ActionResult{Err: fmt.Errorf("Process: SUT err=%v, ref err=%v", sutErr, refErr)}
						}
						if sutOut != refOut || sutOK != refOK {
							return model.ActionResult{Err: fmt.Errorf("Process: SUT=(%q,%v), ref=(%q,%v)", sutOut, sutOK, refOut, refOK)}
						}
						return model.ActionResult{}
					},
				),
			),
		)
	})
}
