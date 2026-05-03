// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package processortest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen/suite/testdata/mixed"
	"go.thesmos.sh/testkit/gen/suite/testdata/mixed/processortest"
)

func TestInMemoryProcessorContract(t *testing.T) {
	t.Parallel()
	factory := func() mixed.Processor { return mixed.NewInMemoryProcessor() }

	processortest.AssertProcessorContract(t, factory,
		processortest.ProcessorOnProcess(
			testkit.AssertWriteSucceeds[mixed.Processor, []byte]([]byte("hello")),
		),
		processortest.ProcessorOnLegacyProcess(
			testkit.AssertWriteSucceeds[mixed.Processor, []byte]([]byte("legacy")),
		),
		processortest.ProcessorOnDescribe(
			testkit.AssertDeterministic[mixed.Processor, string](3),
		),
	)
}
