// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package action_test

import (
	"context"
	"errors"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/model"
	"go.thesmos.sh/testkit/model/action"
)

// simpleStore is a test double with pluggable function fields.
type simpleStore struct {
	getF    func(string) (string, error)
	putF    func(string) error
	delF    func(string) error
	countF  func() (int, error)
	closeF  func() error
	descF   func() string
	emptyF  func() bool
	listF   func() ([]string, error)
	errF    func() error
	mutateF func(string)
}

var errBroken = errors.New("broken")

func TestReader(t *testing.T) {
	t.Parallel()

	t.Run("passes when SUT and ref agree", func(t *testing.T) {
		t.Parallel()
		a := action.Reader("Get", rapid.Just("k"),
			func(_ context.Context, s *simpleStore, _ string) (string, error) {
				return s.getF("k")
			},
		)
		sut := &simpleStore{getF: func(_ string) (string, error) { return "v", nil }}
		ref := &simpleStore{getF: func(_ string) (string, error) { return "v", nil }}
		rapid.Check(t, func(rt *rapid.T) {
			result := a.Run(rt, sut, ref)
			if result.Err != nil {
				rt.Fatalf("unexpected: %v", result.Err)
			}
		})
	})

	t.Run("catches value mismatch", func(t *testing.T) {
		t.Parallel()
		a := action.Reader("Get", rapid.Just("k"),
			func(_ context.Context, s *simpleStore, _ string) (string, error) {
				return s.getF("k")
			},
		)
		sut := &simpleStore{getF: func(_ string) (string, error) { return "wrong", nil }}
		ref := &simpleStore{getF: func(_ string) (string, error) { return "right", nil }}
		rapid.Check(t, func(rt *rapid.T) {
			result := a.Run(rt, sut, ref)
			if result.Err == nil {
				rt.Fatal("should have caught value mismatch")
			}
		})
	})

	t.Run("catches error mismatch", func(t *testing.T) {
		t.Parallel()
		a := action.Reader("Get", rapid.Just("k"),
			func(_ context.Context, s *simpleStore, _ string) (string, error) {
				return s.getF("k")
			},
		)
		sut := &simpleStore{getF: func(_ string) (string, error) { return "", errBroken }}
		ref := &simpleStore{getF: func(_ string) (string, error) { return "v", nil }}
		rapid.Check(t, func(rt *rapid.T) {
			result := a.Run(rt, sut, ref)
			if result.Err == nil {
				rt.Fatal("should have caught error mismatch")
			}
		})
	})
}

func TestWriter(t *testing.T) {
	t.Parallel()

	t.Run("passes when both succeed", func(t *testing.T) {
		t.Parallel()
		a := action.Writer("Put", rapid.Just("v"),
			func(_ context.Context, s *simpleStore, _ string) error { return s.putF("v") },
		)
		sut := &simpleStore{putF: func(_ string) error { return nil }}
		ref := &simpleStore{putF: func(_ string) error { return nil }}
		rapid.Check(t, func(rt *rapid.T) {
			result := a.Run(rt, sut, ref)
			if result.Err != nil {
				rt.Fatalf("unexpected: %v", result.Err)
			}
		})
	})

	t.Run("catches error mismatch", func(t *testing.T) {
		t.Parallel()
		a := action.Writer("Put", rapid.Just("v"),
			func(_ context.Context, s *simpleStore, _ string) error { return s.putF("v") },
		)
		sut := &simpleStore{putF: func(_ string) error { return errBroken }}
		ref := &simpleStore{putF: func(_ string) error { return nil }}
		rapid.Check(t, func(rt *rapid.T) {
			result := a.Run(rt, sut, ref)
			if result.Err == nil {
				rt.Fatal("should have caught error mismatch")
			}
		})
	})
}

