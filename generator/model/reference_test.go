// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model_test

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/generator/internal/gentest"
	"go.thesmos.sh/testkit/generator/model"
)

// TestSuppliedReference pins the escape hatch: ref= replaces the derivation
// wholesale, so nothing else needs to be derivable.
func TestSuppliedReference(t *testing.T) {
	t.Parallel()

	s := mixed(t, storefixture.KV(model.RefKey, "NewFake"))
	b := bindingsOf(t, s)
	testkit.True(t, b.Reference.Supplied(), "the directive named the constructor")
	testkit.Equal(t, len(b.Adapter), 0, "and no adapter is generated over it")

	t.Run("an empty value is no supply", func(t *testing.T) {
		t.Parallel()
		// `ref=` with nothing after it reads as a slip, not a choice; the
		// derivation proceeds as though the key were absent.
		b := bindingsOf(t, mixed(t, storefixture.KV(model.RefKey, "")))
		testkit.False(t, b.Reference.Supplied(), "the empty value fell through to derivation")
	})

	t.Run("a reader alone draws no values", func(t *testing.T) {
		t.Parallel()
		// The pool declarations are conditional on use because an unused local
		// is a compile error in every generated file. ref= is what makes a
		// one-sided interface reachable at all.
		s := storefixture.New().
			Package("ro", "example.com/ro").
			Interface("Fetcher", func(i *storefixture.InterfaceBuilder) {
				i.Pos(gentest.AtFile("ro/iface.go"))
				i.Directive(storefixture.Directive("suite"))
				i.Directive(storefixture.Directive("model",
					storefixture.KV(model.RefKey, "NewFake")))
				i.Method("Fetch", func(m *storefixture.MethodBuilder) {
					gentest.Ctx(m)
					m.Param("key", storefixture.Named("string"))
					m.Return(storefixture.Named("string"))
					gentest.Err(m)
				})
			}).
			Build()
		stampShape(s, "Fetch", "reader", "string", "string")
		b := bindingsOf(t, s)
		testkit.True(t, b.UsesKeys(), "the reader draws keys")
		testkit.False(t, b.UsesValues(), "and nothing draws values")
	})

	t.Run("a writer alone draws no keys", func(t *testing.T) {
		t.Parallel()
		s := storefixture.New().
			Package("wo", "example.com/wo").
			Interface("Sink", func(i *storefixture.InterfaceBuilder) {
				i.Pos(gentest.AtFile("wo/iface.go"))
				i.Directive(storefixture.Directive("suite"))
				i.Directive(storefixture.Directive("model",
					storefixture.KV(model.RefKey, "NewFake")))
				i.Method("Push", func(m *storefixture.MethodBuilder) {
					gentest.Ctx(m)
					m.Param("v", storefixture.Named("string"))
					gentest.Err(m)
				})
			}).
			Build()
		stampShape(s, "Push", "writer", "", "string")
		b := bindingsOf(t, s)
		testkit.True(t, b.UsesValues(), "the writer draws values")
		testkit.False(t, b.UsesKeys(), "and nothing draws keys")
	})
}

