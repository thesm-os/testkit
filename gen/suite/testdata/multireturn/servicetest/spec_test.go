// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package servicetest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen/suite/testdata/multireturn"
	"go.thesmos.sh/testkit/gen/suite/testdata/multireturn/servicetest"
	"go.thesmos.sh/testkit/suite"
)

func TestInMemoryServiceContract(t *testing.T) {
	t.Parallel()
	factory := func() multireturn.Service { return multireturn.NewInMemoryService() }

	servicetest.AssertServiceContract(t, factory,
		servicetest.ServicePrePopulate(func(_ context.Context, _ multireturn.Service) {}),
		servicetest.ServiceOnReset(
			suite.AssertLifecycleSucceeds[multireturn.Service](),
		),
		servicetest.ServiceOnStatus(func(t *testing.T, s multireturn.Service) {
			_, _, err := s.Status(t.Context())
			testkit.NoError(t, err, "Status must succeed")
		}),
	)
}
