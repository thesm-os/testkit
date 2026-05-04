module go.thesmos.sh/testkit/gen

go 1.26.2

replace (
	go.thesmos.sh/testkit => ../
	go.thesmos.sh/testkit/model => ../model
)

require (
	go.thesmos.sh/testkit v0.0.0-00010101000000-000000000000
	golang.org/x/tools v0.44.0
)

require (
	github.com/google/go-cmp v0.7.0 // indirect
	golang.org/x/mod v0.35.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
)