// TestTwinFloor walks the fallback that arms what no store models: the
// reference becomes the subject's own factory, the header carries why, and
// nothing derived — no adapter, no companion — rides along.
func TestTwinFloor(t *testing.T) {
	t.Parallel()

	twinOf := func(t *testing.T, s *sdk.Store) *model.Bindings {
		t.Helper()
		b := bindingsOf(t, s)
		testkit.True(t, b.Reference.Twin(), "the floor is the twin")
		testkit.False(t, b.Reference.Derived(), "which is not a derivation")
		testkit.Equal(t, len(b.Adapter), 0, "so no adapter is generated")
		return b
	}

	t.Run("a reader alone", func(t *testing.T) {
		t.Parallel()
		b := twinOf(t, readerOnly(t))
		testkit.Assert(t, b.Reference.TwinWhy).Contains("no reader/writer pair",
			"the header says what was missing")
	})

	t.Run("reader and writer disagree about the value", func(t *testing.T) {
		t.Parallel()
		b := twinOf(t, kvStore(t, "example.com/kv.Doc", "string"))
		testkit.Assert(t, b.Reference.TwinWhy).Contains("where the writer takes",
			"two value types share no store, but twins compare anything")
	})

	t.Run("the value declares no struct to key on", func(t *testing.T) {
		t.Parallel()
		b := twinOf(t, kvStore(t, "string", "string"))
		testkit.Assert(t, b.Reference.TwinWhy).Contains("no struct declaration",
			"naming what the key projection needed")
	})

	t.Run("no field carries the key type", func(t *testing.T) {
		t.Parallel()
		b := twinOf(t, kvStore(t, "example.com/kv.Doc", "example.com/kv.Doc",
			field{"N", "int"}))
		testkit.Assert(t, b.Reference.TwinWhy).Contains("no field",
			"nothing to project a key out of")
	})

	t.Run("several fields tie without a conventional name", func(t *testing.T) {
		t.Parallel()
		b := twinOf(t, kvStore(t, "example.com/kv.Doc", "example.com/kv.Doc",
			field{"A", "string"}, field{"B", "string"}))
		testkit.Assert(t, b.Reference.TwinWhy).Contains("several fields",
			"an arbitrary pick would key a real oracle on a guess")
		testkit.True(t, b.Values.Wide, "and the twin stays wide unpinned — twins agree on misses")
		testkit.Equal(t, b.Values.Pin, "", "with no field to pin")
	})

	t.Run("a second writer of another type is skipped by name", func(t *testing.T) {
		t.Parallel()
		s := mixedWith(t, func(i *storefixture.InterfaceBuilder) {
			i.Method("StoreRaw", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("v", storefixture.Named("string"))
				gentest.Err(m)
			})
		})
		stampShape(s, "StoreRaw", "writer", "", "string")
		b := bindingsOf(t, s)
		testkit.Equal(t, b.Reference.StoreType(), "MapStore",
			"the matching writer still derives the map")
		reasons := map[string]string{}
		for _, sk := range b.Skipped {
			reasons[sk.Method] = sk.Reason
		}
		testkit.Assert(t, reasons["StoreRaw"]).Contains("values pool draws",
			"the odd writer is listed, not miscompiled")
	})
}

