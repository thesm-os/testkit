module go.thesmos.sh/testkit/conformance

go 1.26.5

require (
	go.thesmos.sh/eidos v1.14.1
	go.thesmos.sh/eidos/backend/golang v1.13.3
	go.thesmos.sh/eidos/frontend/golang v1.14.0
	go.thesmos.sh/eidos/plugins v1.14.1-0.20260811174532-bc257049dd79
	go.thesmos.sh/testkit v0.10.0
	go.thesmos.sh/testkit/engine v0.0.0-00010101000000-000000000000
	go.thesmos.sh/testkit/generator v0.0.0-00010101000000-000000000000
)

require (
	github.com/google/go-cmp v0.7.0 // indirect
	golang.org/x/mod v0.38.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/tools v0.48.0 // indirect
	pgregory.net/rapid v1.3.0 // indirect
)

// The runtime and generator modules are developed in lockstep with this one
// and their current trees are not published. go.work covers the workspace
// build; these replaces are what let `go mod tidy` resolve per-module.
replace (
	go.thesmos.sh/testkit => ../
	go.thesmos.sh/testkit/engine => ../engine
	go.thesmos.sh/testkit/generator => ../generator
)
