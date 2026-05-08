// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package structs

//go:generate testkit builder -o structstest/nested.gen.go Order Holder

// Address is a nested-struct field type.
type Address struct {
	Street string
	City   string
}

// Customer is referenced as a pointer field.
type Customer struct {
	Name  string
	Email string
}

// Metadata is embedded in Order.
type Metadata struct {
	Source string
	Region string
}

// Order exercises the cases the builder generator handles outside
// the scalar/slice/map/bytes path: an embedded struct, two named
// nested struct values, and a pointer-to-struct field. Each shape
// gets the basic With<Field> setter (no specialized variants).
type Order struct {
	Metadata          // embedded
	ID       string
	Billing  Address  // nested struct value
	Shipping Address  // another nested value
	Customer *Customer // pointer to struct
	Priority int
}

// Stringer is the local alias for fmt.Stringer used by Holder. We
// declare it here rather than importing fmt directly so the fixture
// stays self-contained — interface fields work the same regardless
// of where the interface is declared.
type Stringer interface {
	String() string
}

// Holder exercises the rarer-but-valid scalar field shapes the
// builder generator must handle gracefully via the basic With<Field>
// setter: an interface-typed field, a function-typed field, and a
// channel-typed field. Holder has no basic-comparable field — the
// generator must skip the Mutate/Clone independence subtests for
// this struct (no scalar to assert on) but still emit a working
// builder with per-field setters.
type Holder struct {
	Stringer Stringer      // interface field
	OnFail   func(error)   // function-typed field
	Done     chan struct{} // chan field
}
