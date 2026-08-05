module go.thesmos.sh/testkit/cmd

go 1.26.5

require (
	github.com/spf13/cobra v1.10.2
	go.thesmos.sh/eidos v1.3.3
	go.thesmos.sh/eidos/cli v1.3.2
	go.thesmos.sh/testkit v0.0.0-00010101000000-000000000000
)

require (
	github.com/goccy/go-yaml v1.19.2 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
)

// The runtime module is developed in lockstep with this one and its current
// tree is not published. go.work covers the workspace build; this replace is
// what lets `go mod tidy` resolve per-module.
replace go.thesmos.sh/testkit => ../
