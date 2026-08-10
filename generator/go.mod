module go.thesmos.sh/testkit/generator

go 1.26.5

require (
	go.thesmos.sh/eidos v1.13.2
	go.thesmos.sh/eidos/backend/golang v1.13.2
	go.thesmos.sh/eidos/eidostest v1.13.2
	go.thesmos.sh/eidos/frontend/golang v1.13.2
	go.thesmos.sh/eidos/plugins v1.13.2
	go.thesmos.sh/testkit v0.0.0-00010101000000-000000000000
)

require (
	github.com/google/go-cmp v0.7.0 // indirect
	golang.org/x/mod v0.38.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/tools v0.48.0 // indirect
)

// The runtime module is developed in lockstep with this one and its current
// tree is not published. go.work covers the workspace build; this replace is
// what lets `go mod tidy` resolve per-module.
replace go.thesmos.sh/testkit => ../