// TestKeyedReference walks the keyed-put derivation: a composite writer
// selects the keyed store, needs no projection, and the delete's mixin — not
// its shape — names its oracle operation.
func TestKeyedReference(t *testing.T) {
	t.Parallel()

	b := bindingsOf(t, keyedStore(t, "example.com/kv.ErrGone"))

	t.Run("the composite writer selects the keyed oracle", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, b.Reference.Keyed(), "the key is an argument, not a projection")
		testkit.Equal(t, b.Reference.StoreType(), "KeyedStore", "and the store is the keyed one")
		testkit.Equal(t, b.Reference.KeyField, "", "with no field to project")
	})

	t.Run("the delete delegates by its mixin, not its shape", func(t *testing.T) {
		t.Parallel()
		ops := map[string]string{}
		for _, am := range b.Adapter {
			ops[am.Sig.Name] = am.Op
		}
		testkit.Equal(t, ops["Del"], "Delete", "writer-shaped, delete-stamped")
		testkit.Equal(t, ops["Put"], "Put", "the composite put delegates as itself")
		testkit.Equal(t, ops["Get"], "Get", "and the reader reads")
	})

	t.Run("the delete draws keys, the put draws both", func(t *testing.T) {
		t.Parallel()
		pools := map[string]string{}
		for _, a := range b.Actions {
			pools[a.Method] = a.Pool
		}
		testkit.Equal(t, pools["Del"], "keys", "a delete's argument is a key")
		testkit.Equal(t, b.Values.Field, "Value", "values come from the put's second argument")
		testkit.True(t, b.UsesKeys() && b.UsesValues(), "both pools are declared")
	})

	t.Run("an identical second selection binds once", func(t *testing.T) {
		t.Parallel()
		// Two delete-stamped writers naming the same read partner select the
		// delete law twice with one field set; the registry must carry it
		// once, not report one failure twice.
		s := keyedStoreWith(t, "example.com/kv.ErrGone", func(i *storefixture.InterfaceBuilder) {
			i.Method("Del2", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("key", storefixture.Named("string"))
				gentest.Err(m)
			})
		})
		stampShape(s, "Del2", "writer", "string", "")
		for _, iface := range s.Nodes().Interfaces().Items() {
			for _, m := range iface.Methods {
				if m.Name != "Del2" {
					continue
				}
				bag := m.EnsureMeta()
				shape.MetaMixins.Set(bag, []string{"deleteremoves"}, "test")
				shape.MixinParamKey("deleteremoves", "read").
					Set(bag, "example.com/kv.Get", "test")
				shape.MixinParamKey("deleteremoves", "sentinel").
					Set(bag, "example.com/kv.ErrGone", "test")
			}
		}
		got := bindingsOf(t, s)
		bound := 0
		for _, l := range got.Laws {
			if l.ID == lawid.DeleteReturnsNotFound {
				bound++
			}
		}
		testkit.Equal(t, bound, 1, "one binding, whichever carrier selected it")
	})

	t.Run("a partner stamp cleared refuses by name", func(t *testing.T) {
		t.Parallel()
		s := keyedStore(t, "example.com/kv.ErrGone")
		for _, iface := range s.Nodes().Interfaces().Items() {
			for _, m := range iface.Methods {
				if m.Name == "Del" {
					shape.MixinParamKey("deleteremoves", "read").Set(m.EnsureMeta(), "", "test")
				}
			}
		}
		got := bindingsOf(t, s)
		unbound := map[string]string{}
		for _, u := range got.Unbound {
			unbound[u.Method] = u.Reason
		}
		testkit.Assert(t, unbound[lawid.DeleteReturnsNotFound]).Contains("does not stamp",
			"the missing stamp is the named cause")
	})

	t.Run("a partner naming a non-method refuses by name", func(t *testing.T) {
		t.Parallel()
		s := keyedStore(t, "example.com/kv.ErrGone")
		for _, iface := range s.Nodes().Interfaces().Items() {
			for _, m := range iface.Methods {
				if m.Name == "Del" {
					shape.MixinParamKey("deleteremoves", "read").
						Set(m.EnsureMeta(), "example.com/kv.Nowhere", "test")
				}
			}
		}
		got := bindingsOf(t, s)
		unbound := map[string]string{}
		for _, u := range got.Unbound {
			unbound[u.Method] = u.Reason
		}
		testkit.Assert(t, unbound[lawid.DeleteReturnsNotFound]).Contains("not a method",
			"the dangling name is the named cause")
	})

	t.Run("a partner reader off the pools' types refuses by name", func(t *testing.T) {
		t.Parallel()
		// The delete's partner is a second reader answering int beside the
		// string pools, which no generated closure can reconcile — and being
		// the later declaration it also carries the pools, so the store
		// falls to the twin while the law refuses on the types.
		s := keyedStoreWith(t, "example.com/kv.ErrGone", func(i *storefixture.InterfaceBuilder) {
			i.Method("GetOther", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("key", storefixture.Named("string"))
				m.Return(storefixture.Named("int"))
				gentest.Err(m)
			})
		})
		stampShape(s, "GetOther", "reader", "string", "int")
		for _, iface := range s.Nodes().Interfaces().Items() {
			for _, m := range iface.Methods {
				if m.Name == "Del" {
					shape.MixinParamKey("deleteremoves", "read").
						Set(m.EnsureMeta(), "example.com/kv.GetOther", "test")
				}
			}
		}
		got := bindingsOf(t, s)
		unbound := map[string]string{}
		for _, u := range got.Unbound {
			unbound[u.Method] = u.Reason
		}
		testkit.Assert(t, unbound[lawid.DeleteReturnsNotFound]).Contains("beside pools of",
			"the type disagreement is the named cause")
	})

	t.Run("the sentinel binds from the stamp", func(t *testing.T) {
		t.Parallel()
		var bound *model.LawBinding
		for _, l := range b.Laws {
			if l.ID == lawid.DeleteReturnsNotFound {
				bound = l
			}
		}
		testkit.True(t, bound != nil, "the delete law binds")
		testkit.Equal(t, bound.Fields[0].Method, "Get",
			"Read resolves through the deleteremoves.read partner stamp")
		testkit.True(t, bound.Fields[2].Const != nil,
			"Sentinel renders the stamped constant")
		testkit.Equal(t, bound.Kind(), sdk.Kind("model.law"),
			"a binding renders through the one law template")
		testkit.Equal(t, bound.Fields[0].ModelPkg(), model.ModelPkg,
			"and its closures reach the runner's package")
	})
}

// TestKeyedMismatchFallsToTheTwin pins the keyed fork's agreement: the reader
// and the keyed writer speak one (K, V) pair, or no store models them and the
// twin floor stands in with the disagreement in the header.
func TestKeyedMismatchFallsToTheTwin(t *testing.T) {
	t.Parallel()

	s := keyedStore(t, "example.com/kv.ErrGone")
	stampShape(s, "Get", "reader", "int", "string")
	b := bindingsOf(t, s)
	testkit.True(t, b.Reference.Twin(), "one store cannot model the pair")
	testkit.Assert(t, b.Reference.TwinWhy).Contains("keyed writer takes",
		"and the header spells the disagreement")
}

