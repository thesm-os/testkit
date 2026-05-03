// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package nested

//go:generate testkit builder -o nestedtest/builders.gen.go Order

// Address is a value type used as a nested struct field.
type Address struct {
	Street string
	City   string
}

// Customer is used as both a value and pointer field.
type Customer struct {
	Name  string
	Email string
}

// Metadata is embedded in Order.
type Metadata struct {
	Source  string
	Region string
}

// Order exercises embedded structs, nested struct values, and pointer fields.
type Order struct {
	Metadata                // embedded struct
	ID       string
	Billing  Address        // nested struct value
	Shipping Address        // another nested struct value
	Customer *Customer      // pointer to struct
	Priority int
}
