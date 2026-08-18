// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"errors"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/suite"
)

// errSentinel is the error the induction and Must tests key on.
var errSentinel = errors.New("harness_test: sentinel")

func TestOneCtor(t *testing.T) {
	t.Parallel()

	plain := func() int { return 7 }
	start := func(testing.TB) int { return 9 }

	build, err := suite.OneCtor("h", plain, nil)
	if err != nil || build(t) != 7 {
		t.Errorf("the plain form must build: %v", err)
	}
	build, err = suite.OneCtor("h", nil, start)
	if err != nil || build(t) != 9 {
		t.Errorf("the lifecycle form must build: %v", err)
	}
	for name, tc := range map[string]struct {
		hName string
		p     func() int
		s     func(testing.TB) int
		want  string
	}{
		"neither": {"h", nil, nil, "neither New nor Start"},
		"both":    {"h", plain, start, "both New and Start"},
		"unnamed": {"", plain, nil, "no Name"},
	} {
		if _, err := suite.OneCtor(tc.hName, tc.p, tc.s); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: want error about %q, got %v", name, tc.want, err)
		}
	}
}

func TestDistinctPool(t *testing.T) {
	t.Parallel()

	got, err := suite.DistinctPool("KeyPool", nil, []string{"a", "b"})
	if err != nil || len(got) != 2 {
		t.Errorf("nil takes the derived default: %v %v", got, err)
	}
	if _, err := suite.DistinctPool("KeyPool", []string{"k", "k"}, nil); err == nil {
		t.Error("a pool repeating one value must be refused")
	}
	if _, err := suite.DistinctPool("KeyPool", []string{"k", "j", "k"}, nil); err != nil {
		t.Errorf("two distinct among repeats is enough: %v", err)
	}
}

func TestExclusivePair(t *testing.T) {
	t.Parallel()
	if err := suite.ExclusivePair("h", "OnClock", "StartOnClock", true, true); err == nil {
		t.Error("both members set must be refused")
	}
	if err := suite.ExclusivePair("h", "OnClock", "StartOnClock", true, false); err != nil {
		t.Errorf("one member set is the point: %v", err)
	}
}

func TestLowerRecover(t *testing.T) {
	t.Parallel()

	if got := suite.LowerRecover[any, int]("h", nil); got != nil {
		t.Error("a nil recover must lower to nil, so the caller assigns unconditionally")
	}

	lowered := suite.LowerRecover[any, int]("h", func(_ testing.TB, prior int) int {
		return prior + 1
	})
	if got := lowered(t, any(41)); got != any(42) {
		t.Errorf("the typed recover must run over the downcast prior, got %v", got)
	}
}

func TestLowerInductions(t *testing.T) {
	t.Parallel()

	if got := suite.LowerInductions[any, int]("h", nil); got != nil {
		t.Error("an empty induction map must lower to nil")
	}

	var got []any
	lowered := suite.LowerInductions[any, int]("h", suite.Inductions[int]{
		errSentinel: func(s int, sentinel error) { got = append(got, s, sentinel) },
	})
	f := testkit.NewFailableTB()
	lowered[errSentinel](f, any(7))
	if f.Failed() || len(got) != 2 || got[0] != 7 || !errors.Is(got[1].(error), errSentinel) {
		t.Errorf("the trigger must receive the downcast subject and its sentinel, got %v (failed=%v)", got, f.Failed())
	}

	wrong := testkit.NewFailableTB()
	lowered[errSentinel](wrong, any("not an int"))
	if !wrong.Failed() {
		t.Error("a subject of another type must fail loudly, not trigger")
	}
}

func TestExcuseSet(t *testing.T) {
	t.Parallel()

	if suite.ExcuseSet(nil) != nil {
		t.Error("no excuses must lower to nil, not an empty allocation")
	}
	set := suite.ExcuseSet([]suite.ID{"a", "b"})
	if len(set) != 2 || !set["a"] || !set["b"] {
		t.Errorf("every excused ID must be in the set: %v", set)
	}
}

func TestMust(t *testing.T) {
	t.Parallel()

	if got := suite.Must(42, nil); got != 42 {
		t.Errorf("a nil error must pass the value through, got %d", got)
	}
	defer func() {
		if recover() == nil {
			t.Error("an error must panic: Must is the invariant's last resort, not a reporting surface")
		}
	}()
	suite.Must(0, errSentinel)
}

// unwrapsTo is a decorator over the harness's concrete type: the shape
// tracing and metrics shims have, reached through Unwrap.
type unwrapsTo struct{ inner int }

func (u unwrapsTo) Unwrap() any { return any(u.inner) }

func TestLowerRecoverThroughAnUnwrapChain(t *testing.T) {
	t.Parallel()

	lowered := suite.LowerRecover[any, int]("h", func(_ testing.TB, prior int) int { return prior + 1 })
	if got := lowered(t, any(unwrapsTo{inner: 41})); got != any(42) {
		t.Errorf("a decorated prior must reach the typed recover through Unwrap, got %v", got)
	}

	f := testkit.NewFailableTB()
	lowered(f, any("no chain to an int"))
	if !f.Failed() {
		t.Error("a prior that unwraps to nothing usable must fail loudly")
	}
}
