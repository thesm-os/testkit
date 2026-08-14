// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"slices"
	"testing"

	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/ifabsent"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/ifmatch"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/outbox"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/suite"
)

// The roles and keys this generator selects on are spelled as constants, and a
// spelling is only as good as what checks it.
//
// eidos publishes each contract's vocabulary as a slice rather than as named
// constants, so nothing but this holds the two together. Without it a role
// renamed upstream selects nothing, every contract check disappears, and the
// corpus still compiles — which is the failure mode the whole tier exists to
// prevent.
func TestContractVocabularyIsUpstream(t *testing.T) {
	t.Parallel()

	t.Run("names roles the contracts declare", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, slices.Contains(ifabsent.Roles, suite.ContractIfAbsentRole),
			"if-absent declares the role the check selects on")
		testkit.True(t, slices.Contains(ifmatch.Roles, suite.ContractIfMatchRole),
			"if-match declares the role the check selects on")
		testkit.True(t, slices.Contains(outbox.Roles, suite.ContractOutboxRole),
			"outbox declares the role the check selects on")
		testkit.True(t, slices.Contains(outbox.Roles, suite.ContractOutboxPartner),
			"and the partner role the check reaches through")
	})

	t.Run("names the predicate role the contract declares", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, slices.Contains(ifmatch.Roles, suite.ContractIfMatchMatch),
			"if-match declares the role naming its predicate")
	})
}

// A write that only lands when the key is absent, which is the difference
// between "create" and "save".
func TestIfAbsentCheck(t *testing.T) {
	t.Parallel()

	t.Run("emits where the method fills the writer role", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, hasCheckIn(t, ifAbsentFixture(t, suite.ContractIfAbsentRole), "Put", "if-absent"),
			"the role is the whole gate; the claim needs nothing else")
	})

	t.Run("emits nothing for another role of the same contract", func(t *testing.T) {
		t.Parallel()
		// A contract is a protocol rather than a property. Written against
		// whatever member the directive happened to sit on, the check would call
		// the wrong method — and the role is the only thing distinguishing them.
		testkit.False(t, hasCheckIn(t, ifAbsentFixture(t, "reader"), "Put", "if-absent"),
			"a member filling some other role owes no writer's check")
	})

	t.Run("writes the alternate value rather than the seeded one", func(t *testing.T) {
		t.Parallel()
		// The harness seeds each fresh subject through this very method, so
		// handed the seeded value the first write here would already be the
		// second — and a correct implementation would fail on the line that
		// expects it to succeed.
		ck := checkNamed(t, contractIn(t, ifAbsentFixture(t, suite.ContractIfAbsentRole)), "Put", "if-absent")
		testkit.Equal(t, ck.Args, []string{"VOther"},
			"the check supplies a value the seed did not already write")
	})
}

