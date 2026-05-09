// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package storetest_test

import (
	"testing"

	"go.thesmos.sh/testkit/generator/testdata/basic"
	"go.thesmos.sh/testkit/generator/testdata/basic/storetest"
)

// BenchmarkSampler closes the loop on `testkit bench` for the
// //testkit:sample directive. The factory uses
// [basic.NewInMemorySampler] which pre-seeds [basic.SampleKey]() →
// [basic.SampleItem]() — the same pair the directive's call
// expressions resolve to in the generated bench output.
func BenchmarkSampler(b *testing.B) {
	storetest.BenchmarkSamplerContract(b, func() basic.Sampler {
		return basic.NewInMemorySampler()
	})
}
