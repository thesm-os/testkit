// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package user is the struct-kind fixture: the input the builder generator
// reads. Builders are derived from exported fields, so the shapes that matter
// are the ones needing more than a scalar setter — slices, maps, byte bodies,
// embedded structs, and nested types.
package user

import "time"

// Address is nested inside [User], so its builder has to be reachable through
// the parent's rather than only on its own.
type Address struct {
	Street string
	City   string
}

// Role is embedded, so its fields are promoted onto [User] and a builder that
// reads only declared fields misses them.
type Role struct {
	Name  string
	Level int
}

// User carries one field of each shape a builder handles differently.
type User struct {
	Role

	//testkit:default "anonymous"
	Name string

	Age     int
	Active  bool
	Created time.Time

	// Tags is a slice, so the builder owes an Append as well as a With.
	Tags []string

	// Labels is a map, so the builder owes WithEntry and WithEntries.
	Labels map[string]string

	// Body is a byte slice, which takes a string-accepting setter rather than
	// forcing callers to convert.
	Body []byte

	// Home is a nested struct.
	Home Address

	// Manager is a self-referential pointer, which a naive builder recurses
	// into forever.
	Manager *User

	// unexported is never settable from another package, so a generated
	// builder must skip it. Its presence is the test.
	_ string
}

// Defaults returns the zero-value baseline a generated builder starts from.
func Defaults() User {
	return User{Name: "anonymous", Active: true}
}