func TestDeleter(t *testing.T) {
	t.Parallel()

	t.Run("passes when both succeed", func(t *testing.T) {
		t.Parallel()
		a := action.Deleter("Delete", rapid.Just("k"),
			func(_ context.Context, s *simpleStore, _ string) error { return s.delF("k") },
		)
		sut := &simpleStore{delF: func(_ string) error { return nil }}
		ref := &simpleStore{delF: func(_ string) error { return nil }}
		rapid.Check(t, func(rt *rapid.T) {
			result := a.Run(rt, sut, ref)
			if result.Err != nil {
				rt.Fatalf("unexpected: %v", result.Err)
			}
		})
	})

	t.Run("catches error mismatch", func(t *testing.T) {
		t.Parallel()
		a := action.Deleter("Delete", rapid.Just("k"),
			func(_ context.Context, s *simpleStore, _ string) error { return s.delF("k") },
		)
		sut := &simpleStore{delF: func(_ string) error { return errBroken }}
		ref := &simpleStore{delF: func(_ string) error { return nil }}
		rapid.Check(t, func(rt *rapid.T) {
			result := a.Run(rt, sut, ref)
			if result.Err == nil {
				rt.Fatal("should have caught error mismatch")
			}
		})
	})
}

func TestAggregator(t *testing.T) {
	t.Parallel()

	t.Run("passes when values match", func(t *testing.T) {
		t.Parallel()
		a := action.Aggregator("Count",
			func(_ context.Context, s *simpleStore) (int, error) { return s.countF() },
		)
		sut := &simpleStore{countF: func() (int, error) { return 5, nil }}
		ref := &simpleStore{countF: func() (int, error) { return 5, nil }}
		rapid.Check(t, func(rt *rapid.T) {
			result := a.Run(rt, sut, ref)
			if result.Err != nil {
				rt.Fatalf("unexpected: %v", result.Err)
			}
		})
	})

	t.Run("catches value mismatch", func(t *testing.T) {
		t.Parallel()
		a := action.Aggregator("Count",
			func(_ context.Context, s *simpleStore) (int, error) { return s.countF() },
		)
		sut := &simpleStore{countF: func() (int, error) { return 3, nil }}
		ref := &simpleStore{countF: func() (int, error) { return 5, nil }}
		rapid.Check(t, func(rt *rapid.T) {
			result := a.Run(rt, sut, ref)
			if result.Err == nil {
				rt.Fatal("should have caught value mismatch")
			}
		})
	})

	t.Run("catches error mismatch", func(t *testing.T) {
		t.Parallel()
		a := action.Aggregator("Count",
			func(_ context.Context, s *simpleStore) (int, error) { return s.countF() },
		)
		sut := &simpleStore{countF: func() (int, error) { return 0, errBroken }}
		ref := &simpleStore{countF: func() (int, error) { return 0, nil }}
		rapid.Check(t, func(rt *rapid.T) {
			result := a.Run(rt, sut, ref)
			if result.Err == nil {
				rt.Fatal("should have caught error mismatch")
			}
		})
	})
}

func TestLifecycle(t *testing.T) {
	t.Parallel()

	t.Run("passes when both succeed", func(t *testing.T) {
		t.Parallel()
		a := action.Lifecycle("Close",
			func(_ context.Context, s *simpleStore) error { return s.closeF() },
		)
		sut := &simpleStore{closeF: func() error { return nil }}
		ref := &simpleStore{closeF: func() error { return nil }}
		rapid.Check(t, func(rt *rapid.T) {
			result := a.Run(rt, sut, ref)
			if result.Err != nil {
				rt.Fatalf("unexpected: %v", result.Err)
			}
		})
	})

	t.Run("catches error mismatch", func(t *testing.T) {
		t.Parallel()
		a := action.Lifecycle("Close",
			func(_ context.Context, s *simpleStore) error { return s.closeF() },
		)
		sut := &simpleStore{closeF: func() error { return errBroken }}
		ref := &simpleStore{closeF: func() error { return nil }}
		rapid.Check(t, func(rt *rapid.T) {
			result := a.Run(rt, sut, ref)
			if result.Err == nil {
				rt.Fatal("should have caught error mismatch")
			}
		})
	})
}

