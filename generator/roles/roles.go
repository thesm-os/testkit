// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package roles annotates the `//testkit:role` directive: the domain
// role a declared position fills in drawn inputs ("key", "payload").
// The suite tier derives its input pools from the stamps and the
// model tier draws from the same pools, so the role names the pool a
// field opens — and naming is the whole job.
//
// Its own annotator rather than a schema on the defaults plugin,
// although the two stamps feed one derivation: a role is a
// classification, not a value, and a plugin's version is a cache key
// — coupling the role vocabulary to another plugin's version would
// regenerate everything that plugin touches on every vocabulary
// change. Consumed together is not owned together.
//
// The role word is stamped verbatim and validated by its readers,
// which refuse an unknown role by name; holding the vocabulary here
// would give it a second home beside the rules tables that act on it.
package roles

import (
	"go.thesmos.sh/eidos/sdk"
	sdkgolang "go.thesmos.sh/eidos/sdk/golang"
)

// Name is the plugin's stable identifier.
const Name = "roles"

// Version composes into the pipeline's plugin fingerprint. Bump it
// whenever what gets stamped changes: a different parse, a new key,
// a new node kind.
const Version = "1.0.0"

// DirectiveName is the directive this annotator owns, written under
// testkit's namespace as `//testkit:role`.
const DirectiveName sdk.DirectiveName = "role"

// MetaRole holds the declared role, verbatim. Absent means the
// declaration filled none — an unroled field draws no pool.
//
//nolint:gochecknoglobals // meta key registration, immutable after init.
var MetaRole = sdk.EnsureKey("testkit.role", sdk.StringParser)

// Of reads a declaration's stamped role, empty for the unroled — the
// one read path, so no consumer touches the key directly.
func Of(bag *sdk.Bag) string {
	v, _ := MetaRole.Get(bag)
	return v
}

// Plugin is the roles annotator. The zero value is unusable; go
// through [New], which builds the embedded base.
type Plugin struct{ *sdkgolang.Base }

// New returns a plugin instance — an annotator with no output, for
// the reason the defaults plugin gives: it stamps metadata a later
// generator reads.
func New() *Plugin {
	return &Plugin{Base: sdkgolang.NewPlugin(Name).
		Version(Version).
		Directives(directives()...).
		Build()}
}

// directives declares the `//testkit:role` schema. One required
// positional — a role directive without a word is a line the author
// did not finish — and negation denied: a role exists exactly where
// one is declared, so deleting the line is the suppression.
func directives() []sdk.DirectiveSchema {
	return []sdk.DirectiveSchema{
		sdk.NewDirective(DirectiveName).
			Describe(
				"Names the domain role the annotated field fills in drawn "+
					"inputs — key, payload — which is what the suite tier "+
					"derives its input pools from. Takes one positional "+
					"argument, the role word; the reading tier validates the "+
					"vocabulary and refuses an unknown role by name.",
			).
			Positional("role", sdk.Required()).
			On(sdk.NodeKindField).
			DenyKeys().
			DenyNegation().
			Build(),
	}
}

// Annotate stamps every roled field, last write wins — a field
// carrying the directive twice states two intentions, and taking the
// last matches how a reader scans a line list.
//
// The type-level arm — a role on a named type declaration, which the
// bus and cache fixtures use for roled bare parameters — is recorded
// in the suite design doc and lands with that corpus slice.
func (*Plugin) Annotate(ctx *sdk.AnnotatorContext) error {
	for _, s := range ctx.Reader.Structs().Slice() {
		for _, f := range s.Fields {
			dir := sdk.Last(f.Directives(), DirectiveName)
			if dir == nil || len(dir.Args) == 0 {
				continue
			}
			MetaRole.Set(f.EnsureMeta(), dir.Args[0], Name)
		}
	}
	return nil
}