// The predicate that gates the write, and the agreement between the two.
func TestIfMatchCheck(t *testing.T) {
	t.Parallel()

	t.Run("emits where the named predicate answers about the same value", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, hasCheckIn(t, ifMatchFixture(t, "Match", nil), "Put", "if-match"),
			"one bool over the writer's own parameter list is a verdict the check can use")
	})

	t.Run("emits nothing where the contract names no predicate", func(t *testing.T) {
		t.Parallel()
		// Without one there is nothing for the write to agree with, and
		// "refuses something" is not a claim until something is named.
		testkit.False(t, hasCheckIn(t, ifMatchFixture(t, "", nil), "Put", "if-match"),
			"an unnamed predicate is nothing to compare against")
	})

	t.Run("emits nothing where the name matches no method", func(t *testing.T) {
		t.Parallel()
		// The resolver reports a partner naming nothing in scope, so a real run
		// never reaches here — but a check composing a call to a method the
		// subject does not declare would not compile, and a render error is a
		// file that came out short.
		testkit.False(t, hasCheckIn(t, ifMatchFixture(t, "Absent", nil), "Put", "if-match"),
			"a predicate outside the interface is one the subject cannot be asked for")
	})

	t.Run("emits nothing where the predicate answers about something else", func(t *testing.T) {
		t.Parallel()
		// A check receives the writer's arguments and nothing else, so a
		// predicate over another type is one it cannot call.
		testkit.False(t, hasCheckIn(t, ifMatchFixture(t, "Match", storefixture.Named("string")), "Put", "if-match"),
			"a predicate over a different parameter list is a method the directive was pointed at")
	})

	t.Run("emits nothing where the predicate returns no verdict", func(t *testing.T) {
		t.Parallel()
		testkit.False(t, hasCheckIn(t, ifMatchNonBoolFixture(t), "Put", "if-match"),
			"a predicate returning something other than a bool states no verdict")
	})

	t.Run("emits nothing where the predicate takes a different arity", func(t *testing.T) {
		t.Parallel()
		// One value in, one verdict out. A predicate wanting more has
		// parameters the check cannot supply, since it receives the writer's
		// and nothing else.
		testkit.False(t, hasCheckIn(t, ifMatchWiderPredicateFixture(t), "Put", "if-match"),
			"a predicate taking more than the writer does is one the check cannot call")
	})

	t.Run("emits the check where the predicate merely names its parameter differently", func(t *testing.T) {
		t.Parallel()
		// This used to emit nothing, on the grounds that the fixture keys a
		// field on the parameter's name as well as its type — so `Match(ctx,
		// candidate Value)` beside `Put(ctx, v Value)` resolved to a field the
		// check was not handed.
		//
		// The premise was wrong. The check is handed `v`, and `v` is the only
		// parameter of that type, so `Match(ctx, v)` names a value in scope and
		// says exactly what the contract does. What the old rule actually
		// required was identical spelling, which is a stronger condition than
		// the one that makes the call writable.
		testkit.True(t, hasCheckIn(t, ifMatchRenamedParamFixture(t), "Put", "if-match"),
			"one parameter of the type is one candidate, and no guess is involved")
	})

	t.Run("emits nothing where two parameters could be the predicate's", func(t *testing.T) {
		t.Parallel()
		// The condition that replaced identical spelling. `Put(ctx, from, to
		// Value)` beside `Match(ctx, a, b Value)` has two candidates per slot and
		// nothing choosing between them, so a generated call would be a check
		// about whichever end the derivation happened to visit first.
		//
		// This is the ambiguity `partition` settles with `axis=`, arriving on a
		// contract that has no such key — so the answer is to decline and say
		// why, which [TestPartnerArgs] pins the wording of.
		testkit.False(t, hasCheckIn(t, ifMatchAmbiguousParamFixture(t), "Put", "if-match"),
			"two candidates of one type is a correspondence the source has not stated")
	})
}

