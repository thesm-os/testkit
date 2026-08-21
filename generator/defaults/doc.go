// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package defaults owns testkit's `//testkit:default` directive: the value a
// generated constructor seeds a field with when the caller supplies none.
//
// # Why its own plugin
//
// The builder generator is the first reader, not the only one. A fixture
// generator seeds the same field the same way, and a model tier needs the
// declared default to state what "unset" means. A directive may be registered
// once per run, so declaring it inside any one generator would make the others
// depend on that generator being registered.
//
// # Where the readers are
//
// This package writes; [go.thesmos.sh/testkit/generator/internal/stamp] reads.
// A generator wanting the declared default wants two strings, and importing
// the annotator to get them would also put [New] within its reach — which is
// how a generator ends up able to register a second copy of an annotator it
// only meant to read from.
//
// # The literal is carried verbatim
//
// The directive's argument is Go source and is stamped unparsed. `"localhost"`,
// `8080`, `true`, `0.75` and `nil` all reach a template as themselves, which
// costs nothing and avoids a type-directed parser that would have to know every
// literal form Go admits — and would have to be told the field's type to tell
// `0` from `0.0`.
//
// What is checked is that the argument parses as a Go expression. A typo then
// fails here, positioned at the directive, rather than in the consumer's
// compiler against generated code they did not write.
//
// # An explicit zero is not an absent directive
//
// `//testkit:default 0` stamps. A generator reading the stamp sees a value; one
// reading a bare zero cannot tell "seed this to zero" from "no default given",
// and would emit the same constructor either way. That distinction is why the
// stamp is a string rather than a typed value: the empty string is the only
// absence.
package defaults
