// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package nodes answers the questions generators keep asking of source nodes.
//
// Every generator here ends up needing the same handful: is this reference a
// channel, does this type declare that method, which of its fields holds an
// error. Each is two or three lines, which is exactly why a generator that
// cannot find the existing one writes its own — and then two of them disagree
// about an edge case nobody re-derives on the second attempt.
//
// So these live together whether or not a second caller exists yet. The cost of
// a package holding one-line predicates is small and visible; the cost of the
// fourth private copy of "is this the byte type" is a bug that only shows up
// for `uint8`.
package nodes

import (
	"go.thesmos.sh/eidos/core/meta"
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/node"
)

// GoIsChannel and GoChanDir are the Go frontend's published facts about a
// channel. The node model has no channel kind, so a channel arrives as a named
// reference in a synthetic `go` package with these stamped beside it.
//
// Declared here rather than imported because a meta key is interned by name:
// the same call in the frontend and in this package yields the same key, and
// the frontend documents both names as part of its output.
//
//nolint:gochecknoglobals // interned keys, immutable after init.
var (
	GoIsChannel = meta.EnsureKey("go.isChannel", meta.BoolParser)
	GoChanDir   = meta.EnsureKey("go.chanDir", meta.StringParser)
)

// ChanBidirectional is the direction the Go frontend stamps for a channel that
// can be both sent to and received from.
const ChanBidirectional = "both"

// IsBidirectionalChan reports whether t is a channel that can be both sent to
// and received from.
//
// Read from the stamp rather than inferred from the shape, because the shape
// does not carry it: every channel arrives as the same named reference and the
// direction is stamped beside it.
//
// Matched positively rather than by excluding the directional spellings. A
// caller generally wants this in order to write `make(...)`, which is not legal
// on a directional channel, and a direction this does not recognise is likelier
// to be one `make` rejects than one it accepts.
func IsBidirectionalChan(t *node.TypeRef) bool {
	if t == nil {
		return false
	}
	bag := t.Meta()
	if bag == nil {
		return false
	}
	if isChannel, _ := GoIsChannel.Get(bag); !isChannel {
		return false
	}
	dir, _ := GoChanDir.Get(bag)
	return dir == ChanBidirectional
}

// IsError reports whether t is the builtin error interface.
func IsError(t *node.TypeRef) bool {
	return t != nil && t.IsBuiltin() && t.Name == "error"
}

// IsByte reports whether t is the builtin byte.
//
// Both spellings, because the frontend records whichever the author wrote —
// which is the edge case a private copy of this predicate gets wrong.
//
//nolint:goconst // a type name is its own clearest form.
func IsByte(t *node.TypeRef) bool {
	return t != nil && t.Package == "" && (t.Name == "byte" || t.Name == "uint8")
}

// IsEmptyStruct reports whether t is the anonymous empty struct, which is what
// makes a map to it a set rather than a mapping.
//
// Both emptiness tests, because an anonymous struct may carry embedded types as
// well as declared fields, and one holding either is a value a caller has
// something to say about.
func IsEmptyStruct(t *node.TypeRef) bool {
	return t != nil && t.IsAnonStruct() && len(t.Fields) == 0 && len(t.Embeds) == 0
}

// IsEmptyInterface reports whether t is `any` written as an empty interface,
// which the frontend records as an anonymous interface declaring nothing.
func IsEmptyInterface(t *node.TypeRef) bool {
	return t != nil && t.TypeKind == node.TypeRefAnonInterface &&
		len(t.Methods) == 0 && len(t.Embeds) == 0
}

// EmbedName returns the identifier an embedded type contributes as a field
// name, and whether it was embedded by pointer.
//
// An embed by pointer carries its name on the pointee rather than on the
// reference itself, so reading the reference's own name yields the empty string
// and the whole field is silently dropped.
func EmbedName(t *node.TypeRef) (name string, pointer bool) {
	switch {
	case t == nil:
		return "", false
	case t.TypeKind == node.TypeRefPointer:
		if t.Elem == nil {
			return "", false
		}
		return t.Elem.Name, true
	default:
		return t.Name, false
	}
}

// Declares reports whether s has a method of the given name.
func Declares(s *node.Struct, method string) bool {
	return find(s, method) != nil
}

// PointerReceiver reports whether the named method is declared on the pointer
// receiver, which decides whether a caller writes `&T{}` or `T{}`.
//
// False when the method is absent, so a caller that has not checked [Declares]
// gets the value form rather than a claim about a method that is not there.
func PointerReceiver(s *node.Struct, method string) bool {
	m := find(s, method)
	return m != nil && m.Receiver != nil && m.Receiver.TypeKind == node.TypeRefPointer
}

// FieldOfType returns the first exported field of s whose type is the named
// builtin, or empty when it has none.
//
// First rather than only: a type carrying two fields of one type has no answer
// this can read to "which one did you mean", and the caller wanting a
// particular one has to say so itself.
func FieldOfType(s *node.Struct, typeName string) string {
	if s == nil {
		return ""
	}
	for _, f := range s.Fields {
		if golang.IsExported(f.Name) && f.Type != nil && f.Type.IsBuiltin() && f.Type.Name == typeName {
			return f.Name
		}
	}
	return ""
}

// find returns the named method, or nil.
func find(s *node.Struct, method string) *node.Method {
	if s == nil {
		return nil
	}
	for _, m := range s.Methods {
		if m.Name == method {
			return m
		}
	}
	return nil
}
