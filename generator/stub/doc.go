// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package stub generates a recording test double for every interface
// annotated `//testkit:stub`, plus a companion test file that proves the
// double satisfies the interface it stands in for.
//
// The double is the substrate the other conformance tiers compose against: a
// generated suite drives it, a bench measures through it, and a model runs it
// against a reference implementation. That is why it records rather than
// merely returning — a tier that can only observe return values cannot assert
// on the interaction, which is most of what a conformance suite checks.
//
// # Directive
//
// A source interface opts in with `//testkit:stub`. The directive takes no
// positional argument and denies the negated form — a double exists exactly
// where one is declared, so removing the line is the suppression.
//
//	//testkit:stub
//	type Store interface { ... }
//
// The framework additionally accepts `out=` and `pkg=` keys on the directive,
// which route both outputs together.
//
// # Output set
//
// Two outputs flow from one annotated interface:
//
//   - Primary, untagged, suffix `_stub.gen.go`. Hosts the [KindStub] emit
//     value, rendered by `stub.double.tmpl`. Declares the source package, so
//     the double is importable by other packages' tests rather than trapped
//     in this one — which is what lets a suite in a sibling package drive it.
//   - Tagged `test`, suffix `_stub.gen_test.go`. Hosts the [KindStubTests]
//     value, rendered by `stub.test.tmpl`. The `_test.go` ending triggers the
//     framework's automatic `<pkg>_test` package shift, so the generated test
//     cannot read package-private state and exercises the double exactly the
//     way a consumer does.
//
// The tag is what makes the two independently routable. A source author
// redirects one without the other through `//testkit:out tag=test <path>`,
// project config under the plugin's `tags:` block, or the CLI
// `-o stub:test=<path>`.
//
// Both emit values append into the file's `top` slot — the region between the
// package clause and the first core decl, which is the natural placement for
// a block of whole declarations rendered by one template.
//
// # Worked example
//
// Given `users/store.go`:
//
//	package users
//
//	//testkit:stub
//	type Store interface {
//	    Get(ctx context.Context, id string) (item User, err error)
//	    Put(ctx context.Context, u User) error
//	}
//
// The primary output lands at `users/store_stub.gen.go`:
//
//	type StoreGetCall struct {
//	    Ctx  context.Context
//	    ID   string
//	    Item User
//	    Err  error
//	}
//
//	type StoreStub struct {
//	    GetFunc func(ctx context.Context, id string) (User, error)
//	    PutFunc func(ctx context.Context, u User) error
//
//	    GetCalls []StoreGetCall
//	    PutCalls []StorePutCall
//	}
//
//	func (s *StoreStub) Get(ctx context.Context, id string) (item User, err error) {
//	    item, err = s.GetFunc(ctx, id)
//	    s.GetCalls = append(s.GetCalls, StoreGetCall{Ctx: ctx, ID: id, Item: item, Err: err})
//	    return item, err
//	}
//
// and the companion at `users/store_stub.gen_test.go`, in `users_test`:
//
//	var _ users.Store = (*users.StoreStub)(nil)
//
//	func TestStoreStubRecordsGet(t *testing.T) { ... }
//
// # Recorded-call field names
//
// `Item` and `Err` above are not positional placeholders — they are the
// source's declared return names, read from [sdk.Return.Name]. A signature
// written `(item User, err error)` documents what its returns mean, and a
// recorded-call struct is the main consumer of that documentation: it is what
// a reader sees in a failure message.
//
// Returns without declared names fall back to the framework's rule
// ([golang.Sig]): the error slot is `Err`, a lone value slot is `Result`, and
// several value slots are `Result0`, `Result1`, … numbered across the value
// slots only — so adding an error return does not renumber the fields beside
// it. The blank identifier counts as unnamed, since `_` cannot be a field
// name.
//
// # Named returns on the generated methods
//
// The generated method carries the source's return names on its own signature
// — `(item User, err error)`, not `(User, error)` — so the documentation the
// interface author wrote survives into the double. The body assigns with `=`
// rather than `:=`, since named results are already declared, and returns
// explicitly rather than bare: a naked return in generated code reads as an
// omission.
//
// Propagation is all-or-nothing and falls back to unnamed returns when either
// condition fails. [golang.NamedReturnsUsable] owns that rule and explains
// both; [golang.SigOf] applies it and records the answer on the projection
// this plugin embeds in each [Method].
//
// # A nil func field panics
//
// The generated method calls its `<Method>Func` field unconditionally, so a
// double used without assigning the field for a method under test panics with
// a nil dereference rather than returning zero values.
//
// It follows from recording: the method must invoke the func, capture what
// came back, append the record, and only then return. A double that tolerated
// a nil field would have to invent return values, and inventing them is what
// makes a double lie about the system under test. The panic names the
// unassigned field, which is the fastest available diagnosis.
//
// # Imports
//
// Every cross-package and stdlib reference is expressed as an
// [sdk.NewExternal] expression rather than a hard-coded import in the
// template. The Go backend's `renderExpr` funcmap registers the referenced
// package on the rendered file's import set, so the two templates carry no
// import blocks and the same source renders correctly whether the double
// lands beside its source or in a centralised directory.
//
// The companion test is in an external test package, so identifiers from the
// source package qualify through the same mechanism. The double's own package
// is not known until the Layout phase resolves routing, which is why [Tests]
// implements the framework's output-package callback rather than computing a
// path here.
//
// References into testkit's own runtime use the backend's `external` builtin
// too. The import paths come off the emit value — see [RuntimePaths] — so the
// module path is one Go constant rather than a literal in each template, and
// the plugin registers no template function to resolve them.
//
// # Embedded interfaces
//
// A double stands in for the interface's whole method set, embeds included: a
// double short one embedded method does not satisfy the interface it doubles,
// which is the one thing a double has to do. Resolution is
// [sdk.StoreReader.MethodSet]'s, and so is the order — the interface's own
// declarations first, then each embed's contribution in embed order. That
// order fixes the generated field order, so it is part of what a consumer
// upgrading this plugin sees change.
//
// An embed that contributed nothing is reported and the interface is skipped.
// An embed this run did not load is a warning, because a narrow invocation is
// legitimate and one unreachable dependency should not cost a project the rest
// of its doubles; a non-interface or parameterised embed is an error, because
// no wider run repairs it. A type-set term carries no name and is passed over
// — a constraint is never a stub target.
//
// # Hazards
//
// Generators run inside eidos's pipeline and may execute concurrently with
// others in the same priority bucket, so the plugin holds no state across
// [Plugin.Generate] calls. Reads go through the per-plugin store reader
// because captured reads form the cache key — bypassing it yields output that
// is stale but looks current.
//
// Output must be byte-identical across runs and machines. Map iteration is
// the usual way that breaks, so anything the templates range over is ordered
// before it reaches them.
package stub
