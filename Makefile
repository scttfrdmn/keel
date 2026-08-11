# keel — all fast-path targets set GOEXPERIMENT=simd; `stock` proves the
# scalar path builds on an unmodified toolchain (release requirement, P5).
GOEXP := GOEXPERIMENT=simd

.PHONY: build stock test test-scalar bench gate-p0 gate-p1 gate-p2 gate-p3 gate-p4 gate-p5 lint

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

lint:
	go vet ./...

gate-p%:
	scripts/gate-p$*.sh
