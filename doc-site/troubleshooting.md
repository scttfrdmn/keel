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

**Cause.** keel requires **Go 1.26 or newer**, and the *vector* path requires
**1.27 or newer** (see the `archsimd` name section below). The `simd` and
`archsimd` packages do not exist before 1.26 at all.

**Fix.** Upgrade the toolchain. If that is not possible, nothing in keel works on
an older one — there is no build tag that trades the vector path for an older Go,
because the module's minimum is set by the language version, not by the kernels.
Note that `go.mod` states the 1.26 floor and *cannot* state the vector path's 1.27
one: `archsimd` ships with the toolchain, so it is not a module requirement.

## `-race` dies with a `checkptr` fatal error

**Symptom.** Under `-race`, or under `-gcflags=all=-d=checkptr`, the process
aborts:

```
fatal error: checkptr: converted pointer straddles multiple allocations
	simd/archsimd.paFloat32x16(...)
	simd/archsimd.LoadFloat32x16Part(...)
```

**Cause.** Not keel. `archsimd`'s partial slice load and store reach a masked
full-width operation by converting `&s[0]` to a pointer to a full-width array.
The mask keeps the *instruction* in bounds, but `checkptr` instruments the
*conversion* and cannot know that. It is
[golang/go#80856](https://github.com/golang/go/issues/80856).

**Upstream status, as of 2026-08-28, and it splits by how you asked.** Go 1.27
annotates the `pa*` helpers `//go:nocheckptr`
([CL 761120](https://go-review.googlesource.com/c/go/+/761120), merged; it closed
[golang/go#78413](https://github.com/golang/go/issues/78413), of which golang/go#80856
is a duplicate). That is expected to settle `-gcflags=all=-d=checkptr`. It is **not**
expected to settle `-race`, because
[golang/go#42880](https://github.com/golang/go/issues/42880) — open — records that
`-race` does not obey `go:nocheckptr`. One annotation, two consumers, and keel has
not yet run either on an AVX-512 host under 1.27, so treat both as unconfirmed here.

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

## The build fails on `archsimd` load and store names

**Symptom.** On **Go 1.26**, `GOEXPERIMENT=simd go build ./...` reports type
errors throughout `internal/vec`, all of them about `archsimd` load and store —
either `undefined: archsimd.LoadFloat32x16` used with a slice, or

```
cannot use bp[0:16] (value of type []float32) as *[16]float32 value
	in argument to archsimd.LoadFloat32x16
```

**Cause.** `archsimd` swapped its load and store names between Go 1.26 and 1.27:
the slice forms took over the bare names and the array forms gained an `Array`
suffix, and two partial operations additionally changed their return types. keel
is written against the **1.27** names. This is the experimental-package risk being
ordinary rather than theoretical.

**Fix, as of 2026-08-28.** Build the vector path with Go 1.27 or newer
([issue #69](https://github.com/scttfrdmn/keel/issues/69), landed). Because the
change is a *swap* rather than a set of renames, there is no source that satisfies
both toolchains and no build tag that would trade one for the other, so 1.27 is a
floor. The scalar path is unaffected and still builds on 1.26.

Note the direction: before 2026-08-28 this section said the opposite, and the fix
was to *downgrade* to 1.26. If a future release swaps them back, expect it to
invert again — `archsimd` carries no compatibility promise.

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
