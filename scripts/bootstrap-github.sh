#!/usr/bin/env bash
# Bootstrap the keel GitHub presence: repo, labels, milestones, umbrella
# issues, and a Projects (v2) board. Requires an authenticated `gh` CLI
# (gh auth status) with project scope: gh auth refresh -s project
#
# Usage: scripts/bootstrap-github.sh [--public]
set -euo pipefail

OWNER="scttfrdmn"
REPO="keel"
VISIBILITY="--private"
[[ "${1:-}" == "--public" ]] && VISIBILITY="--public"

echo "==> repo"
if ! gh repo view "$OWNER/$REPO" >/dev/null 2>&1; then
  gh repo create "$OWNER/$REPO" $VISIBILITY \
    --description "Pure-Go float32 BLAS subset on GOEXPERIMENT=simd" \
    --source . --push
else
  echo "    exists; ensuring remote + push"
  git remote get-url origin >/dev/null 2>&1 || git remote add origin "https://github.com/$OWNER/$REPO.git"
  git push -u origin HEAD
fi

echo "==> labels"
label() { gh label create "$1" --color "$2" --description "$3" -R "$OWNER/$REPO" --force; }
label "phase:P0" "0e8a16" "Toolchain probe & shim"
label "phase:P1" "0e8a16" "Level 1 + test harness"
label "phase:P2" "b60205" "Microkernel + spill audit (go/no-go)"
label "phase:P3" "0e8a16" "Packing + blocking -> SGEMM"
label "phase:P4" "0e8a16" "Level 2 + derived Level 3"
label "phase:P5" "0e8a16" "Parallelism, dispatch, polish"
label "gate" "5319e7" "Gate criteria / gate script work"
label "kind:shim" "1d76db" "internal/vec — the simd shim"
label "kind:kernel" "1d76db" "Microkernels"
label "kind:infra" "c5def5" "Build, CI, tooling, project plumbing"
label "toolchain-report" "fbca04" "GOEXPERIMENT=simd field note; feeds docs/toolchain-notes.md"
label "upstream" "fbca04" "Candidate golang/go issue"
label "perf" "d93f0b" "Performance regression or target"
label "correctness" "d93f0b" "Numerics / oracle mismatch"
label "blocked" "000000" "Needs a human decision to proceed"

echo "==> milestones"
ms() {
  gh api -X POST "repos/$OWNER/$REPO/milestones" -f title="$1" -f description="$2" >/dev/null 2>&1 \
    || echo "    milestone '$1' exists"
}
ms "P0 — Toolchain probe & shim"        "Gate: shim differential tests green on all backends; FMA lowers to VFMADD231PS. DESIGN.md §4/P0."
ms "P1 — Level 1 + harness"             "Gate: L1 green vs oracle incl. edge shapes; Sdot >=4x scalar at n=4096. DESIGN.md §4/P1."
ms "P2 — Microkernel (go/no-go)"        "Gate: 0 accumulator spills in K-loop AND raw kernel >=55% of theoretical peak. On fail: spill-report + STOP. DESIGN.md §4/P2."
ms "P3 — SGEMM"                         "Gate: Sgemm matches oracle across size sweep; single-thread >=60% OpenBLAS at 2048^3. DESIGN.md §4/P3."
ms "P4 — L2 + derived L3"               "Gate: sgemv/sger/syrk/symm/trsm green incl. full flag lattice; Ssyrk >=85% of Sgemm GFLOPS. DESIGN.md §4/P4."
ms "P5 — Parallel + polish"             "Gate: >=6x at 8 cores on 4096^3; race clean; scalar-only stock-toolchain build green; vet/lint clean. DESIGN.md §4/P5."

mnum() { gh api "repos/$OWNER/$REPO/milestones" --jq ".[] | select(.title|startswith(\"$1\")) | .number"; }

echo "==> umbrella issues"
issue() { # title, milestone-prefix, labels, body
  local t="$1" mp="$2" l="$3" b="$4"
  if gh issue list -R "$OWNER/$REPO" --search "in:title \"$t\"" --json title --jq '.[].title' | grep -qF "$t"; then
    echo "    issue '$t' exists"; return
  fi
  gh issue create -R "$OWNER/$REPO" --title "$t" --milestone "$(gh api "repos/$OWNER/$REPO/milestones" --jq ".[] | select(.number==$(mnum "$mp")) | .title")" \
    $(for x in $l; do printf -- "--label %s " "$x"; done) --body "$b" >/dev/null
  echo "    created '$t'"
}

issue "P0 umbrella: toolchain probe & the shim" "P0" "phase:P0 kind:shim gate" "$(cat <<'BODY'
Tracking issue for phase P0 (DESIGN.md §4/P0). Session-end status comments and Scott's course corrections live here.

- [ ] Toolchain installed (1.27rc or newest 1.26.x); `make build` and `make stock` green
- [ ] archsimd/simd API read via `go doc` + GOROOT sources (no recalled identifiers) — note toolchain version here
- [ ] `internal/vec` scalar backend: the ~12 ops SGEMM needs, as the executable spec
- [ ] AVX-512 + AVX2 backends over the read API
- [ ] Differential tests binding every vector op to its scalar twin (NaN, ±Inf, denormals, -0, empty)
- [ ] FMA wrapper disassembles to a single VFMADD231PS (grep `-gcflags=-S`); if no fused op exists in the API — STOP and surface it
- [ ] `scripts/gate-p0.sh` implements the above and exits 0

