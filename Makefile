# keel — all fast-path targets set GOEXPERIMENT=simd; `stock` proves the
# scalar path builds on an unmodified toolchain (release requirement, P5).
GOEXP := GOEXPERIMENT=simd

.PHONY: build stock test test-scalar bench gate-p0 gate-p1 gate-p2 gate-p3 gate-p4 gate-p5 lint docs docs-serve

build:
	$(GOEXP) go build ./...

stock:
	go build ./...

test:
	$(GOEXP) go test ./...

test-scalar:
	KEEL_FORCE=scalar $(GOEXP) go test ./...

bench:
	$(GOEXP) go test -run=NONE -bench=. -benchtime=3x ./bench/...

# The citation lint resolves every DESIGN.md rule citation in the tree against
# DESIGN.md's actual structure (#85). Its self-test runs first, because the lint
# reaches only its clean path on a healthy tree and a green from it would otherwise
# say nothing about the seven branches that matter.
lint:
	go vet ./...
	bash scripts/citation-lint-test.sh
	bash scripts/citation-lint.sh

gate-p%:
	scripts/gate-p$*.sh

# The documentation site. `docs` is the whole gate — the same script CI runs, so
# there is one definition of what green means rather than a copy of the check list
# in a workflow file. It generates the extracted and linked pages first; the site
# does not build without them.
docs:
	scripts/gate-docs.sh

# Local preview. Generates first for the same reason, and serves the same tree the
# gate builds. Not part of any gate.
docs-serve:
	scripts/docs-gen.sh && mkdocs serve
