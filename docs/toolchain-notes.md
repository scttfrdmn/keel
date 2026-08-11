# Toolchain field notes — GOEXPERIMENT=simd

A first-class deliverable (DESIGN.md §7 rule 8): every surprise from the
experimental simd packages — missing intrinsics, unexpected lowerings,
allocator behavior, API changes between releases — gets an entry here with
a minimal repro *before* any workaround lands. Entries feed upstream issues.

| Date | Toolchain | Observation | Repro | Upstream issue |
|---|---|---|---|---|
