# Strsm refresh, judged arm64 run 2 of 2 (#37)

Rev `5948dda`, era `ceil8`, us-east-1 on-demand, 2026-10-12 17:30Z–18:48Z (1h18m,
$10.35/hr, ~$13.5). Completes N=2 on arm64 and therefore the whole refresh.

**51 PASS / 4 FAIL / 0 UNMEASURED / 0 BASELINE → RED**, the same shape as run 1.
The OpenBLAS clone retry added in `5948dda` was not needed (0 retries used); it is
insurance against the transient that killed the previous attempt.

## Strsm: arm64's pair is tight

| host | row | published | run 1 | run 2 | between-run drift |
|---|---|---|---|---|---|
| Neoverse-V1 | Strsm/threads=8 | 206.80 | 217.90 | 217.80 | **−0.05%** |
| Neoverse-V2 | Strsm/threads=8 | 230.10 | 242.85 | 243.40 | **+0.23%** |
| both | Strsm/threads=1 | — | passed | passed | — |

Both runs flagged exactly the same two rows, and the readings reproduce to 0.25%.
For contrast the amd64 pair drifted up to **7.83%** (zen5/threads=8, with its two
thread arms moving in opposite directions), so Graviton is markedly more
reproducible here than the c5n/c7a/c8a fleet — worth recording because it bears on
how many runs each architecture actually needs.

## The pre-registered L1 discriminator: INCONCLUSIVE

The measured test binary is byte-identical at run 1's rev and run 2's
(`519fca33…`), so this was a same-code repeat on fresh instances. Pre-registered
bands were **≥11 of 14 systematic / ≤6 noise / 7–10 inconclusive**, against a chance
expectation of 4.1.

**Overlap = 9. By its own stated bands the test is INCONCLUSIVE, and that is the
verdict.**

| host | row | run 1 | run 2 |
|---|---|---|---|
| keel-gvt3 | Snrm2/n=256 | −2.782% | −2.832% |
| keel-gvt3 | Snrm2/n=4096 | −0.506% | −0.596% |
| keel-gvt3 | Snrm2/n=65536 | −0.638% | −0.638% |
| keel-gvt3 | Isamax/n=256 | −0.347% | −0.342% |
| keel-gvt3 | Isamax/n=65536 | −0.535% | −0.594% |
| keel-gvt4 | Snrm2/n=256 | −1.042% | −1.340% |
| keel-gvt4 | Sasum/n=256 | −0.383% | −0.478% |
| keel-gvt4 | Isamax/n=256 | −0.445% | −0.424% |
| keel-gvt4 | Isamax/n=4096 | −0.004% | −0.014% |

**Two observations that were NOT pre-registered and therefore do not change the
verdict.** Stated as observations precisely because promoting them now would be
amending a criterion after seeing its result:

1. **Run 2 tripped zero rows that run 1 did not.** Its 9 are a strict subset of run
   1's 14. Under noise, new trips in run 2 would be expected.
2. **The repeated rows reproduce closely** — `Snrm2/n=65536` is −0.638% in both, and
   the rest agree within ~0.3pp. The 5 non-repeating rows were all marginal in run 1
   (−0.004% to −0.718%), i.e. sitting on the trip threshold.

Together those point systematic, but the test I committed to says inconclusive, so
inconclusive is what this records.

**What would settle it:** a run at the pre-fix rev on these same two hosts — a true
L1 A/B rather than a repeat — or landing #179 so the criterion has a state for rows
it already judges to be unusable as references. No baseline was re-registered.

`VOID-openblas-clone-eof-6f53c4a.log` is the preceding attempt, killed in 5.5
minutes (~$1) by `fatal: early EOF` on the OpenBLAS clone. Kept as apparatus
evidence; it is what the retry in `5948dda` answers.