func TestPure(t *testing.T) {
	t.Parallel()

	t.Run("passes when results match", func(t *testing.T) {
		t.Parallel()
		a := action.Pure("Describe", func(s *simpleStore) string { return s.descF() })
		sut := &simpleStore{descF: func() string { return "hello" }}
		ref := &simpleStore{descF: func() string { return "hello" }}
		rapid.Check(t, func(rt *rapid.T) {
			result := a.Run(rt, sut, ref)
			if result.Err != nil {
				rt.Fatalf("unexpected: %v", result.Err)
			}
		})
	})

	t.Run("catches result mismatch", func(t *testing.T) {
		t.Parallel()
		a := action.Pure("Describe", func(s *simpleStore) string { return s.descF() })
		sut := &simpleStore{descF: func() string { return "wrong" }}
		ref := &simpleStore{descF: func() string { return "right" }}
		rapid.Check(t, func(rt *rapid.T) {
			result := a.Run(rt, sut, ref)
			if result.Err == nil {
				rt.Fatal("should have caught result mismatch")
			}
		})
	})
}

func TestPredicate(t *testing.T) {
	t.Parallel()

	t.Run("passes when bools match", func(t *testing.T) {
		t.Parallel()
		a := action.Predicate("IsEmpty", func(s *simpleStore) bool { return s.emptyF() })
		sut := &simpleStore{emptyF: func() bool { return true }}
		ref := &simpleStore{emptyF: func() bool { return true }}
		rapid.Check(t, func(rt *rapid.T) {
			result := a.Run(rt, sut, ref)
			if result.Err != nil {
				rt.Fatalf("unexpected: %v", result.Err)
			}
		})
	})

	t.Run("catches bool mismatch", func(t *testing.T) {
		t.Parallel()
		a := action.Predicate("IsEmpty", func(s *simpleStore) bool { return s.emptyF() })
		sut := &simpleStore{emptyF: func() bool { return false }}
		ref := &simpleStore{emptyF: func() bool { return true }}
		rapid.Check(t, func(rt *rapid.T) {
			result := a.Run(rt, sut, ref)
			if result.Err == nil {
				rt.Fatal("should have caught bool mismatch")
			}
		})
	})
}

func TestStream(t *testing.T) {
	t.Parallel()

	t.Run("passes when items match", func(t *testing.T) {
		t.Parallel()
		a := action.Stream("List",
			func(_ context.Context, s *simpleStore) ([]string, error) { return s.listF() },
		)
		sut := &simpleStore{listF: func() ([]string, error) { return []string{"a", "b"}, nil }}
		ref := &simpleStore{listF: func() ([]string, error) { return []string{"a", "b"}, nil }}
		rapid.Check(t, func(rt *rapid.T) {
			result := a.Run(rt, sut, ref)
			if result.Err != nil {
				rt.Fatalf("unexpected: %v", result.Err)
			}
		})
	})

	t.Run("order insensitive", func(t *testing.T) {
		t.Parallel()
		a := action.Stream("List",
			func(_ context.Context, s *simpleStore) ([]string, error) { return s.listF() },
		)
		sut := &simpleStore{listF: func() ([]string, error) { return []string{"b", "a"}, nil }}
		ref := &simpleStore{listF: func() ([]string, error) { return []string{"a", "b"}, nil }}
		rapid.Check(t, func(rt *rapid.T) {
			result := a.Run(rt, sut, ref)
			if result.Err != nil {
				rt.Fatalf("unexpected: %v", result.Err)
			}
		})
	})

	t.Run("catches item mismatch", func(t *testing.T) {
		t.Parallel()
		a := action.Stream("List",
			func(_ context.Context, s *simpleStore) ([]string, error) { return s.listF() },
		)
		sut := &simpleStore{listF: func() ([]string, error) { return []string{"x"}, nil }}
		ref := &simpleStore{listF: func() ([]string, error) { return []string{"y"}, nil }}
		rapid.Check(t, func(rt *rapid.T) {
			result := a.Run(rt, sut, ref)
			if result.Err == nil {
				rt.Fatal("should have caught item mismatch")
			}
		})
	})

	t.Run("catches error mismatch", func(t *testing.T) {
		t.Parallel()
		a := action.Stream("List",
			func(_ context.Context, s *simpleStore) ([]string, error) { return s.listF() },
		)
		sut := &simpleStore{listF: func() ([]string, error) { return nil, errBroken }}
		ref := &simpleStore{listF: func() ([]string, error) { return []string{"a"}, nil }}
		rapid.Check(t, func(rt *rapid.T) {
			result := a.Run(rt, sut, ref)
			if result.Err == nil {
				rt.Fatal("should have caught error mismatch")
			}
		})
	})
}

