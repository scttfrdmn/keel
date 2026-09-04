<!-- Copyright 2026 Scott Friedman -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# `#150` freq150: the in-arm clock sampler, and the elevation that did not recur

`#150` recorded a 13–16% between-arm level term on janus that no treatment explained, and named a
clock change as the leading hypothesis. That hypothesis was untestable because `freq_khz` is
sampled only *between* arms (`#81`), so every frequency reading in the record is idle. freq150
built the instrument the attribution needs — a sampler that reads cpu0's clock *while* an arm
runs — and ran test 3's ten arms underneath it, plus a four-arm control that measures the
sampler's own cost.

**Two things came back. The instrument is validated: its own perturbation is 0.4%, ~30× below the
term it hunts. And the 13–16% elevation did not recur — all ten arms landed within ±0.6% of the
floor in GFLOP/s — so the frequency comparison has no subject and every registered cell is
`UNMEASURED`, not "clock refuted."** This is the outcome the registration reserved as OOD item 4:
a second independent observation that the phenomenon does not reproduce under an identical
sequence, which is itself a finding.

The registration (`archive/freq150/predictions-freq150.py`) was committed before the run;
the analyzer (`archive/freq150/analyze-freq150.py`) imports it and restates none of it.

## 1. Provenance

| | |
|---|---|
| rev | `71a73da`, tree clean at launch and throughout |
| host | `janus.local`, i9-9960X (Skylake-X, 16c/32t), `intel_pstate` `governor=performance`, up 42 days, no interactive users |
| driver host | darwin/arm64, detached under tmux as `keel-freq150-71a73da`; survived a Claude Code restart mid-run |
| binary | `sha256=d0d46d26c15cc8b2`, identical to the one `#150`'s elevation was measured on |
| toolchain | `go1.27.0-X:simd`, read off the artifact |
| arms | 10 hunt (`aref ac0 agc apre aboth` then reversed), sampler on; 4 control on `pre` (`k1 k2 k3 k4` = off/on/on/off) |
| samples | `SAMPLE_PERIOD=0.2s`; 4191–5057 in-arm frequency samples per arm, 0 NA |

## 2. The sampler is smaller than the term it hunts (`sampler_ok`: PASS)

Scott's condition on this run was that the sampler's own perturbation be *measured*, not assumed —
an in-arm sampler is a co-tenant by construction, and an instrument must not become the noun it
measures. The palindrome control (`off/on/on/off`, so drift straddles the contrast) measured it:

| row | off GFLOP/s | on GFLOP/s | \|Δ\|/off | verdict |
|---|---|---|---|---|
| 2x32 kc=8  | 67.70 | 67.42 | 0.41% | ok |
| 4x32 kc=8  | 61.23 | 60.96 | 0.44% | ok |
| 2x32 kc=32 | 91.10 | 90.77 | 0.37% | ok |
| 4x32 kc=32 | 72.98 | 72.62 | 0.49% | ok |

All four rows are an order of magnitude under the `SAMPLER_PERTURBATION_MAX = 4.2%` bound (fixed
before the run at one third of the smallest elevation hunted) and ~30× under the 12.6% term itself.
The instrument is admissible, and is now available for any future clock question on this host.

## 3. The elevation did not recur (`elevation_present`: FAIL)

Per-arm median GFLOP/s as a ratio to the floor (median-of-arm-medians per row):

| arm | 2x32/kc8 | 4x32/kc8 | 2x32/kc32 | 4x32/kc32 | elevated? |
|---|---|---|---|---|---|
| aref  | 1.004 | 1.004 | 1.004 | 1.004 | no |
| ac0   | 1.000 | 1.000 | 1.000 | 1.000 | no |
| agc   | 1.000 | 1.000 | 0.998 | 1.000 | no |
| apre  | 1.000 | 1.002 | 1.000 | 1.002 | no |
| aboth | 1.000 | 1.000 | 1.001 | 1.000 | no |
| bboth | 0.999 | 0.999 | 0.999 | 0.999 | no |
| bpre  | 1.000 | 0.999 | 1.000 | 0.994 | no |
| bgc   | 0.999 | 0.999 | 0.999 | 0.998 | no |
| bc0   | 1.001 | 0.998 | 1.000 | 0.998 | no |
| bref  | 1.003 | 1.003 | 1.003 | 1.004 | no |

No arm reached the `ELEVATION_FACTOR = 1.08` witness on any registered row; the widest excursion is
0.6%. The 13–16% term that appeared at run positions 1 and 4 of ten in `#150` was simply not
present this time. Registered consequence: the frequency comparison has no subject, so every
prediction cell is **`UNMEASURED`** — deliberately *not* "H_clock is refuted." Those two readings
are output-indistinguishable in a null and opposite in meaning (rule 26, applied at the subject
rather than the treatment). **The clock hypothesis for `#150` is untouched by this run.**

## 4. Reported, never scored: the clock stepped, the throughput did not

The one substantive thing the samples show, stated *outside* the scored frame because there is no
elevated subject and so it is evidence for nothing about `#150` (reported per rule 12, so a null on
the registered question is not silent about a visible feature of the data):

- `apre`, `bboth`, `bpre` spent ≥30% of their samples above the 3.92 GHz classification boundary
  (`floor_khz` 3.70 GHz × 1.06); `ac0`, `agc`, `aboth`, `bgc`, `bc0` stayed below it entirely.
- Yet GFLOP/s is flat to ±0.6% across all ten arms (§3).

So on this run the clock varied by ~0.3 GHz between arms with **no** measurable throughput
consequence. This is a fact about the samples, not a verdict — but it is the kind of fact that, if
it held under a recurrence, would argue the 13–16% term is not a pure clock term. It is logged
here for whoever measures `#150` next.

**Scope note.** `aref`/`bref` show a median clock of 1.20 GHz because the `ref` config spans two
cores (`mask 0,1`) and the sampler reads cpu0 only, which the runtime need not keep busy. For the
two-core arm, cpu0 is not a reliable witness of the busy clock; the eight single-core arms
(`mask 0`) are.

## 5. What this run does not address

Carried verbatim from the registration, not re-litigated:

1. **Row-level attribution.** The trace is timestamped but the benchmark log carries no per-row
   wall-clock, so every criterion here is arm-level. `#150` item 3 (aref's row-selective
   elevation) is explicitly not addressed, and a null on it is not evidence.
2. **Why a step occurs.** AVX-512 license transitions are the leading candidate on this silicon
   and are not tested; the sampler sees frequency, not instruction mix.
3. **Control-phase generality.** Perturbation is measured on the `pre` config only.
4. **Whether the elevation recurs at all.** It appeared once, in 2 of 10 arms. This run is the
   second observation, and it did not reproduce.
