![keel — pure-Go float32 BLAS, SIMD-accelerated linear algebra for Go](assets/keel-hero.webp)

# keel

keel is a float32 BLAS subset in pure Go. Levels 1 and 2, and a GEMM-centered
Level 3, with AVX-512 kernels written in Go against the experimental
`simd`/`archsimd` packages and a scalar path that builds on an ordinary
toolchain. No cgo, no assembly, no code generation, no background goroutines: it
is a normal Go module that respects `GOMAXPROCS`.

!!! warning "Status as of 2026-08-16: pre-release"

    Levels 1–3 are implemented and gated. Phases P0–P4 are green; P5
    (parallelism, dispatch, polish) is in progress. There is no tagged release
    yet, so the API may still change.

## Install

```
go get github.com/scttfrdmn/keel
```

Go 1.26 or newer. The vector kernels additionally need a toolchain built with
the SIMD experiment enabled, and amd64 hardware with AVX-512:

```
GOEXPERIMENT=simd go build ./...   # vector kernels
go build ./...                     # scalar path, any toolchain
```

Both modes compile the same source and produce the same answers. The vector
kernels are compiled only in the first, so a build without `GOEXPERIMENT=simd`
is correct and slow. Nothing warns you — see
[Troubleshooting](troubleshooting.md) for how to check what you got.

## Quickstart

```go
package main

import (
	"fmt"

	"github.com/scttfrdmn/keel"
)

func main() {
	// C = A·B, A 2×3, B 3×2, C 2×2. Matrices are row-major []float32; the
	// trailing int after each is its leading dimension, the element count from
	// one row to the next.
	a := []float32{
		1, 2, 3,
		4, 5, 6,
	}
	b := []float32{
		7, 8,
		9, 10,
		11, 12,
	}
	c := make([]float32, 2*2)

	keel.Sgemm(keel.NoTrans, keel.NoTrans,
		2, 2, 3, // m, n, k
		1, a, 3, // alpha, A, lda
		b, 2, // B, ldb
		0, c, 2) // beta, C, ldc

	fmt.Println(c) // [58 64 139 154]
}
```

Then read [Usage](usage.md) for the full routine list and the storage
conventions, or [Capabilities & limits](limits.md) for what keel does not do.

## Where to go next

<div class="grid cards" markdown>

-   __[Usage](usage.md)__ — every routine, the row-major convention, the
    environment variables.

-   __[Capabilities & limits](limits.md)__ — float32 only, row-major only,
    amd64 fast paths, and what is not supported at all.

-   __[Numbers](numbers.md)__ — measured rates, each with the CPU it was
    measured on and the denominator it was divided by.

-   __[Troubleshooting](troubleshooting.md)__ — the three things that actually
    go wrong.

</div>