// TestConventionalKeyFieldBreaksTheTie pins the preference order: among
// several fields of the key's type, ID outranks everything, because that is
// the name an author gives the field that is the identity.
func TestConventionalKeyFieldBreaksTheTie(t *testing.T) {
	t.Parallel()

	s := kvStore(t, "example.com/kv.Doc", "example.com/kv.Doc",
		field{"Name", "string"}, field{"ID", "string"})
	b := bindingsOf(t, s)
	testkit.Equal(t, b.Reference.KeyField, "ID", "the conventional spelling wins")
}

// TestDrainMismatchFallsToTheTwin pins the collection fork's one agreement:
// the writer adds what the collector returns, or no collection models the
// pair and the twin floor stands in.
func TestDrainMismatchFallsToTheTwin(t *testing.T) {
	t.Parallel()

	s := drainStore(t, false)
	stampShape(s, "Add", "writer", "", "example.com/bag.Other")
	b := bindingsOf(t, s)
	testkit.True(t, b.Reference.Twin(), "one collection cannot model the pair")
	testkit.Assert(t, b.Reference.TwinWhy).Contains("the writer adds",
		"and the header spells the disagreement")
}

// TestOracleDefeatingClaims pins the eventually arm: reads may lag writes, so
// every immediate oracle mis-models the subject and the twins stand in.
func TestOracleDefeatingClaims(t *testing.T) {
	t.Parallel()

	s := mixed(t)
	for _, iface := range s.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			if m.Name == "Read" {
				shape.MetaMixins.Set(m.EnsureMeta(), []string{"eventually"}, "test")
			}
		}
	}
	b := bindingsOf(t, s)
	testkit.True(t, b.Reference.Twin(), "no immediate store models the lag")
	testkit.Assert(t, b.Reference.TwinWhy).Contains("lag",
		"and the header names the claim's slack")
}

