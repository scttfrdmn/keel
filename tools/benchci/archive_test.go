// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package main

// The archived path, exercised. Rule 20 (docs/rulings.md, DESIGN.md §5) shipped
// with a stated coverage limit: "no persisted CSV anywhere in the tree carries the
// seven columns the range needs", so rule 20's whole coverage was six hand-written
// fixtures in scripts/remote-exec-test.sh. That limit was
// about persisted CSVs and not about persisted DATA: every archive/*/*.txt is a
// raw benchmark log, so `summarize` re-derives the seven columns from all of them.
// This file is where that re-derivation is driven by CI rather than by memory,
// which is what #133 asked for.
//
// It lives in Go, not in scripts/remote-exec-test.sh, for one reason worth saying
// out loud: CI runs `go test ./...` and does not run remote-exec-test.sh, so a
// shell fixture is re-driven only when someone runs the remote suite by hand. The
// placement also costs the scripts/ ledger nothing, which is a real effect and not
// the justification — it is disclosed here so the next reader can tell the two
// apart.

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/perf/benchmath"
)

// As of 2026-08-30 the tracked corpus is 80 files and 1420 bounded-or-unbounded
// readings, 0 of them unbounded. These are floors, not pins: an era added later
// raises them and must not fail, while an archive that goes missing takes the
// coverage claim in rule 20 with it and must.
const (
	archivedFilesAsOf20260830 = 80
	archivedRowsAsOf20260830  = 1420
)

