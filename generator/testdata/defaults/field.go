// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package defaults

//go:generate testkit builder -o defaultstest/field.gen.go Config

// Config seeds its builder via per-field //testkit:default
// directives. When New<Config>() is called, the builder constructs
// a value with each annotated field set to the directive's
// argument; un-annotated fields stay zero-valued. This mechanism
// runs without any sibling defaults function.
type Config struct {
	Host    string //testkit:default "localhost"
	Port    int    //testkit:default 8080
	Verbose bool   //testkit:default true
	Name    string // no default — uses zero value
}
