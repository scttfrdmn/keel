# Strsm refresh, judged arm64 run 1 of 2 (#37)

Rev `3745b72`, era `ceil8`, us-east-1 on-demand, 2026-10-12 00:43Z–01:59Z (1h16m,
$10.35/hr, ~$13.1). `keel-gvt3` = c7g.16xlarge (Neoverse-V1), `keel-gvt4` =
c8g.48xlarge (Neoverse-V2) — era ceil8's own types.

**51 PASS / 4 FAIL / 0 UNMEASURED / 0 BASELINE → RED.**

## A VOIDED FIRST ATTEMPT IS ARCHIVED BESIDE THIS ONE
`VOID-wrong-goarch-3745b72.log` is a 32-minute run that cost ~$5.5 and measured
nothing. `scripts/remote.sh` builds with `GOARCH="${KEEL_GOARCH:-amd64}"`, so an
arm64 fleet with `KEEL_GOARCH` unset gets **amd64 binaries shipped to Graviton**.
The kernel refuses the exec and the shell interprets the ELF:
`./keel.test: 1: Syntax error: ")" unexpected`. That failed the suite on both hosts
and left 11 criteria UNMEASURED downstream. The fleet layer was entirely correct —
right instance types, right hosts, right uarch labels — which is the trap: *the
fleet layer being arch-aware is not the gate being arch-aware*. Kept rather than
deleted because a voided run is evidence about the apparatus.

## The prediction was CONFIRMED, and its value is that it predicted a SMALL number

Pre-registered before launch: +2% to +8%, REFUTED above +20%, and criterion 9 might
legitimately pass since a sub-5% move sits inside `README_TOL`.

| host | row | published | this run | delta |
|---|---|---|---|---|
| Neoverse-V2 | Strsm/threads=8 | 230.1 | 242.85 | **+5.5%** |
| Neoverse-V1 | Strsm/threads=8 | 206.8 | 217.9 | **+5.4%** |
| both | Strsm/threads=1 | — | — | **passed criterion 9** (inside the band) |

So the same code change is **+58% to +115% on amd64 and ~+5% on arm64**, and the
model predicted both ends from one quantity: the diagonal solve is ~65% of Strsm's
time at 16 lanes and 7.2% at 4 lanes. A model that only predicted the large number
would be far weaker evidence than one that also predicted the small one.

## An UNPREDICTED failure: 14 of 48 judged L1 rows regressed

Not Strsm, and not predicted anywhere. Shortfalls run **−0.0045% to −2.78%**, with
3 of 14 under 0.1% — below any plausible measurement resolution. The routines are
Snrm2 and Isamax (8 of 14), plus Sdot, Sasum and Saxpy.

**Why it is visible here and not on amd64:** arm64's L1 rows are *registered* and
therefore judged, while the amd64 hosts' L1 rows were deliberately held back (#179),
so they render BASELINE there. The same movement could exist on amd64 and be
unjudged. L1 was GREEN on arm64 at the last judged run (`5e769f7`), so something
did change.

**Two candidate causes, not separated by this run.** (a) My l1 change added a
`Kernels` field and `neonScalDiv` to `l1_arm64.go`, which shifts function addresses
— the third alignment finding of the day (#61; the arm64 Trsm A/B moved untouched
code 8.4% through a `block.gemm.func1` 16→32 mod 64 shift). (b) These are the exact
rows #179 filed as too irreproducible to be references, and #60 records Sdot moving
4–9% in different directions per host. A shortfall of 0.0045% cannot be produced by
any deliberate mechanism. **Run 2 at this same rev is the discriminator**: the same
14 rows tripping means systematic, a different set means noise.

No baseline was re-registered. Lowering a baseline to turn a red green is a
loosening and needs a decision, not a commit.

## A SECOND voided run is archived here too, and it was my fault directly

`VOID-dirty-tree-3745b72.log` is arm64 run 2, refused after 7.5 minutes (~$1.3) with
`!! working tree is dirty; git archive HEAD would ship something other than what is
here`. Its preflight saw a clean tree at 02:01:03Z. **I then archived run 1 — these
very files — at 02:02:45Z, inside the run's window**, and the gate's own re-check
caught it at 02:08:39Z and refused.

`CLAUDE.md`: *"The tree stays frozen for a run's whole life, detached or not."* The
gate defended itself correctly; the defect was mine. The failure mode worth naming
is that I did not classify *archiving a finished run* as an edit — it creates
untracked files, which is exactly what `git status --porcelain` reports and what
`git archive HEAD` would omit.

**Campaign tally, stated because three botched launches is the signal to stop:**
the brace-in-expansion bug (caught pre-launch, $0), `KEEL_GOARCH` unset (~$5.5), and
this (~$1.3). ~$6.8 of ~$101 bought nothing. arm64 therefore sits at N=1 and the
README regeneration is blocked on one more arm64 run.