// A record appended before anyone subscribed, which is what an outbox holds and
// a publisher may drop.
func TestOutboxCheck(t *testing.T) {
	t.Parallel()

	t.Run("emits where the subscriber streams what the appender takes", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, hasCheckIn(t, outboxFixture(t, "Subscribe", nil), "Append", "outbox"),
			"a stream of the appended type is one the check can hold a record up to")
	})

	t.Run("measures on the run's clock", func(t *testing.T) {
		t.Parallel()
		// The wait is a liveness guard, and reading the wall clock would put
		// the machine the run is on into the subject.
		ck := checkNamed(t, contractIn(t, outboxFixture(t, "Subscribe", nil)), "Append", "outbox")
		testkit.True(t, ck.NeedsClock, "the check is handed the clock the run holds")
	})

	t.Run("emits nothing where the contract names no subscriber", func(t *testing.T) {
		t.Parallel()
		testkit.False(t, hasCheckIn(t, outboxFixture(t, "", nil), "Append", "outbox"),
			"an unnamed subscriber is nothing to receive through")
	})

	t.Run("emits nothing where the stream carries something else", func(t *testing.T) {
		t.Parallel()
		// Holding a received string up to an appended struct would not
		// compile, and a render error is a file that came out short.
		testkit.False(t, hasCheckIn(t, outboxFixture(t, "Subscribe", storefixture.Named("string")), "Append", "outbox"),
			"a stream of another type is one the check cannot compare through")
	})

	t.Run("emits nothing where the subscriber returns no stream", func(t *testing.T) {
		t.Parallel()
		testkit.False(t, hasCheckIn(t, outboxNoStreamFixture(t), "Append", "outbox"),
			"a partner handing back a value rather than a channel has nothing to receive from")
	})

	t.Run("emits nothing where the stream is send-only", func(t *testing.T) {
		t.Parallel()
		// The check receives from it, and Go refuses a receive on a send-only
		// channel — so this is a generated call the toolchain would reject.
		testkit.False(t, hasCheckIn(t, outboxSendOnlyFixture(t), "Append", "outbox"),
			"a send-only stream is one the check cannot read")
	})

	t.Run("emits nothing where the subscriber hands back several values", func(t *testing.T) {
		t.Parallel()
		// Which one carries the records is a guess, and the contract names the
		// partner without saying — the ambiguity partition needed an axis for.
		testkit.False(t, hasCheckIn(t, outboxTwoValueFixture(t), "Append", "outbox"),
			"two produced values and one stream to read is a pairing nothing states")
	})

	t.Run("emits nothing where the appender cannot report a refusal", func(t *testing.T) {
		t.Parallel()
		// The claim is that a record the subject *accepted* is delivered, so an
		// append with nowhere to say it refused leaves the wait unable to tell
		// "dropped" from "never taken".
		testkit.False(t, hasCheckIn(t, outboxSilentAppenderFixture(t), "Append", "outbox"),
			"an appender returning nothing states no acceptance to hold it to")
	})

	t.Run("emits nothing where the subscriber needs what the appender does not take", func(t *testing.T) {
		t.Parallel()
		// A check receives the appender's parameters and nothing else, so a
		// subscriber wanting more cannot be called from inside it.
		testkit.False(t, hasCheckIn(t, outboxWiderSubscriberFixture(t), "Append", "outbox"),
			"a subscriber taking a parameter the appender does not is one the check cannot call")
	})
}

// ifAbsentFixture is a single-writer interface filling the given role of the
// if-absent contract.
func ifAbsentFixture(t *testing.T, role string) *sdk.Store {
	t.Helper()
	s := valueStore(t, func(i *storefixture.InterfaceBuilder) {
		i.Method("Put", func(m *storefixture.MethodBuilder) {
			m.Param("ctx", storefixture.PkgNamed("context", "Context"))
			m.Param("v", storefixture.PkgNamed("example.com/box", "Value"))
			m.Return(storefixture.Named("error"))
		})
	})
	stampContract(s, "Put", suite.ContractIfAbsent, role)
	return s
}

// ifMatchFixture pairs a writer with a predicate over the given type, or over
// the writer's own parameter when nil.
func ifMatchFixture(t *testing.T, pred string, over *sdk.TypeRef) *sdk.Store {
	t.Helper()
	if over == nil {
		over = storefixture.PkgNamed("example.com/box", "Value")
	}
	s := valueStore(t, func(i *storefixture.InterfaceBuilder) {
		i.Method("Put", func(m *storefixture.MethodBuilder) {
			m.Param("ctx", storefixture.PkgNamed("context", "Context"))
			m.Param("v", storefixture.PkgNamed("example.com/box", "Value"))
			m.Return(storefixture.Named("error"))
		})
		i.Method("Match", func(m *storefixture.MethodBuilder) {
			m.Param("ctx", storefixture.PkgNamed("context", "Context"))
			m.Param("v", over)
			m.Return(storefixture.Named("bool"))
			m.Return(storefixture.Named("error"))
		})
	})
	stampContract(s, "Put", suite.ContractIfMatch, suite.ContractIfMatchRole)
	if pred != "" {
		stampContractPartner(s, "Put", suite.ContractIfMatch, suite.ContractIfMatchMatch, pred)
	}
	return s
}

