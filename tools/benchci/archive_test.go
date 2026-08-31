// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package main

// The archived path, exercised. Rule 20 (docs/rulings.md, DESIGN.md §5) shipped
// with a stated coverage limit: "no persisted CSV anywhere in the tree carries the
// seven columns the range needs", so the RANK-WINDOW-BLIND marker's whole coverage
// was six hand-written fixtures in scripts/remote-exec-test.sh. That limit was
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
// cutoff on D follows from the rank geometry, and why #132's candidate had to be
// argued from the data instead. This is the positive control for that claim, run
// in the instrument that motivated it (DESIGN.md §5 rule 11).
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
// reading rather than only over hand-written fixtures, with the marker made to
// fire on purpose there. The third arm is the one that matters most — it is the
// miss #132 is open about.
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

	for _, tc := range []struct {
		what, name, csv, want string
		marker                bool
	}{
		{
			// Fires on purpose, over a re-derived archived reading: 28 of 30 samples
			// inside 0.1% of each other and two contaminated draws outside, so the
			// rank window discards exactly the evidence of trouble.
			what:   "the archived reading the marker names",
			name:   "Ceiling/compute/scalar/threads=8",
			csv:    seven,
			want:   "9.025e-05 s +/- 0.0% [9.017e-05, 0.0001154] RANK-WINDOW-BLIND(span 27.99% under a 0.0% interval)",
			marker: true,
		},
		{
			// The miss, pinned as a miss (#132). Same file, adjacent arm, same
			// mechanism — two recurring draws ~15.5% slow at ranks 29 and 30 of 30 —
			// and the marker stays silent because the surviving window's spread
			// rounded to 0.1% instead of 0.0%. The trigger's equality at zero is
			// what separates these two rows; nothing about the contamination does.
			// This pin is behaviour as shipped, NOT an assertion that 0.1% asserts
			// anything the range refutes.
			what: "the adjacent arm one display quantum away, which it does not name",
			name: "Ceiling/compute/avx2/threads=8",
			csv:  seven,
			want: "7.32e-05 s +/- 0.1% [7.306e-05, 8.458e-05]",
		},
		{
			// The empty-field guard over the historical shape: no range printed, no
			// [0, 0] invented from absent columns, no marker.
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
		if fired := strings.Contains(string(out), "RANK-WINDOW-BLIND"); fired != tc.marker {
			t.Errorf("%s: marker fired = %v, want %v", tc.what, fired, tc.marker)
		}
	}
}
