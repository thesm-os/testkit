// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stamp_test

import (
	"testing"

	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/internal/stamp"
)

// A written stamp reads back as itself, verbatim.
//
// Verbatim is the whole contract: the directive's argument is Go source
// and nothing here parses it, so a reader that trimmed, unquoted or
// normalised would hand a template something the author did not write.
//
// The case body declines a helper mark: TableTest calls it from inside
// its own t.Run, so t.Helper() would report each failure at the
// runner's call site rather than at the assertion line that failed.
//
//nolint:thelper // the case body is the test, not a helper; see above
func TestDefaultOfReadsWhatWasWritten(t *testing.T) {
	t.Parallel()

	testkit.TableTest(t, []stampCase{
		{"a quoted string keeps its quotes", `"localhost"`},
		{"a number stays unparsed", "8080"},
		{"a keyword is a value like any other", "nil"},
		{"an explicit zero is not an absent stamp", "0"},
	}, func(t *testing.T, tc stampCase) {
		bag := sdk.NewBag()
		stamp.MetaDefault.Set(bag, tc.value, "test")

		testkit.Equal(t, stamp.DefaultOf(bag), tc.value,
			"the stamp reaches a template as the author spelled it")
	})
}

// The package a qualified default resolved to, and its absence.
func TestDefaultPackage(t *testing.T) {
	t.Parallel()

	t.Run("a qualified default names its import", func(t *testing.T) {
		t.Parallel()
		bag := sdk.NewBag()
		stamp.MetaDefaultPkg.Set(bag, "example.com/seed", "test")

		testkit.Equal(t, stamp.DefaultPackage(bag), "example.com/seed",
			"a rendered file has to register the import, so the reference is carried")
	})

	t.Run("a literal names none", func(t *testing.T) {
		t.Parallel()
		bag := sdk.NewBag()
		stamp.MetaDefault.Set(bag, `"localhost"`, "test")

		testkit.Equal(t, stamp.DefaultPackage(bag), "",
			"a value written out needs nothing imported")
	})
}

// An unstamped owner reads as empty rather than panicking.
//
// A nil bag is what every declaration holds until something writes to
// it, so this is the common path and not an edge: a generator walking a
// tree asks about far more declarations than carry a directive.
func TestUnstampedReadsEmpty(t *testing.T) {
	t.Parallel()

	testkit.Equal(t, stamp.DefaultOf(nil), "", "an unstamped field declares no default")
	testkit.Equal(t, stamp.DefaultPackage(nil), "", "and names no package")
}

// stampCase is one written value and what a reader must give back.
type stampCase struct {
	name  string
	value string
}

func (c stampCase) Name() string { return c.name }