// TestContractOracle walks the role-stamped derivation over the lease
// family: the carrier's role= names its own part, the partner key names the
// sibling, the store's type argument is the acquire key's, and the
// constructor's sentinels follow the claims — minted where the strict
// dialect holds, lenified to the twin where every sentinel would be nil,
// because the kill matrix proved a never-disagreeing oracle checks nothing.
func TestContractOracle(t *testing.T) {
	t.Parallel()

	leased := func(t *testing.T, idempotent bool) *sdk.Store {
		t.Helper()
		return leasedWith(t, idempotent, nil)
	}

	t.Run("the strict dialect derives the tracker", func(t *testing.T) {
		t.Parallel()
		b := bindingsOf(t, leased(t, false))
		testkit.True(t, b.Reference.IsContract(), "the claim outranks the shapes")
		testkit.Equal(t, b.Reference.StoreType(), "LeaseTracker", "to its own store")
		ops := map[string]string{}
		for _, am := range b.Adapter {
			ops[am.Sig.Name] = am.Op
		}
		testkit.Equal(t, ops["Acquire"], "Acquire", "the carrier delegates by role")
		testkit.Equal(t, ops["Release"], "Release", "and the partner beside it")
		testkit.Equal(t, len(b.Reference.CtorErrs), 2, "two constructor slots")
		testkit.True(t, b.Reference.CtorErrs[0].Name != "", "the held sentinel is minted")
		testkit.Equal(t, b.Reference.CtorErrs[1].Name, "", "and the release slot stays lenient")
	})

	t.Run("the idempotent claim lenifies to the twin", func(t *testing.T) {
		t.Parallel()
		b := bindingsOf(t, leased(t, true))
		testkit.True(t, b.Reference.Twin(), "a never-disagreeing oracle checks nothing")
		testkit.Assert(t, b.Reference.TwinWhy).Contains("lenify",
			"and the header says which claims did it")
	})

	t.Run("non-role methods stay inert on the contract oracle", func(t *testing.T) {
		t.Parallel()
		s := leasedWith(t, false, func(i *storefixture.InterfaceBuilder) {
			i.Method("Count", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Return(storefixture.Named("int"))
				gentest.Err(m)
			})
		})
		stampShape(s, "Count", "aggregator", "", "int")
		b := bindingsOf(t, s)
		testkit.True(t, b.Reference.IsContract(), "the roles still resolve")
		inert := map[string]string{}
		for _, am := range b.Adapter {
			if am.Op == "" {
				inert[am.Sig.Name] = am.Reason
			}
		}
		testkit.Assert(t, inert["Count"]).Contains("only its roles",
			"a method outside the vocabulary is inert, with why")
	})

	t.Run("selection deduplicates across the role carriers", func(t *testing.T) {
		t.Parallel()
		// The classification rides every role method; re-selecting its laws
		// from each carrier must not register one law twice or print one
		// refusal per method.
		s := leasedWith(t, false, nil)
		for _, iface := range s.Nodes().Interfaces().Items() {
			for _, m := range iface.Methods {
				if m.Name == "Release" {
					shape.MetaContracts.Set(m.EnsureMeta(), []string{"lease"}, "test")
				}
			}
		}
		b := bindingsOf(t, s)
		seen := map[string]int{}
		for _, u := range b.Unbound {
			seen[u.Method+"\x00"+u.Reason]++
		}
		for key, n := range seen {
			testkit.Equal(t, n, 1, "one refusal per (law, reason): "+key)
		}
	})

	t.Run("a partner naming nothing leaves the role unresolved", func(t *testing.T) {
		t.Parallel()
		s := leasedWith(t, false, nil)
		for _, iface := range s.Nodes().Interfaces().Items() {
			for _, m := range iface.Methods {
				if m.Name == "Acquire" {
					shape.ContractPartnerKey("lease", "release").
						Set(m.EnsureMeta(), "example.com/ls.Nowhere", "test")
				}
			}
		}
		b := bindingsOf(t, s)
		testkit.False(t, b.Reference.IsContract(),
			"a role resolved to nothing derives nothing")
	})

	t.Run("a carrier without a role stamp falls through", func(t *testing.T) {
		t.Parallel()
		s := leasedWith(t, false, nil)
		for _, iface := range s.Nodes().Interfaces().Items() {
			for _, m := range iface.Methods {
				if m.Name == "Acquire" {
					shape.ContractRoleKey("lease").Set(m.EnsureMeta(), "", "test")
				}
			}
		}
		got := bindingsOf(t, s)
		testkit.False(t, got.Reference.IsContract(),
			"no role, no delegation to hang the oracle on")
	})

	t.Run("a type-arg role speaking no type falls through", func(t *testing.T) {
		t.Parallel()
		s := storefixture.New().
			Package("ls", "example.com/ls").
			Interface("Locker", func(i *storefixture.InterfaceBuilder) {
				i.Pos(gentest.AtFile("ls/iface.go"))
				i.Directive(storefixture.Directive("suite"))
				i.Directive(storefixture.Directive("model"))
				// The acquire role takes nothing, so the tracker's type
				// argument has nowhere to come from.
				i.Method("Acquire", func(m *storefixture.MethodBuilder) {
					gentest.Ctx(m)
					gentest.Err(m)
				})
				i.Method("Release", func(m *storefixture.MethodBuilder) {
					gentest.Ctx(m)
					gentest.Err(m)
				})
			}).
			Build()
		stampShape(s, "Acquire", "lifecycle", "", "")
		stampShape(s, "Release", "lifecycle", "", "")
		for _, iface := range s.Nodes().Interfaces().Items() {
			for _, m := range iface.Methods {
				if m.Name != "Acquire" {
					continue
				}
				bag := m.EnsureMeta()
				shape.MetaContracts.Set(bag, []string{"lease"}, "test")
				shape.ContractRoleKey("lease").Set(bag, "acquire", "test")
				shape.ContractPartnerKey("lease", "release").
					Set(bag, "example.com/ls.Release", "test")
			}
		}
		got := bindingsOf(t, s)
		testkit.False(t, got.Reference.IsContract(),
			"an oracle with no type argument cannot be instantiated")
	})

	t.Run("an unresolved role falls through to the shapes", func(t *testing.T) {
		t.Parallel()
		s := leased(t, false)
		for _, iface := range s.Nodes().Interfaces().Items() {
			for _, m := range iface.Methods {
				if m.Name == "Acquire" {
					shape.ContractPartnerKey("lease", "release").Set(m.EnsureMeta(), "", "test")
				}
			}
		}
		b := bindingsOf(t, s)
		testkit.False(t, b.Reference.IsContract(),
			"half a lease checks nothing a twin does not")
	})
}

// TestUnqualifiedSentinelIsRefused pins the constant renderer's refusal: a
// stamp without a package is a name the generator cannot import — bare or
// dangling, the two ways a qualifier can fail to be one.
func TestUnqualifiedSentinelIsRefused(t *testing.T) {
	t.Parallel()

	for _, sentinel := range []string{"ErrGone", "example.com/kv."} {
		b := bindingsOf(t, keyedStore(t, sentinel))
		unbound := map[string]string{}
		for _, u := range b.Unbound {
			unbound[u.Method] = u.Reason
		}
		testkit.Assert(t, unbound[lawid.DeleteReturnsNotFound]).Contains("neither a qualified symbol",
			"the refusal names what the stamp is missing")
	}
}
