# Usage

## Storage: row-major, with a leading dimension

Every matrix argument is one `[]float32` in row-major order, followed by its
**leading dimension**: the element count from the start of one row to the start
of the next. For an m×n matrix `A` with leading dimension `lda`:

```go
A[i][j] == a[i*lda+j]
```

`lda` may be larger than `n`. That is how you pass a submatrix without copying
it — here a 2×3 matrix inside a 5-wide array:

```text
      col 0   1   2   3   4        m = 2, n = 3, lda = 5
row 0   [ 1   2   3   ·   · ]      A[1][2] is a[1*5+2], which is 6
row 1   [ 4   5   6   ·   · ]      the · elements are never read
```

Reference BLAS is column-major. keel is not a translation of it, so there is no
`order` argument to pass and no `order` argument to get wrong.

A leading dimension always describes the array **as stored**, before any
transpose flag is applied. It must be at least the number of columns of the
stored array, and the slice must reach the last element of the last row —
`(rows-1)*ld + cols` elements. Padding past that is allowed and never read.

## Storage: vectors take a stride

Vector arguments take a count `n` and a stride `incX`. A stride of 1 is
contiguous; 2 reads every other element; a negative stride walks backwards from
the end.

`n` is the number of elements **touched**, never the length of the slice. A slice
must be long enough for `(n-1)*|incX| + 1` elements.

`Sdot`, `Saxpy`, `Sgemv` and `Sger` accept a negative stride. `Sscal`, `Sasum`,
`Snrm2` and `Isamax` require `incX > 0`, which is how reference BLAS defines
them.

## Routines

### Level 1 — vectors

```go
func Sdot(n int, x []float32, incX int, y []float32, incY int) float32
func Saxpy(n int, alpha float32, x []float32, incX int, y []float32, incY int)
func Sscal(n int, alpha float32, x []float32, incX int)
func Sasum(n int, x []float32, incX int) float32
func Snrm2(n int, x []float32, incX int) float32
func Isamax(n int, x []float32, incX int) int
```

`Sdot` returns xᵀy. `Saxpy` computes `y += alpha*x`. `Sscal` computes
`x *= alpha`. `Sasum` returns Σ|xᵢ|, `Snrm2` the Euclidean norm.

`Isamax` returns the index of the first element of greatest magnitude, and it is
**0-based** — Fortran `ISAMAX` returns a 1-based index. It returns -1 for an
empty vector, where Fortran returns 0.

### Level 2 — matrix–vector

```go
func Sgemv(tA Transpose, m, n int, alpha float32, a []float32, lda int, x []float32, incX int, beta float32, y []float32, incY int)
func Sger(m, n int, alpha float32, x []float32, incX int, y []float32, incY int, a []float32, lda int)
```

`Sgemv` computes `y = alpha·op(A)·x + beta·y`; `Sger` computes
`A += alpha·x·yᵀ`.

`m` and `n` are always the dimensions of `A` **as stored**. Under `Trans` the
vector lengths swap: `x` is `m` long and `y` is `n` long.

### Level 3 — matrix–matrix

```go
func Sgemm(tA, tB Transpose, m, n, k int, alpha float32, a []float32, lda int, b []float32, ldb int, beta float32, c []float32, ldc int)
func Ssyrk(ul Uplo, t Transpose, n, k int, alpha float32, a []float32, lda int, beta float32, c []float32, ldc int)
func Ssymm(s Side, ul Uplo, m, n int, alpha float32, a []float32, lda int, b []float32, ldb int, beta float32, c []float32, ldc int)
func Strsm(s Side, ul Uplo, tA Transpose, d Diag, m, n int, alpha float32, a []float32, lda int, b []float32, ldb int)
```

| routine | computes |
| --- | --- |
| `Sgemm` | `C = alpha·op(A)·op(B) + beta·C`, with `op(A)` m×k, `op(B)` k×n, `C` m×n |
| `Ssyrk` | `C = alpha·A·Aᵀ + beta·C`, or `alpha·Aᵀ·A + beta·C` under `Trans`; only the `ul` triangle of the symmetric `C` is read or written |
| `Ssymm` | `C = alpha·A·B + beta·C` (`Left`) or `C = alpha·B·A + beta·C` (`Right`), `A` symmetric in the `ul` triangle, `C` general |
| `Strsm` | solves `op(A)·X = alpha·B` (`Left`) or `X·op(A) = alpha·B` (`Right`) for `X`, `A` triangular, **overwriting `B` with `X`** |

`Strsm` does not check that `A` is nonsingular, and neither does reference BLAS.
A zero on the referenced diagonal produces `Inf` or `NaN` in `B`.

## Flags

Each flag holds the same letter as reference BLAS, so a call ported from Fortran
or CBLAS reads across unchanged.