// ifMatchNonBoolFixture names a predicate whose verdict slot is not a bool.
func ifMatchNonBoolFixture(t *testing.T) *sdk.Store {
	t.Helper()
	s := valueStore(t, func(i *storefixture.InterfaceBuilder) {
		i.Method("Put", func(m *storefixture.MethodBuilder) {
			m.Param("ctx", storefixture.PkgNamed("context", "Context"))
			m.Param("v", storefixture.PkgNamed("example.com/box", "Value"))
			m.Return(storefixture.Named("error"))
		})
		i.Method("Match", func(m *storefixture.MethodBuilder) {
			m.Param("ctx", storefixture.PkgNamed("context", "Context"))
			m.Param("v", storefixture.PkgNamed("example.com/box", "Value"))
			m.Return(storefixture.Named("int"))
			m.Return(storefixture.Named("error"))
		})
	})
	stampContract(s, "Put", suite.ContractIfMatch, suite.ContractIfMatchRole)
	stampContractPartner(s, "Put", suite.ContractIfMatch, suite.ContractIfMatchMatch, "Match")
	return s
}

// outboxFixture pairs an appender with a subscriber streaming the given
// element type, or the appended type when nil.
func outboxFixture(t *testing.T, partner string, elem *sdk.TypeRef) *sdk.Store {
	t.Helper()
	if elem == nil {
		elem = storefixture.PkgNamed("example.com/box", "Value")
	}
	s := valueStore(t, func(i *storefixture.InterfaceBuilder) {
		i.Method("Append", func(m *storefixture.MethodBuilder) {
			m.Param("ctx", storefixture.PkgNamed("context", "Context"))
			m.Param("v", storefixture.PkgNamed("example.com/box", "Value"))
			m.Return(storefixture.Named("error"))
		})
		i.Method("Subscribe", func(m *storefixture.MethodBuilder) {
			m.Param("ctx", storefixture.PkgNamed("context", "Context"))
			m.Return(storefixture.RecvChan(elem))
			m.Return(storefixture.Named("error"))
		})
	})
	stampContract(s, "Append", suite.ContractOutbox, suite.ContractOutboxRole)
	if partner != "" {
		stampContractPartner(s, "Append", suite.ContractOutbox, suite.ContractOutboxPartner, partner)
	}
	return s
}

// outboxNoStreamFixture names a subscriber handing back a value rather than a
// channel.
func outboxNoStreamFixture(t *testing.T) *sdk.Store {
	t.Helper()
	s := valueStore(t, func(i *storefixture.InterfaceBuilder) {
		i.Method("Append", func(m *storefixture.MethodBuilder) {
			m.Param("ctx", storefixture.PkgNamed("context", "Context"))
			m.Param("v", storefixture.PkgNamed("example.com/box", "Value"))
			m.Return(storefixture.Named("error"))
		})
		i.Method("Subscribe", func(m *storefixture.MethodBuilder) {
			m.Param("ctx", storefixture.PkgNamed("context", "Context"))
			m.Return(storefixture.PkgNamed("example.com/box", "Value"))
			m.Return(storefixture.Named("error"))
		})
	})
	stampContract(s, "Append", suite.ContractOutbox, suite.ContractOutboxRole)
	stampContractPartner(s, "Append", suite.ContractOutbox, suite.ContractOutboxPartner, "Subscribe")
	return s
}

// valueStore declares one struct and one opted-in interface over it, so each
// fixture states only the methods its case is about.
func valueStore(t *testing.T, methods func(*storefixture.InterfaceBuilder)) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("box", "example.com/box").
		Struct("Value", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At("box/iface.go", 1, 1))
			b.Field("Key", storefixture.Named("string"), nil)
			b.Field("Body", storefixture.Named("string"), nil)
		}).
		Interface("Contract", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("box/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			methods(i)
		}).
		Build()
}

