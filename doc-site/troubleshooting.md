# Troubleshooting

## keel is much slower than expected

**Symptom.** Rates far below what the [numbers](numbers.md) page shows, on
hardware that should reach them. Nothing failed and nothing warned.

**Cause.** The build did not have `GOEXPERIMENT=simd`, so the vector kernels
were never compiled. keel falls back to the scalar path, which is correct and
several times slower. This is by far the most common surprise, because there is
no error to notice.

**Check.** Ask the library what it dispatched to:

```go
fmt.Println(keel.ActiveL1Backend(), keel.ActiveKernBackend())
```

`scalar scalar` means the vector path is not in this binary, or this CPU has no
AVX-512. `keel.AvailableKernels()` separates those two: it is empty of `avx512`
entries when the hardware lacks the feature, and reports them when the hardware
has it.

**Fix.** Set the experiment at build time — it is a build flag, not a run-time
one, so setting it only when you run the binary changes nothing:

```
GOEXPERIMENT=simd go build ./...
GOEXPERIMENT=simd go test ./...
```

## The build fails on a Go version

**Symptom.** `go` reports that `go.mod` requires a newer Go than the one running,
or the build cannot find `simd/archsimd`.

**Cause.** keel requires **Go 1.26 or newer**. The `simd` and `archsimd`
packages do not exist before it.

**Fix.** Upgrade the toolchain. If that is not possible, nothing in keel works on
an older one — there is no build tag that trades the vector path for an older Go,
because the module's minimum is set by the language version, not by the kernels.

## `-race` dies with a `checkptr` fatal error

**Symptom.** Under `-race`, or under `-gcflags=all=-d=checkptr`, the process
aborts:

```
fatal error: checkptr: converted pointer straddles multiple allocations
	simd/archsimd.paFloat32x16(...)
	simd/archsimd.LoadFloat32x16SlicePart(...)
```

**Cause.** Not keel. `archsimd`'s partial slice load and store reach a masked
full-width operation by converting `&s[0]` to a pointer to a full-width array.
The mask keeps the *instruction* in bounds, but `checkptr` instruments the
*conversion* and cannot know that. It is
[golang/go#80856](https://github.com/golang/go/issues/80856), fixed upstream in
Go 1.27.

Three properties make it worse than an ordinary instrumentation complaint: it is
**fatal rather than reported**, so it destroys any race-detector run it precedes;
`checkptr` is the trigger and not `-race`, so no race-detector option dodges it;
and it fires on how much room the *allocation* has past `&s[0]`, not on the slice
length — so a call site can be quiet for a whole suite and abort after an
allocator layout change.

**Fix.** Either run the race detector against the scalar backend, which never
takes a partial vector operation:

```
KEEL_FORCE=scalar GOEXPERIMENT=simd go test -race ./...
```

or use a toolchain that carries the upstream fix.

## The build fails with `undefined: archsimd.LoadFloat32x16Slice`

**Symptom.** On Go 1.27, `GOEXPERIMENT=simd go build ./...` reports type errors
in `internal/vec`, all of them about `archsimd` load and store names.

**Cause.** `archsimd` swapped its load and store names between Go 1.26 and 1.27:
the slice forms took over the bare names and the array forms gained an `Array`
suffix, and two partial operations additionally changed their return types. keel
is written against the 1.26 names. This is the experimental-package risk being
ordinary rather than theoretical.

**Fix, as of 2026-08-16.** Build the vector path with Go 1.26.x. The port is
[issue #69](https://github.com/scttfrdmn/keel/issues/69) and is held until
`go1.27.0` final. The scalar path builds on either.

## `panic: keel: KEEL_FORCE=avx512 is not available on this machine`

**Symptom.** A panic at process start, naming the backends the machine does have.

**Cause.** `KEEL_FORCE` names a backend this build or this CPU cannot run. keel
panics rather than downgrading, deliberately: a run that believed it was
measuring AVX-512 and quietly measured scalar is worse than a crash.

**Fix.** Unset `KEEL_FORCE` to auto-select, or set it to a backend
`keel.AvailableL1Backends()` reports. Note that `KEEL_FORCE=avx2` is *not* an
error — it gives an AVX2 Level 1 and a scalar Level 3, because there is no AVX2
microkernel, and `keel.ActiveKernBackend()` reports `scalar` so that no
measurement can mistake it for one.
