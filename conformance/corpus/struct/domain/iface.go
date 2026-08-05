// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package domain is the struct-kind fixture: the input the builder generator
// reads.
//
// A builder is derived per field, and the emitted setter depends on the
// field's type rather than on its name — a slice owes an Append as well as a
// With, a map owes WithEntry and WithEntries, a byte slice owes a
// string-accepting setter so callers need not convert, and an unexported field
// owes nothing at all. So the fixture's job is to carry one field of every
// shape that changes the answer, not to look like a realistic domain type.
//
// [UserDefaults] and [AddressDefaults] are the companions generated builders
// seed from. The package holds several types on purpose: a companion named
// Defaults would collide on the second, which is why the convention carries
// the type name. A single-type fixture cannot show that, and it also makes the
// name stutter against its own package.
//
// The types are split by what varies rather than collected into one struct:
// [Primitives] covers the scalar spread where the only difference is width and
// signedness, [Containers] covers the shapes needing more than one setter, and
// [User] covers composition — embedding, nesting, and self-reference.
package domain

import (
	"context"
	"io"
	"time"
)

// Weekday is a defined type over a builtin. A builder must set it as Weekday
// rather than as its underlying int, or callers lose the type.
type Weekday int

// Celsius is a defined type over a float, present so the numeric spread covers
// a named float as well as named integers.
type Celsius float64

// ID is a defined type over a string, the case where a naive builder emits a
// string setter and silently discards the type.
type ID string

// Bytes is an alias rather than a defined type. An alias is its underlying
// type, so it must take the same setter as []byte and not a distinct one.
type Bytes = []byte

// Primitives carries one field of every builtin scalar type.
//
// Width and signedness do not change the setter's shape, which is exactly why
// they belong in one struct: if a builder handles int and int64 differently,
// that is a bug this fixture surfaces by having both.
type Primitives struct {
	Bool bool

	String String
	Rune   rune
	Byte   byte

	Int   int
	Int8  int8
	Int16 int16
	Int32 int32
	Int64 int64

	Uint    uint
	Uint8   uint8
	Uint16  uint16
	Uint32  uint32
	Uint64  uint64
	Uintptr uintptr

	Float32 float32
	Float64 float64

	Complex64  complex64
	Complex128 complex128

	// Named types over builtins. The setter must preserve the defined type.
	Day  Weekday
	Temp Celsius
	Ref  ID
}

// String is a defined string type used as a field type in [Primitives], so the
// spread includes a named string alongside the builtin.
type String = string

// Containers carries every shape that owes more than a single With setter.
type Containers struct {
	// Slice owes both With and Append.
	Slice []string

	// SliceOfStruct owes an Append taking the element type, which a builder
	// that only handles scalar elements gets wrong.
	SliceOfStruct []Address

	// Array is fixed-length, so it cannot take an Append and must reject one.
	Array [3]int

	// Bytes owes a string-accepting setter so callers need not convert.
	Body []byte

	// Aliased is []byte under another name and must take the same setter.
	Aliased Bytes

	// Map owes WithEntry for one pair and WithEntries for many.
	Map map[string]string

	// MapOfStruct has a composite value, so WithEntry's second parameter is
	// not a scalar.
	MapOfStruct map[string]Address

	// Set is a map to empty struct, the idiomatic set. Its WithEntry takes no
	// value parameter at all.
	Set map[string]struct{}

	// Pointer distinguishes unset from zero, so its setter takes a value and
	// takes the address rather than requiring the caller to.
	Pointer *int

	// Interface can only be set, never constructed by the builder.
	Reader io.Reader

	// Func is settable but not comparable, which breaks any builder that
	// derives equality from the field set.
	Callback func(context.Context) error

	// Chan is settable but a builder must not construct one, since capacity is
	// a decision the caller owns.
	Events chan string

	// Any holds an arbitrary value and must not be special-cased into an
	// interface setter.
	Extra any

	// Err is an interface with a well-known name, present because a builder
	// that special-cases the identifier rather than the type gets it wrong.
	Err error
}

// Address is nested inside [User] and used as a slice and map element in
// [Containers], so its own builder has to be reachable from three directions.
type Address struct {
	Street string
	City   string
	Postal string
}

// Role is embedded into [User], so its fields are promoted. A builder reading
// only declared fields misses Name and Level entirely.
type Role struct {
	Name  string
	Level int
}

// Audit is embedded by pointer rather than by value. The promoted fields are
// only reachable once the pointer is non-nil, so a builder has to allocate
// before setting through it.
type Audit struct {
	CreatedAt time.Time
	CreatedBy string
}

// User covers composition: embedding by value and by pointer, nesting, self
// reference, and the unexported field a builder must skip.
type User struct {
	// Embedded by value — fields promote directly.
	Role

	// Embedded by pointer — fields promote only through an allocation.
	*Audit

	//testkit:default "anonymous"
	Username string

	Age    int
	Active bool

	// Home is a nested struct set as a whole.
	Home Address

	// Manager is self-referential. A builder that recurses into nested struct
	// types without a cycle check never terminates here.
	Manager *User

	// Reports is a self-referential slice, the same cycle by another route.
	Reports []*User

	// Deadline is a stdlib struct the builder must treat as opaque rather than
	// walking into its unexported fields.
	Deadline time.Time

	// Timeout is a defined type over int64 in the stdlib, which must keep its
	// type rather than degrading to a bare duration count.
	Timeout time.Duration

	// unexported is not settable from another package, so a generated builder
	// must skip it. Its presence is the test.
	_ string
}

// UserDefaults is the companion the builder generator seeds New<User>() from.
//
// The values are deliberately non-zero across several field shapes. A seed
// that happened to equal the zero value would make the companion path
// indistinguishable from having no companion at all, which is the branch
// [go.thesmos.sh/testkit/conformance/corpus/struct/plain] covers.
func UserDefaults() User {
	return User{
		Role:     Role{Name: "member", Level: 1},
		Username: "anonymous",
		Age:      30,
		Active:   true,
		Home:     Address{City: "Amsterdam"},
		Deadline: time.Unix(0, 0).UTC(),
		Timeout:  5 * time.Second,
	}
}

// AddressDefaults is [Address]'s companion. Its presence is the point: two
// companions in one package is what the <Type>Defaults convention exists for,
// and a fixture with one cannot demonstrate it.
func AddressDefaults() Address {
	return Address{Street: "Keizersgracht 1", City: "Amsterdam", Postal: "1015"}
}