// stampContract records a method's membership and role, which is what the
// annotator writes for a `//testkit:contract` line.
func stampContract(s *sdk.Store, method, contract, role string) {
	forMethod(s, method, func(bag *sdk.Bag) {
		shape.MetaContracts.Set(bag, []string{contract}, "test")
		shape.ContractRoleKey(contract).Set(bag, role, "test")
	})
}

// stampContractPartner records the callable filling a partner role, in the
// qualified form the refinement resolver rewrites it to.
func stampContractPartner(s *sdk.Store, method, contract, role, partner string) {
	forMethod(s, method, func(bag *sdk.Bag) {
		shape.ContractPartnerKey(contract, role).
			Set(bag, "example.com/box.Contract."+partner, "test")
	})
}

// forMethod applies fn to the meta bag of every method of that name.
func forMethod(s *sdk.Store, method string, fn func(*sdk.Bag)) {
	for _, iface := range s.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			if m.Name == method {
				fn(m.EnsureMeta())
			}
		}
	}
}

// ifMatchWiderPredicateFixture names a predicate taking more than the writer.
func ifMatchWiderPredicateFixture(t *testing.T) *sdk.Store {
	t.Helper()
	s := valueStore(t, func(i *storefixture.InterfaceBuilder) {
		i.Method("Put", func(m *storefixture.MethodBuilder) {
			m.Param("ctx", storefixture.PkgNamed("context", "Context"))
			m.Param("v", storefixture.PkgNamed("example.com/box", "Value"))
			m.Return(storefixture.Named("error"))
		})
		i.Method("Match", func(m *storefixture.MethodBuilder) {
			m.Param("ctx", storefixture.PkgNamed("context", "Context"))
			m.Param("v", storefixture.PkgNamed("example.com/box", "Value"))
			m.Param("scope", storefixture.Named("string"))
			m.Return(storefixture.Named("bool"))
		})
	})
	stampContract(s, "Put", suite.ContractIfMatch, suite.ContractIfMatchRole)
	stampContractPartner(s, "Put", suite.ContractIfMatch, suite.ContractIfMatchMatch, "Match")
	return s
}

// ifMatchRenamedParamFixture names a predicate over the same type under another
// identifier, which the fixture files as a field of its own.
func ifMatchRenamedParamFixture(t *testing.T) *sdk.Store {
	t.Helper()
	s := valueStore(t, func(i *storefixture.InterfaceBuilder) {
		i.Method("Put", func(m *storefixture.MethodBuilder) {
			m.Param("ctx", storefixture.PkgNamed("context", "Context"))
			m.Param("v", storefixture.PkgNamed("example.com/box", "Value"))
			m.Return(storefixture.Named("error"))
		})
		i.Method("Match", func(m *storefixture.MethodBuilder) {
			m.Param("ctx", storefixture.PkgNamed("context", "Context"))
			m.Param("candidate", storefixture.PkgNamed("example.com/box", "Value"))
			m.Return(storefixture.Named("bool"))
		})
	})
	stampContract(s, "Put", suite.ContractIfMatch, suite.ContractIfMatchRole)
	stampContractPartner(s, "Put", suite.ContractIfMatch, suite.ContractIfMatchMatch, "Match")
	return s
}

// ifMatchAmbiguousParamFixture gives the writer two parameters of the
// predicate's type, so nothing in the source says which one it is about.
func ifMatchAmbiguousParamFixture(t *testing.T) *sdk.Store {
	t.Helper()
	s := valueStore(t, func(i *storefixture.InterfaceBuilder) {
		i.Method("Put", func(m *storefixture.MethodBuilder) {
			m.Param("ctx", storefixture.PkgNamed("context", "Context"))
			m.Param("from", storefixture.PkgNamed("example.com/box", "Value"))
			m.Param("to", storefixture.PkgNamed("example.com/box", "Value"))
			m.Return(storefixture.Named("error"))
		})
		// Same arity and the same types in the same order, so the predicate
		// passes every structural test and only the correspondence is open.
		i.Method("Match", func(m *storefixture.MethodBuilder) {
			m.Param("ctx", storefixture.PkgNamed("context", "Context"))
			m.Param("a", storefixture.PkgNamed("example.com/box", "Value"))
			m.Param("b", storefixture.PkgNamed("example.com/box", "Value"))
			m.Return(storefixture.Named("bool"))
		})
	})
	stampContract(s, "Put", suite.ContractIfMatch, suite.ContractIfMatchRole)
	stampContractPartner(s, "Put", suite.ContractIfMatch, suite.ContractIfMatchMatch, "Match")
	return s
}