**Gate:** shim differential tests green on all backends; FMA check passes.
BODY
)"

issue "P1 umbrella: Level 1 + test harness" "P1" "phase:P1 gate" "$(cat <<'BODY'
Tracking issue for phase P1 (DESIGN.md §4/P1).

- [ ] float64 oracle implementations for all six L1 routines
- [ ] `oracle.Tolerance` wired as the only epsilon source
- [ ] Sdot/Saxpy/Sscal/Snrm2/Sasum/Isamax over the shim
- [ ] Property tests: aliasing, zero-length, stride≠1, NaN per BLAS convention
- [ ] Benchmarks in harness with peak/denominator reporting
- [ ] `scripts/gate-p1.sh` green: all L1 vs oracle on avx512+scalar; Sdot ≥4× scalar at n=4096
BODY
)"

issue "P2 umbrella: microkernel + spill audit (GO/NO-GO)" "P2" "phase:P2 kind:kernel gate" "$(cat <<'BODY'
Tracking issue for phase P2 (DESIGN.md §4/P2). **This phase can end the project's current approach; treat the gate as a decision point, not a hurdle.**

- [ ] `internal/spill` audit tool: counts stack-relative vector moves in the steady-state K-loop from `-gcflags=-S`; archives GOSSAFUNC ssa.html per commit
- [ ] Kernel `sgemm_32x6_avx512.go` (12 accumulators, 15 live zmm) — shaping rules: pre-sliced panels (BCE verified), no calls in K-loop, K unrolled ×4
- [ ] Grow tile toward 32×12 until spills appear; back off one step; record frontier in `KERNEL.md`
- [ ] Raw kernel benchmark vs theoretical peak (harness computes peak from CPUID + measured freq)
- [ ] `scripts/gate-p2.sh` green: **0 accumulator spills AND ≥55% of peak**

**On gate failure after shaping + one shrink:** write `docs/spill-report.md` with ssa.html evidence, label this issue `blocked`, comment with the one-line decision needed (file upstream / avo fallback / AVX2 shapes), and STOP.
BODY
)"

issue "P3 umbrella: packing + blocking → full SGEMM" "P3" "phase:P3 gate" "$(cat <<'BODY'
Tracking issue for phase P3 (DESIGN.md §4/P3).

- [ ] Goto blocking NC→KC→MC→NR→MR (initial KC=384 MC=144 NC=4096, as vars)
- [ ] SIMD packing routines incl. transpose absorption
- [ ] Edge handling for M%MR, N%NR (masked ops if the API supports them well, else scalar fringe — decide from the read API and note here)
- [ ] β=0 / β=1 kernel variants (no branch in loop)
- [ ] `scripts/gate-p3.sh` green: oracle sweep (1..17, 63,64,65, 500, 1000, 2048 × flags) AND single-thread ≥60% OpenBLAS at 2048³
BODY
)"

issue "P4 umbrella: Level 2 + derived Level 3" "P4" "phase:P4 gate" "$(cat <<'BODY'
Tracking issue for phase P4 (DESIGN.md §4/P4).

- [ ] Sgemv (both transposes), Sger
- [ ] Ssyrk, Ssymm as blocked GEMM with triangular C-update masking
- [ ] Strsm: unblocked diagonal solves + GEMM rank updates (BLIS recipe, cited in comments)
- [ ] `scripts/gate-p4.sh` green: full flag lattice vs oracle; Ssyrk ≥85% of Sgemm GFLOPS
BODY
)"

issue "P5 umbrella: parallelism, dispatch, polish" "P5" "phase:P5 kind:infra gate" "$(cat <<'BODY'
Tracking issue for phase P5 (DESIGN.md §4/P5).

- [ ] MC-loop parallelism: bounded workers by GOMAXPROCS, shared packed-B per NC, per-worker packed-A via sync.Pool; no background threads
- [ ] Dispatch avx512→avx2→scalar; `KEEL_FORCE` override
- [ ] Package builds and passes scalar-only on stock toolchain (no GOEXPERIMENT)
- [ ] Scaling benchmark; README updated with honest numbers; doc.go
- [ ] CHANGELOG `[Unreleased]` → v0.1.0 prep
- [ ] `scripts/gate-p5.sh` green: ≥6× at 8 cores on 4096³; `-race` clean; vet/lint clean; stock build green
BODY
)"

echo "==> project board"
PROJ_TITLE="keel v0"
if ! gh project list --owner "$OWNER" --format json --jq '.projects[].title' 2>/dev/null | grep -qxF "$PROJ_TITLE"; then
  gh project create --owner "$OWNER" --title "$PROJ_TITLE" >/dev/null && echo "    created project '$PROJ_TITLE'"
else
  echo "    project exists"
fi
PNUM=$(gh project list --owner "$OWNER" --format json --jq ".projects[] | select(.title==\"$PROJ_TITLE\") | .number")
for url in $(gh issue list -R "$OWNER/$REPO" --state open --json url --jq '.[].url'); do
  gh project item-add "$PNUM" --owner "$OWNER" --url "$url" >/dev/null 2>&1 || true
done
echo "    open issues added to project $PNUM"

echo "==> done. https://github.com/$OWNER/$REPO"
