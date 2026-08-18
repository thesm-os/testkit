// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package projection_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/suite/projection"
)

func TestOptionNamePolicy(t *testing.T) {
	t.Parallel()

	testkit.Equal(t, projection.OptionName("Log", "Append"), projection.Option("WithLogAppend"),
		"the option policy is With<Iface><Method>")
}

// fixtureCase is one fixture-accessor spelling.
type fixtureCase struct {
	name  string
	token string
	field string
	want  projection.Expr
}

func (c fixtureCase) Name() string { return c.name }

func TestFixtureCallPolicy(t *testing.T) {
	t.Parallel()

	testkit.TableTest(t, []fixtureCase{
		{"token plus exported field plus parens", "log", "entry", "logEntry()"},
		{"initialisms case the platform's way", "kv", "id", "kvID()"},
		{"an empty field degrades to the bare token call", "kv", "", "kv()"},
	}, func(t *testing.T, tc fixtureCase) {
		testkit.Equal(t, projection.FixtureCall(tc.token, tc.field), tc.want,
			"the fixture accessor spelling has this one home")
	})
}