// outboxSendOnlyFixture hands back a channel the check cannot receive from.
func outboxSendOnlyFixture(t *testing.T) *sdk.Store {
	t.Helper()
	return outboxWith(t, func(m *storefixture.MethodBuilder) {
		m.Param("ctx", storefixture.PkgNamed("context", "Context"))
		m.Return(storefixture.SendChan(storefixture.PkgNamed("example.com/box", "Value")))
		m.Return(storefixture.Named("error"))
	})
}

// outboxTwoValueFixture hands back two values, so which carries the records is
// a guess.
func outboxTwoValueFixture(t *testing.T) *sdk.Store {
	t.Helper()
	return outboxWith(t, func(m *storefixture.MethodBuilder) {
		m.Param("ctx", storefixture.PkgNamed("context", "Context"))
		m.Return(storefixture.RecvChan(storefixture.PkgNamed("example.com/box", "Value")))
		m.Return(storefixture.Named("int"))
		m.Return(storefixture.Named("error"))
	})
}

// outboxWiderSubscriberFixture takes a parameter the appender does not.
func outboxWiderSubscriberFixture(t *testing.T) *sdk.Store {
	t.Helper()
	return outboxWith(t, func(m *storefixture.MethodBuilder) {
		m.Param("ctx", storefixture.PkgNamed("context", "Context"))
		m.Param("topic", storefixture.Named("string"))
		m.Return(storefixture.RecvChan(storefixture.PkgNamed("example.com/box", "Value")))
		m.Return(storefixture.Named("error"))
	})
}

// outboxWith is an appender beside a subscriber of the caller's shape, with the
// contract stamped and the partner named.
func outboxWith(t *testing.T, subscribe func(*storefixture.MethodBuilder)) *sdk.Store {
	t.Helper()
	s := valueStore(t, func(i *storefixture.InterfaceBuilder) {
		i.Method("Append", func(m *storefixture.MethodBuilder) {
			m.Param("ctx", storefixture.PkgNamed("context", "Context"))
			m.Param("v", storefixture.PkgNamed("example.com/box", "Value"))
			m.Return(storefixture.Named("error"))
		})
		i.Method("Subscribe", subscribe)
	})
	stampContract(s, "Append", suite.ContractOutbox, suite.ContractOutboxRole)
	stampContractPartner(s, "Append", suite.ContractOutbox, suite.ContractOutboxPartner, "Subscribe")
	return s
}

// outboxSilentAppenderFixture appends without reporting whether it accepted.
func outboxSilentAppenderFixture(t *testing.T) *sdk.Store {
	t.Helper()
	s := valueStore(t, func(i *storefixture.InterfaceBuilder) {
		i.Method("Append", func(m *storefixture.MethodBuilder) {
			m.Param("ctx", storefixture.PkgNamed("context", "Context"))
			m.Param("v", storefixture.PkgNamed("example.com/box", "Value"))
		})
		i.Method("Subscribe", func(m *storefixture.MethodBuilder) {
			m.Param("ctx", storefixture.PkgNamed("context", "Context"))
			m.Return(storefixture.RecvChan(storefixture.PkgNamed("example.com/box", "Value")))
			m.Return(storefixture.Named("error"))
		})
	})
	stampContract(s, "Append", suite.ContractOutbox, suite.ContractOutboxRole)
	stampContractPartner(s, "Append", suite.ContractOutbox, suite.ContractOutboxPartner, "Subscribe")
	return s
}