func TestPoisonCheck(t *testing.T) {
	t.Parallel()

	t.Run("passes when both nil", func(t *testing.T) {
		t.Parallel()
		a := action.PoisonCheck("Err", func(s *simpleStore) error { return s.errF() })
		sut := &simpleStore{errF: func() error { return nil }}
		ref := &simpleStore{errF: func() error { return nil }}
		rapid.Check(t, func(rt *rapid.T) {
			result := a.Run(rt, sut, ref)
			if result.Err != nil {
				rt.Fatalf("unexpected: %v", result.Err)
			}
		})
	})

	t.Run("catches error mismatch", func(t *testing.T) {
		t.Parallel()
		a := action.PoisonCheck("Err", func(s *simpleStore) error { return s.errF() })
		sut := &simpleStore{errF: func() error { return errBroken }}
		ref := &simpleStore{errF: func() error { return nil }}
		rapid.Check(t, func(rt *rapid.T) {
			result := a.Run(rt, sut, ref)
			if result.Err == nil {
				rt.Fatal("should have caught error mismatch")
			}
		})
	})
}

func TestMutator(t *testing.T) {
	t.Parallel()

	t.Run("calls both SUT and ref", func(t *testing.T) {
		t.Parallel()
		var sutCalled, refCalled bool
		a := action.Mutator("Mutate", rapid.Just("v"),
			func(_ context.Context, s *simpleStore, _ string) { s.mutateF("v") },
		)
		sut := &simpleStore{mutateF: func(_ string) { sutCalled = true }}
		ref := &simpleStore{mutateF: func(_ string) { refCalled = true }}
		rapid.Check(t, func(rt *rapid.T) { a.Run(rt, sut, ref) })
		if !sutCalled || !refCalled {
			t.Fatalf("both SUT and ref must be called: sut=%v ref=%v", sutCalled, refCalled)
		}
	})
}

func TestStress(t *testing.T) {
	t.Parallel()

	t.Run("calls SUT without comparison", func(t *testing.T) {
		t.Parallel()
		var called bool
		a := action.Stress("Count", func(s *simpleStore) {
			called = true
		})
		sut := &simpleStore{}
		rapid.Check(t, func(rt *rapid.T) { a.Run(rt, sut, nil) })
		if !called {
			t.Fatal("stress action must be called")
		}
	})
}

func TestReaderWithBool(t *testing.T) {
	t.Parallel()

	t.Run("passes when ok and values match", func(t *testing.T) {
		t.Parallel()
		a := action.ReaderWithBool("Get", rapid.Just("k"),
			func(_ context.Context, _ *simpleStore, _ string) (string, bool) { return "v", true },
		)
		rapid.Check(t, func(rt *rapid.T) {
			result := a.Run(rt, &simpleStore{}, &simpleStore{})
			if result.Err != nil {
				rt.Fatalf("unexpected: %v", result.Err)
			}
		})
	})

	t.Run("catches ok flag mismatch", func(t *testing.T) {
		t.Parallel()
		a := action.ReaderWithBool("Get", rapid.Just("k"),
			func(_ context.Context, s *simpleStore, _ string) (string, bool) {
				if s.emptyF != nil {
					return "", false // SUT returns not-found
				}
				return "v", true // ref returns found
			},
		)
		sut := &simpleStore{emptyF: func() bool { return true }} // marker
		ref := &simpleStore{}
		rapid.Check(t, func(rt *rapid.T) {
			result := a.Run(rt, sut, ref)
			if result.Err == nil {
				rt.Fatal("should have caught ok flag mismatch")
			}
		})
	})
}

