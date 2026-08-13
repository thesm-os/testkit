// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law_test

import (
	"errors"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/engine/model/law"
)

// counted is a lifecycle whose census is its own: the subject the
// Outstanding arm exists for.
type counted struct {
	held           int
	stuck          bool
	readErr        error
	reads          int
	failAfterReads int
}

// TestLeakFreeOutstanding pins the subject-census arm: balanced cycles rest
// at zero, a subject that never releases grows by name, and a census that
// cannot be read is a refused precondition rather than a verdict.
func TestLeakFreeOutstanding(t *testing.T) {
	t.Parallel()

	mk := func(s *counted) law.LeakFree[*counted] {
		return law.LeakFree[*counted]{
			Open: func(_ *rapid.T, c *counted) error { c.held++; return nil },
			Close: func(_ *rapid.T, c *counted) error {
				if !c.stuck {
					c.held--
				}
				return nil
			},
			Outstanding: func(_ *rapid.T, c *counted) (int, error) {
				c.reads++
				if c.failAfterReads > 0 && c.reads > c.failAfterReads {
					return 0, errors.New("law_test: the census failed late")
				}
				return c.held, c.readErr
			},
			Cycles: 4,
		}
	}

	rapid.Check(t, func(rt *rapid.T) {
		balanced := &counted{}
		if err := mk(balanced).Check(rt, balanced, balanced); err != nil {
			rt.Fatalf("a balanced subject rests at its own zero: %v", err)
		}

		leaky := &counted{stuck: true}
		if err := mk(leaky).Check(rt, leaky, leaky); err == nil {
			rt.Fatal("a subject that never releases must grow by its own count")
		}

		unreadable := &counted{readErr: errors.New("unreadable")}
		if err := mk(unreadable).Check(rt, unreadable, unreadable); !law.Holds(err) {
			rt.Fatalf("an unreadable census is a refused precondition: %v", err)
		}

		lateFail := &counted{failAfterReads: 1}
		if err := mk(lateFail).Check(rt, lateFail, lateFail); !law.Holds(err) {
			rt.Fatalf("a census that fails after the cycles is a refused precondition: %v", err)
		}

		refusing := &counted{}
		l := mk(refusing)
		l.Open = func(*rapid.T, *counted) error { return errors.New("refused") }
		if err := l.Check(rt, refusing, refusing); !law.Holds(err) {
			rt.Fatalf("a refused open is a precondition, not a verdict: %v", err)
		}
		l = mk(refusing)
		l.Close = func(*rapid.T, *counted) error { return errors.New("refused") }
		if err := l.Check(rt, refusing, refusing); !law.Holds(err) {
			rt.Fatalf("a refused close is a precondition, not a verdict: %v", err)
		}
	})
}
