# The first green L1 result, amd64 (#179/#180 confirmation)

Rev `dc519af`, same rev as the arm64 run, with nothing committed between them. us-east-1
on-demand, 2026-10-21 15:47Z–17:28Z (1h41m, ~$41). `keel-skx` c5n.18xlarge, `keel-zen4`
c7a.48xlarge, `keel-zen5` c8a.48xlarge.

**73 PASS / 0 FAIL / 0 UNMEASURED / 0 BASELINE / 5 REPORTED → GREEN.**

- **L1 holds on all three hosts** (6 judged of 24 each; 18 reported per #179). 3 drift lines, all
  under 0.4%. **Neither named risk (zen5 Sdot, Saxpy) appeared.**
- **Criterion 9 passes on all three.**
- **The prediction MISSED on REPORTED: 5, not 3.** Three are #179's tier lines, as predicted. The
  other two are **Strsm's share criterion on zen4 and zen5, refused as noise-limited under rule 19**:
  the intervals cost the share 2.70 and 3.20 points, past the 2.6-point margin. This is a
  consequence of the Strsm fix I did not foresee. The share roughly DOUBLED against its pre-fix
  baselines (16% → 33%, 27% → 47%), and a doubled share carries proportionally wider intervals in
  points. Both refresh runs, at the same library code, kept the cost under 2.6 and PASSED, so this
  is a draw straddling the margin, not a change in the code.
- **A rule-19 property worth recording, not fixed here:** the refusal ignores distance. zen5's
  worst-case value, 33.1%, clears its 13.4% bar by about 20 points, yet the comparison is refused
  because the interval is wide. Rule 19 is designed that way; whether it should stay that way is a
  question for a ruling, not for this run.
