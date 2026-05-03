// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package suite implements the conformance suite generator for testkit.
//
// The generator reads Go interfaces annotated with //testkit: directives
// and produces Assert<Interface>Contract test harnesses with auto-detected
// subtests and typed plug-in extension points.
//
// # Architecture
//
// The generator follows a four-stage pipeline:
//
//	Analyze → Enrich → Validate → Render
//
// Analyze loads the Go package and extracts interface methods.
// Enrich applies directive-driven enrichers to add contract subtests.
// Validate checks for conflicting or invalid directive combinations.
// Render executes Go templates to produce the output file.
//
// # Method shape detection
//
// Each interface method is classified into a shape based on its signature.
// The shape determines which typed context (ReaderContext, WriterContext, etc.)
// is used for plug-in assertions, and which auto-detected subtests are emitted.
//
// Detection applies rules in order — first match wins. The rules use these
// definitions:
//
//   - ctx: a parameter whose type is context.Context
//   - non-ctx param: any parameter that is not context.Context (excluding variadic)
//   - error return: a return value whose type is the built-in error interface
//   - non-error return: any return value that is not error
//
// # Rules
//
// Rule 1 — StreamReader: any return type is iter.Seq[T] or iter.Seq2[T, error].
//
//	Tasks(ctx context.Context) iter.Seq2[TaskStatus, error]  → StreamReader
//	Keys(ctx context.Context) iter.Seq[string]               → StreamReader
//
// Rule 2 — Predicate: no ctx, exactly one return of type bool.
//
//	Handles(mime string) bool   → Predicate
//	IsEmpty() bool              → Predicate
//
// Rule 3 — Pure: no ctx, no error return.
//
//	ContentType() string        → Pure
//	Name() string               → Pure
//	Describe() string           → Pure
//
// Rule 4 — Reader: ctx + exactly one non-ctx param + exactly one non-error
// return + error return.
//
//	Get(ctx context.Context, id string) (Item, error)        → Reader[T, string, Item]
//	Status(ctx context.Context, id string) (TaskStatus, error) → Reader[T, string, TaskStatus]
//
// Rule 5 — Writer or Deleter: ctx + exactly one non-ctx param + error-only return.
// Defaults to Writer. The //testkit:deleter directive overrides to Deleter.
//
//	Put(ctx context.Context, item Item) error                → Writer[T, Item]
//	Delete(ctx context.Context, id string) error             → Writer[T, string]  (default)
//	//testkit:deleter
//	Cancel(ctx context.Context, id string) error             → Deleter[T, string] (with directive)
//
// Rule 6 — Writer with result: ctx + exactly one non-ctx param + exactly one
// non-error return + error return. This is the same as Rule 4 structurally,
// but Rule 4 is checked first because it has the same arity. Rule 6 only
// triggers if Rule 4 already matched (i.e., it doesn't independently fire).
//
// Rule 7 — Aggregator: ctx only (no non-ctx params) + exactly one non-error
// return + error return.
//
//	Count(ctx context.Context) (int, error)                  → Aggregator[T, int]
//	Running(ctx context.Context) (int, error)                → Aggregator[T, int]
//
// Rule 8 — Lifecycle: ctx only (no non-ctx params) + error-only return.
//
//	Ping(ctx context.Context) error                          → Lifecycle
//	Flush(ctx context.Context) error                         → Lifecycle
//	Close(ctx context.Context) error                         → Lifecycle
//
// Rule 9 — Unknown: anything that doesn't match the above. Unknown-shaped
// methods get only a smoke subtest and an untyped On<Method> accepting
// func(*testing.T, T).
//
//	Schedule(ctx context.Context, id string, interval time.Duration, fn func(context.Context) error) error → Unknown (3 non-ctx params)
//	Encode(w io.Writer, v any) error                         → Unknown (no ctx, has error)
//	Status(ctx context.Context) (string, int, error)         → Unknown (2 non-error returns)
//
// # Worked examples
//
// The following examples show how real methods get classified and why:
//
//	Count() int
//	  → no ctx, no error → Rule 3 → Pure[T, int]
//
//	Count(ctx context.Context) int
//	  → has ctx, no error → Rule 3 doesn't apply (requires !hasCtx)
//	  → Rules 4-8 all require hasError → none match
//	  → Rule 9 → Unknown
//
//	Process(ctx context.Context, data []byte) error
//	  → has ctx, 1 non-ctx param, error-only return → Rule 5 → Writer[T, []byte]
//
//	//testkit:deleter
//	Cancel(ctx context.Context, id string) error
//	  → has ctx, 1 non-ctx param, error-only return, directive present → Rule 5 → Deleter[T, string]
//
//	Status(ctx context.Context) (X, Y, error)
//	  → has ctx, 0 non-ctx params, 2 non-error returns → Rule 7 requires exactly 1 → no match
//	  → Rule 9 → Unknown
//
//	MarshalBinary(v any) ([]byte, error)
//	  → no ctx, has error → Rule 3 doesn't apply (has error)
//	  → Rules 4-8 require hasCtx → none match
//	  → Rule 9 → Unknown
//
// # Observer context alignment
//
// Plug-in primitives for each shape receive context.Context in their
// closures if and only if the method itself takes context.Context.
// Pure and Predicate methods are ctx-free by definition (rules 2-3
// require the absence of a ctx parameter), so PureContext and
// PredicateContext closures are also ctx-free. This is intentional.
package suite