func archivedLogs(t *testing.T) []string {
	t.Helper()
	logs, err := filepath.Glob(filepath.Join("..", "..", "archive", "*", "*.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) < archivedFilesAsOf20260830 {
		t.Fatalf("found %d archived log(s), want >= %d: rule 20's coverage paragraph "+
			"claims this corpus exercises the archived path", len(logs), archivedFilesAsOf20260830)
	}
	return logs
}

// The premise the whole #132 derivation rests on: wherever benchstat's median CI
// is BOUNDED, its rank window has already discarded at least one sample from each
// tail, so the interval is strictly inside the range and the excluded values are
// unconstrained by it. Driven through the same benchmath.AssumeNothing that
// summarize() uses, over samples 1..n, so Lo and Hi come back as the literal
// order statistics x_(r) and x_(n+1-r) — i.e. Lo IS r.
func TestRankWindowDiscardsBothTailsWhereverItIsBounded(t *testing.T) {
	thresholds := benchmath.DefaultThresholds
	// n=5 is the unbounded case and is here on purpose: it is the reason "r >= 2
	// always" is not vacuous. QuantileCI's r hits 1 only where the confidence
	// cannot be met at all, and there Lo comes back -Inf rather than x_(1).
	for _, tc := range []struct{ n, lo, hi int }{
		{5, 0, 0}, // unbounded: lo/hi checked as ±Inf below
		{10, 2, 9},
		{20, 6, 15},
		{30, 10, 21},
		{50, 18, 32},
		{90, 36, 55},
	} {
		values := make([]float64, tc.n)
		for i := range values {
			values[i] = float64(i + 1)
		}
		s := benchmath.AssumeNothing.Summary(benchmath.NewSample(values, &thresholds), confidence)
		if tc.lo == 0 {
			if bounded(s) {
				t.Errorf("n=%d: interval came back bounded [%g, %g]; the 95%% median CI "+
					"is not attainable at n=5 and must stay unbounded", tc.n, s.Lo, s.Hi)
			}
			continue
		}
		if !bounded(s) {
			t.Fatalf("n=%d: interval unbounded [%g, %g], want [%d, %d]", tc.n, s.Lo, s.Hi, tc.lo, tc.hi)
		}
		if s.Lo != float64(tc.lo) || s.Hi != float64(tc.hi) {
			t.Errorf("n=%d: interval [%g, %g], want [%d, %d] — both bounds are order "+
				"statistics, so over samples 1..n they equal their own ranks", tc.n, s.Lo, s.Hi, tc.lo, tc.hi)
		}
		if tc.lo < 2 || tc.hi > tc.n-1 {
			t.Errorf("n=%d: rank pair [%d, %d] keeps a tail; #132's derivation needs "+
				"r >= 2 at every bounded n", tc.n, tc.lo, tc.hi)
		}
	}
}

// escapes reports whether a reading's interval reaches outside the samples it was
// computed from. Factored out so the corpus sweep below and its own negative
// control run the identical predicate: a checker that only ever sees clean input
// cannot distinguish "checked" from "skipped".
func escapes(r outRow) bool {
	return r.bounded && (r.lo < r.sampleMin || r.hi > r.sampleMax)
}

// The disparity ratio D = (max-min)/(hi-lo) has achievable set [1, ∞] at every
// bounded n, and D >= 1 is the half that a corpus can refute: it holds iff no
// interval bound escapes its samples. Over the whole archived corpus, 1420
// readings as of 2026-08-30, there are no violations — which is why no finite
// cutoff on D follows from the rank geometry, and is half of why #132 ruled that
// D be printed and nothing thresholded (2026-08-31): a scale whose floor the whole
// corpus respects needs no bar to be readable. This is the positive control for
// that claim, run in the instrument that motivated it (DESIGN.md §5 rule 11).
func TestArchivedIntervalsNeverEscapeTheirSamples(t *testing.T) {
	logs := archivedLogs(t)
	var rows, unbounded, checked int
	for _, path := range logs {
		out, _, err := summarize(path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if len(out) == 0 {
			// A log that yields nothing is a parse that greened on unread input.
			t.Errorf("%s: re-derived 0 readings", path)
		}
		rows += len(out)
		for _, r := range out {
			if !r.bounded {
				unbounded++
				continue
			}
			checked++
			if escapes(r) {
				t.Errorf("%s: %s (%s): interval [%g, %g] escapes samples [%g, %g]",
					filepath.Base(path), r.name, r.unit, r.lo, r.hi, r.sampleMin, r.sampleMax)
			}
		}
	}
	if rows < archivedRowsAsOf20260830 {
		t.Errorf("re-derived %d reading(s) from %d file(s), want >= %d", rows, len(logs), archivedRowsAsOf20260830)
	}
	// Reported, not asserted: #132 asked how much held-out data exists, and the
	// answer moves as eras are added.
	t.Logf("archived corpus: %d file(s), %d reading(s), %d unbounded, %d bounded and checked",
		len(logs), rows, unbounded, checked)

	// The negative control. Same predicate, one fabricated bound of exactly the
	// shape #116 shipped: hi above the sample maximum.
	if !escapes(outRow{bounded: true, lo: 90, hi: 2588, sampleMin: 90, sampleMax: 2295}) {
		t.Error("the escape predicate accepted a bound 293 above the sample maximum; the sweep above proves nothing")
	}
}

// #133's second criterion: bench_describe driven over a RE-DERIVED ARCHIVED
// reading rather than only over hand-written fixtures. Since #132 it is also the
// only place D's division is driven on measured intervals — every fixture in
// scripts/remote-exec-test.sh §9f has lo == hi — so these three arms are the
// renderer's arithmetic coverage and not just its shape.
func TestBenchDescribeOverReDerivedArchivedReadings(t *testing.T) {
	const exhibit = "bench-gate-p5-6ba6566-keel-zen5-20260823T004407Z-2.txt"
	path := filepath.Join("..", "..", "archive", "pinned8", exhibit)
	rows, _, err := summarize(path)
	if err != nil {
		t.Fatalf("%s: %v", exhibit, err)
	}
	dir := t.TempDir()
	seven := filepath.Join(dir, "seven.csv")
	f, err := os.Create(seven)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeCSV(f, rows); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	// The pre-#116 three-column shape, produced by dropping the four columns #116
	// added. Faithful to benchstat's own writer on its first three fields because
	// TestWriteCSVColumnsAndUnboundedTogether pins them byte-identical; NOT
	// faithful in one stated respect, which is why the assertion below is only
	// about the absence of a range: benchstat rounds the CI to %.0f%%, so over its
	// real output this file's 0.128% row prints 0.0% instead of 0.1%. Measured
	// against `go tool benchstat -format=csv` on this same archive on 2026-08-30.
	three := filepath.Join(dir, "three.csv")
	b, err := os.ReadFile(seven)
	if err != nil {
		t.Fatal(err)
	}
	var trimmed bytes.Buffer
	for _, line := range strings.Split(string(b), "\n") {
		fields := strings.Split(line, ",")
		if len(fields) == 7 {
			line = strings.Join(fields[:3], ",")
		}
		trimmed.WriteString(line + "\n")
	}
	if err := os.WriteFile(three, trimmed.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	// THE THREE ARMS OF ONE FILE, ONE HOST, ONE UNIT, ONE SWEEP — and the whole
	// argument for #132's ruling, which is why they are pinned here and not merely
	// written up. Reading down, D is 459.3, 180 and 142.6; the retired trigger named
	// the first and the third and stayed silent on the second, because what selected
	// was which side of a display quantum the surviving window's spread landed on.
	// So the marker's verdict was not monotone in the blindness it claimed to report:
	// it named 142.6 and skipped 180. That is a discontinuity, not a sensitivity.
	for _, tc := range []struct {
		what, name, csv, want string
	}{
		{
			// 28 of 30 samples inside 0.1% of each other and two contaminated draws
			// outside, so the rank window discards exactly the evidence of trouble:
			// 459 interval widths of sample disagreement under a printed 0.0%.
			what: "the worst-blinded arm, formerly named",
			name: "Ceiling/compute/scalar/threads=8",
			csv:  seven,
			want: "9.025e-05 s +/- 0.0% [9.017e-05, 0.0001154] D=459.3 (span 27.99% / interval 0.0609%)",
		},
		{
			// The row #132 was filed over: same mechanism — two recurring draws
			// ~15.5% slow at ranks 29 and 30 of 30 — reported nothing at all,
			// because the surviving window's spread rounded to 0.1% and not 0.0%.
			what: "the arm one display quantum away, formerly silent",
			name: "Ceiling/compute/avx2/threads=8",
			csv:  seven,
			want: "7.32e-05 s +/- 0.1% [7.306e-05, 8.458e-05] D=180 (span 15.74% / interval 0.0874%)",
		},
		{
			// The arm that convicts the trigger rather than merely embarrassing it:
			// strictly LESS blind than the row above by every measure — 8.11% of
			// span against 15.74% — and the trigger named this one.
			what: "the less-blinded arm the trigger named anyway",
			name: "Ceiling/compute/avx512/threads=8",
			csv:  seven,
			want: "8.787e-05 s +/- 0.0% [8.77e-05, 9.483e-05] D=142.6 (span 8.11% / interval 0.0569%)",
		},
		{
			// The empty-field guard over the historical shape: no range printed, no
			// [0, 0] invented from absent columns, and hence no D either.
			what: "the pre-#116 three-column shape",
			name: "Ceiling/compute/scalar/threads=8",
			csv:  three,
			want: "9.025e-05 s +/- 0.0%",
		},
	} {
		cmd := exec.Command("bash", "-c",
			fmt.Sprintf(`source scripts/bench.sh; bench_describe %q %q sec/op`, tc.name, tc.csv))
		cmd.Dir = filepath.Join("..", "..")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s: bench_describe: %v\n%s", tc.what, err, out)
		}
		if got := strings.TrimRight(string(out), "\n"); got != tc.want {
			t.Errorf("%s:\n got %q\nwant %q", tc.what, got, tc.want)
		}
		// The retired trigger, asserted absent by name rather than left to the want
		// strings: a renderer that still emitted it would have to fail four exact-match
		// arms to be caught, and this fails on one. Deleted 2026-08-31 (#132).
		if strings.Contains(string(out), "RANK-WINDOW-BLIND") {
			t.Errorf("%s: the thresholded marker #132 deleted is still being emitted", tc.what)
		}
		// And the disparity is not optional on a row that carries a range. Without this
		// the three-column arm above would pass a renderer that printed D nowhere.
		if hasRange := strings.Contains(tc.want, "["); hasRange != strings.Contains(string(out), " D=") {
			t.Errorf("%s: range present = %v but D present = %v", tc.what, hasRange, !hasRange)
		}
	}
}
