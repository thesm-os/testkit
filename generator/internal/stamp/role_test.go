// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stamp_test

import (
	"testing"

	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/internal/stamp"
)

// The role word reads back verbatim, unknown words included.
//
// Validation belongs to the readers, which refuse an unknown role by
// name against their own tables. A reader that filtered here would give
// the vocabulary a second home and turn "this role is not one we know"
// into "this field has no role", which is a different derivation and a
// worse diagnostic.
//
//nolint:thelper // the case body is the test, not a helper; see default_test.go
func TestRoleOfReadsWhatWasWritten(t *testing.T) {
	t.Parallel()

	testkit.TableTest(t, []roleCase{
		{"a role the rules tables know", "key"},
		{"another", "payload"},
		{"one nothing knows, carried anyway", "sourdough"},
	}, func(t *testing.T, tc roleCase) {
		bag := sdk.NewBag()
		stamp.MetaRole.Set(bag, tc.role, "test")

		testkit.Equal(t, stamp.RoleOf(bag), tc.role,
			"the role names the pool a field opens, and naming is the whole job")
	})
}

// An unroled declaration reads as empty, which is what draws no pool.
func TestUnroledReadsEmpty(t *testing.T) {
	t.Parallel()

	testkit.Equal(t, stamp.RoleOf(nil), "", "an unroled field draws no pool")
	testkit.Equal(t, stamp.RoleOf(sdk.NewBag()), "",
		"and a bag written to by some other plugin is still unroled")
}

// roleCase is one written role word.
type roleCase struct {
	name string
	role string
}

func (c roleCase) Name() string { return c.name }
