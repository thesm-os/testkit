// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package processortest_test

import (
	"testing"

	"go.thesmos.sh/testkit/gen/suite/testdata/mixed"
	"go.thesmos.sh/testkit/gen/suite/testdata/mixed/processortest"
	"go.thesmos.sh/testkit/suite"
)

func TestInMemoryProcessorContract(t *testing.T) {
	t.Parallel()
	factory := func() mixed.Processor { return mixed.NewInMemoryProcessor() }

	processortest.AssertProcessorContract(t, factory,
		processortest.ProcessorOnProcess(
			suite.AssertWriteSucceeds[mixed.Processor, []byte]([]byte("hello")),
		),
		processortest.ProcessorOnLegacyProcess(
			suite.AssertWriteSucceeds[mixed.Processor, []byte]([]byte("legacy")),
		),
		processortest.ProcessorOnDescribe(
			suite.AssertDeterministic[mixed.Processor, string](3),
		),
	)
}
