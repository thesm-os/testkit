module go.thesmos.sh/testkit/engine

go 1.26.5

require (
	github.com/anishathalye/porcupine v1.3.0
	github.com/google/go-cmp v0.7.0
	go.thesmos.sh/testkit v0.10.0
	pgregory.net/rapid v1.3.0
)

// The runtime module is developed in lockstep with this one and its current
// tree is not published. go.work covers the workspace build; this replace is
// what lets `go mod tidy` resolve per-module. It is ignored by consumers of
// this module, whose resolution comes from the require above.
replace go.thesmos.sh/testkit => ../