| type | values | meaning |
| --- | --- | --- |
| `Transpose` | `NoTrans`, `Trans` | read the matrix as stored, or transposed |
| `Uplo` | `Upper`, `Lower` | which triangle holds the data; the diagonal belongs to both |
| `Side` | `Left`, `Right` | which side of the product the special matrix is on |
| `Diag` | `NonUnit`, `Unit` | read the stored diagonal, or take it to be 1 |

There is no `ConjTrans`: keel is real-valued.

Two flags are stronger promises than they look:

- **`Uplo` — the other triangle is never read and never written.** Not zeroed,
  not mirrored. It may hold a second matrix packed into the same array.
- **`Unit` — the diagonal is not referenced at all.** This is stronger than "the
  diagonal contains ones", so a caller holding an LU factorization in one array,
  where L's unit diagonal is overwritten by U's, may pass it as-is.

## Invalid arguments panic

A negative dimension, a zero stride, an unrecognized flag, a leading dimension
narrower than the row it describes, or a slice too short for the shape it was
given **panics at the call**. Reference BLAS returns silently from several of
these; in Go there is no `XERBLA` to consult afterwards, so keel reports the
problem where it happened.

`n == 0` is not an error. It is the empty vector or the empty matrix, the call
does nothing, and the slices may be `nil`. A matrix whose declared shape is not
empty must still be long enough for that shape, even on a call that will not read
it.

Where reference BLAS skips work for a zero multiplier, keel skips exactly the
same work — `alpha == 0` does not read the operands, `beta == 0` does not read
the destination — because those rules are observable: `0·Inf` is `NaN`, and a
`beta == 0` destination is allowed to be uninitialized.

## Parallelism: `GOMAXPROCS`, and nothing else

The Level-3 routines spread their work over goroutines sized by
`runtime.GOMAXPROCS(0)`, started per call and joined before the call returns.

- At `GOMAXPROCS=1` the work runs in the calling goroutine, with no goroutine and
  no atomic in the path.
- No goroutine outlives a call. Repeated calls leak nothing, and keel is safe to
  call from code that counts goroutines.
- **The result is bit-identical at every `GOMAXPROCS`** — exactly, not within a
  tolerance. The parallel axis splits the output, never a single output element's
  sum, so the answer does not move with the core count.

Level 1 is not parallelized: those routines are memory-bound at every size where
a thread would pay for itself. Reach Level-1 parallelism by calling from parallel
code.

## What the scalar fallback means

There is one source tree and two build modes. Both compute the same results; only
the speed differs.

| | vector path | scalar path |
| --- | --- | --- |
| built by | `GOEXPERIMENT=simd go build` on amd64 | any Go build |
| needs | AVX-512 at run time for Level 3 | nothing |
| Level 1 | AVX-512, else AVX2 | plain Go loops |
| Level 3 | AVX-512 microkernel | plain Go microkernel |

The scalar path is not a stub. It is the reference the vector kernels are tested
against, element by element, so it is the definition of a correct answer here
rather than a degraded one. It is several times slower.

Results are **not** bit-identical between the two paths, or to a textbook triple
loop: blocked accumulation and vector reduction add in a different order. They
are bit-stable within one backend.

## Environment variables

| variable | values | effect |
| --- | --- | --- |
| `GOEXPERIMENT=simd` | — | build-time. Compiles the vector kernels at all. |
| `KEEL_FORCE` | `scalar`, `avx2`, `avx512` | pins the backend at process start |
| `KEEL_KERN_CLASS` | `fma`, `issue` | pins the host classification the microkernel shape is chosen from |

`KEEL_FORCE` is a testing knob — it is how a machine with AVX-512 runs a suite
scalar-only. Naming a backend the machine cannot run **panics at start** rather
than quietly downgrading, because a run that believed it measured AVX-512 and
quietly measured scalar is worse than a crash.

There is no AVX2 microkernel, so `KEEL_FORCE=avx2` gives an AVX2 Level 1 and a
scalar Level 3, and `keel.ActiveKernBackend()` reports `scalar` so no measurement
can believe otherwise.

## Asking what you got

Every dispatch decision is readable at run time:

```go
fmt.Println(keel.ActiveL1Backend())     // avx512 | avx2 | scalar
fmt.Println(keel.ActiveKernBackend())   // avx512 | scalar
fmt.Println(keel.ActiveKernTile())      // e.g. 4x32 — the microkernel shape
fmt.Println(keel.AvailableL1Backends()) // what this machine can run
fmt.Println(keel.AvailableKernels())
fmt.Println(keel.Workers(1024))         // goroutines a Level-3 call would use
```

The full API, with a runnable example on each routine, is in the
[package documentation](https://pkg.go.dev/github.com/scttfrdmn/keel).
