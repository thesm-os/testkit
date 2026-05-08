// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package servicetest_test

import (
	"testing"

	"go.thesmos.sh/testkit/gen/testdata/multireturns"
	"go.thesmos.sh/testkit/gen/testdata/multireturns/servicetest"
)

func inmemoryFactory() multireturns.Service { return multireturns.NewInMemoryService() }
func stubFactory() multireturns.Service {
	return servicetest.NewServiceStub(nil, servicetest.ServiceStubDelegateTo(inmemoryFactory()))
}
func stubBenchFactory() multireturns.Service {
	return servicetest.NewServiceStub(nil, servicetest.ServiceStubBenchMode(), servicetest.ServiceStubDelegateTo(inmemoryFactory()))
}

func TestServiceContract_InMemory(t *testing.T) {
	t.Parallel()
	servicetest.AssertServiceContract(t, inmemoryFactory)
}
func BenchmarkServiceContract_InMemory(b *testing.B) {
	servicetest.BenchmarkServiceContract(b, inmemoryFactory)
}
func TestServiceContract_StubDelegateTo(t *testing.T) {
	t.Parallel()
	servicetest.AssertServiceContract(t, stubFactory)
}
func BenchmarkServiceContract_StubDelegateTo(b *testing.B) {
	servicetest.BenchmarkServiceContract(b, stubFactory)
}
func BenchmarkServiceContract_StubBenchMode(b *testing.B) {
	servicetest.BenchmarkServiceContract(b, stubBenchFactory)
}