func TestLookup(t *testing.T) {
	t.Parallel()

	t.Run("passes when ok and values match", func(t *testing.T) {
		t.Parallel()
		a := action.Lookup("Inspect", rapid.Just("k"),
			func(_ *simpleStore, _ string) (string, int, bool) {
				return "v", 42, true
			},
		)
		rapid.Check(t, func(rt *rapid.T) {
			result := a.Run(rt, &simpleStore{}, &simpleStore{})
			if result.Err != nil {
				rt.Fatalf("unexpected: %v", result.Err)
			}
		})
	})

	t.Run("catches ok flag mismatch", func(t *testing.T) {
		t.Parallel()
		sut := &simpleStore{emptyF: func() bool { return true }}
		ref := &simpleStore{}
		a := action.Lookup("Inspect", rapid.Just("k"),
			func(s *simpleStore, _ string) (string, int, bool) {
				if s.emptyF != nil {
					return "", 0, false
				}
				return "v", 42, true
			},
		)
		rapid.Check(t, func(rt *rapid.T) {
			result := a.Run(rt, sut, ref)
			if result.Err == nil {
				rt.Fatal("should catch ok mismatch")
			}
		})
	})

	t.Run("catches R1 value mismatch", func(t *testing.T) {
		t.Parallel()
		sut := &simpleStore{emptyF: func() bool { return true }}
		ref := &simpleStore{}
		a := action.Lookup("Inspect", rapid.Just("k"),
			func(s *simpleStore, _ string) (string, int, bool) {
				if s.emptyF != nil {
					return "WRONG", 42, true
				}
				return "correct", 42, true
			},
		)
		rapid.Check(t, func(rt *rapid.T) {
			result := a.Run(rt, sut, ref)
			if result.Err == nil {
				rt.Fatal("should catch R1 mismatch")
			}
		})
	})
}

func TestUnknown(t *testing.T) {
	t.Parallel()

	t.Run("delegates to consumer run function", func(t *testing.T) {
		t.Parallel()
		called := false
		a := action.Unknown[*simpleStore]("Custom", func(_ *rapid.T, sut, ref *simpleStore) model.ActionResult {
			called = true
			return model.ActionResult{}
		})
		if a.Name != "Custom" {
			t.Fatalf("expected name Custom, got %s", a.Name)
		}
		rapid.Check(t, func(rt *rapid.T) {
			result := a.Run(rt, &simpleStore{}, &simpleStore{})
			if result.Err != nil {
				rt.Fatalf("unexpected: %v", result.Err)
			}
		})
		if !called {
			t.Fatal("consumer run function must be called")
		}
	})
}

func TestActionKind(t *testing.T) {
	t.Parallel()

	t.Run("Reader sets FailureSemantic", func(t *testing.T) {
		t.Parallel()
		a := action.Reader("Get", rapid.Just("k"),
			func(_ context.Context, _ *simpleStore, _ string) (string, error) { return "", nil },
		)
		if a.Kind != model.FailureSemantic {
			t.Fatalf("expected FailureSemantic, got %v", a.Kind)
		}
	})

	t.Run("ChainAppend sets FailureStructural", func(t *testing.T) {
		t.Parallel()
		a := action.ChainAppend("Append", rapid.Just("entry"),
			func(_ context.Context, _ *simpleStore, _ string) error { return nil },
		)
		if a.Kind != model.FailureStructural {
			t.Fatalf("expected FailureStructural, got %v", a.Kind)
		}
	})
}
