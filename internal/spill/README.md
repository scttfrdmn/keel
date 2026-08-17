Spill-audit tooling (built at start of P2):

- `GOEXPERIMENT=simd go run ./internal/spill/cmd/spill-audit -pkg ./internal/vec
  -func Kernel2x32,Kernel4x32 -mode spill` runs `go build -gcflags=-S`, isolates
  the steady-state K-loop, and counts vector loads/stores to stack-relative
  addresses. Gate P2 requires zero accumulator spills.

  Corrected 2026-08-16: this line documented `cmd/spillaudit -pkg ./internal/kern
  -fn Kernel32x6`, and **not one of those four tokens was real** — the command is
  `spill-audit`, the flag is `-func`, the kernels live in `./internal/vec`, and
  `Kernel32x6` never existed under that name (the tile is `MR`×`NR` = rows ×
  columns, so it is `Kernel6x32`, which spills and does not ship). The invocation
  gate-p2 actually runs is at `scripts/gate-p2.sh:448`; `KERN_PKG` and
  `KERN_FUNCS` at lines 180–181 are the single statement of what is audited.
- The same run archives `GOSSAFUNC` output (`ssa.html`) so a failed audit
  ships with evidence of *why* the allocator spilled.
