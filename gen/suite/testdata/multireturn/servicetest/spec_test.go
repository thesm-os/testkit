// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package servicetest_test

import (
	"testing"

	"go.thesmos.sh/testkit/gen/suite/testdata/multireturn"
	"go.thesmos.sh/testkit/gen/suite/testdata/multireturn/servicetest"
)

func TestInMemoryServiceContract(t *testing.T) {
	t.Parallel()
	factory := func() multireturn.Service { return multireturn.NewInMemoryService() }

	servicetest.AssertServiceContract(t, factory,
		servicetest.PrePopulate(func(_ testing.TB, _ multireturn.Service) {}),
	)
}
