module github.com/scttfrdmn/keel/gonumblas

go 1.26

require (
	github.com/scttfrdmn/keel v0.2.4
	gonum.org/v1/gonum v0.17.0
)

// Development only: the published module will require the first keel release
// cut after gonumblas moved out of the root module (v0.2.4 still contains it, so
// requiring it is an ambiguous import). Consumers ignore this replace.
replace github.com/scttfrdmn/keel => ../
