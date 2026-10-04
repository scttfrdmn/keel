#136 judged-tier evidence, us-east-1, 2026-10-04, rev d757ea9. BenchmarkKernel kc=128, n=10.

host file                      3x24     4x16      8x8 3x24 gain
bench-...-1.txt               38.48    34.91    26.94   +10.23%
bench-...-2.txt               35.56    32.65    24.70    +8.91%

Intervals are essentially zero-width (3x24 38.47-38.49 / 35.55-35.57; 4x16 34.90-34.91 / 32.64-32.66),
so both comparisons are CI-disjoint by orders of magnitude.

Bracket from the GB10 characterization (#136): X925 +7.40%, A725 +20.78%.
The judged Neoverse readings land at the LOW end, as predicted for 4-pipe-class FP.

NOTE: the two files cannot be attributed to gvt3/gvt4 from their own markers --
keel-bench-cpu reads "unknown (no model name in /proc/cpuinfo)" on arm64 and
keel-bench-cores reads "8 logical" on both because the placement mask is applied.
Both parts are reported; which is V1 and which is V2 is NOT claimed here.
