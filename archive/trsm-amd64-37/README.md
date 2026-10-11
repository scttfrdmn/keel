# #37's amd64 verification — the diagonal-solve fix at 16 lanes

**Dev tier, bare metal, not a published number.** janus.local (i9-9960X Skylake-X,
`governor=performance`, load1=0.04, virt=metal) is the same microarchitecture as the
AWS `keel-skx` host whose published README row reads Strsm/Sgemm = 0.358. The
README's figures come from virtualized AWS, so refreshing them needs a judged run;
this answers the mechanism question the judged run would otherwise be spent to ask.

`prereg.md` was written before any arm ran and fixes the thresholds, the void
condition and an explicit REFUTED band.

## Result: CONFIRMED, at the centre of the predicted range

All six controls landed inside janus's 0.99% between-binary layout floor, so the
run is valid rather than voided, and the harness's planted +10.00% delta was
recovered.

| row | base `79d629a` | new `d62848a` | delta |
|---|---|---|---|
| `Strsm/n=4096/threads=1` | 2.544 s | 1.545 s | **−39.26%** (p=0.000, n=10) |
| `Strsm/n=4096/threads=8` | 369.2 ms | 224.4 ms | **−39.23%** (p=0.000, n=10) |
| `Sgemm/threads=1` (control) | 1.851 s | 1.841 s | ~ (p=0.529) |
| `Sgemm/threads=8` (control) | 279.5 ms | 277.8 ms | −0.59% |
| `Ssymm/threads=1` (control) | 1.893 s | 1.891 s | ~ (p=0.315) |
| `Ssymm/threads=8` (control) | 281.4 ms | 281.9 ms | +0.20% |
| `Ssyrk/threads=1` (control) | 994.8 ms | 991.8 ms | −0.30% |
| `Ssyrk/threads=8` (control) | 139.3 ms | 138.9 ms | ~ (p=0.190) |

Pre-registered threshold was >15% for CONFIRMED against a model range of −30% to
−50%. Measured −39.26%.

## What it establishes about the mechanism
Strsm's single-thread GFLOP/s goes **27.02 → 44.48 (+64.6%)**, so the
Strsm/Sgemm efficiency ratio moves **0.364 → 0.596**. The 0.364 sits exactly in the
published AVX-512 cluster (0.336, 0.356, 0.358) and 0.596 is about halfway to the
NEON cluster (0.804, 0.813). **The vector-width split in Strsm's efficiency was the
scalar diagonal solve**, which is what the cross-host model claimed and what this
measures. Solving 1−f+f/s = 0.6074 puts the diagonal solve at ~65% of Strsm's time
beforehand if it sped up 2.5×, which is the predicted share.

## One prediction that was wrong, and it refines the model
`prereg.md` expected the 8-thread row to improve *less*, on the reasoning that the
diagonal solve is a serial dependency that Amdahl would blunt. Both widths improved
by the **same 39.2%**, so at this shape the diagonal solve parallelizes about as
well as the rank update and was not acting as the serial term. Recorded because it
is the part of the stated model the data contradicts.
