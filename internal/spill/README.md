Spill-audit tooling (built at start of P2):

- `go run ./internal/spill/cmd/spillaudit -pkg ./internal/kern -fn Kernel32x6`
  runs `go build -gcflags=-S`, isolates the steady-state K-loop, and counts
  vector loads/stores to stack-relative addresses. Gate P2 requires zero
  accumulator spills.
- The same run archives `GOSSAFUNC` output (`ssa.html`) so a failed audit
  ships with evidence of *why* the allocator spilled.
