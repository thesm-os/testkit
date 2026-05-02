module go.thesmos.sh/testkit/cmd

go 1.26.2

require (
	go.thesmos.sh/testkit v0.0.0
	go.thesmos.sh/testkit/gen v0.0.0
)

replace (
	go.thesmos.sh/testkit => ../
	go.thesmos.sh/testkit/gen => ../gen
)
