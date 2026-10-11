// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package main

// The archived path, exercised. Rule 20 (docs/rulings.md, DESIGN.md §5) shipped
// with a stated coverage limit: "no persisted CSV anywhere in the tree carries the
// seven columns the range needs", so rule 20's whole coverage was six hand-written
// fixtures in scripts/remote-exec-test.sh. That limit was
// about persisted CSVs and not about persisted DATA: the archive is full of raw
// benchmark logs, so `summarize` re-derives the seven columns from all of them.
// This file is where that re-derivation is driven by CI rather than by memory,
// which is what #133 asked for.
//
// This paragraph used to say "every archive/*/*.txt is a raw benchmark log" and
// name the glob as the corpus. Both halves were false — a `.txt` in an archive
// directory need not be a log, and 10 files of real readings were named `.log` —
// which is the defect `benchmarkLogsUnder` below was written to fix. The sentence
// is corrected rather than deleted because it is the claim the fix falsified, and
// a reader comparing against an older checkout is owed that.
//
// It lives in Go, not in scripts/remote-exec-test.sh, for one reason worth saying
// out loud: CI runs `go test ./...` and does not run remote-exec-test.sh, so a
// shell fixture is re-driven only when someone runs the remote suite by hand. The
// placement also costs the scripts/ ledger nothing, which is a real effect and not
// the justification — it is disclosed here so the next reader can tell the two
// apart.

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"golang.org/x/perf/benchmath"
)

// Floors, not pins: an era added later raises them and must not fail, while an
// archive that goes missing takes the coverage claim in rule 20 with it and must.
// RE-DATED 2026-10-11 along with the membership fix below — they had stood at 80
// and 1420 since 2026-08-30 against a corpus of 204 files, so losing 124 of them
// would not have failed. A floor two and a half times below its subject is not a
// floor.
const (
	archivedFilesAsOf20261011 = 204
	archivedRowsAsOf20261011  = 6440
)

// benchmarkLine matches a line of Go benchmark output: the name, the iteration
// count, then a unit column. Anchored, so prose that merely says "Benchmark"
// does not qualify — every archived gate log contains that word in its caption.
var benchmarkLine = regexp.MustCompile(`^Benchmark\S+\s+\d+\s`)

// benchmarkLogsUnder returns every file under root that CARRIES benchmark output.
//
// MEMBERSHIP IS BY CONTENT, NOT BY EXTENSION (#165-adjacent, fixed 2026-10-11).
// This was `archive/*/*.txt`, which was wrong in both directions — and the two
// directions are not the same defect (`a-corpus-glob-fails-in-both-directions`):
//
//   - LOUD: a non-benchmark `.txt` dropped into an archive directory matched, then
//     re-derived 0 readings and reddened the suite. It happened twice —
//     `region-probe.txt` and `measurements.txt` — and BOTH times the file was
//     renamed to `.md` to appease the glob. That is bending the evidence to fit
//     the instrument; the membership test was the thing that was wrong.
//   - SILENT, and the reason this got fixed rather than renamed a third time: 10
//     archived files carrying real readings were OUTSIDE the corpus entirely —
//     four `neon-3x24` sweep logs and six `pinned8` janus drift arms, all named
//     `.log`. The janus six are from the campaign where within-run bimodality was
//     found (#147), i.e. the likeliest place in the tree for an interval to
//     misbehave was the place this assertion could not see.
//
// The glob also only ever looked one directory deep, so an archive with any
// internal structure was invisible. A walk has no such edge.
func benchmarkLogsUnder(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || strings.HasSuffix(path, ".gz") {
			return nil // compressed archives are evidence, not a parse target
		}
		ok, err := carriesBenchmarkOutput(path)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if ok {
			out = append(out, path)
		}
		return nil
	})
	sort.Strings(out)
	return out, err
}

// carriesBenchmarkOutput scans a file for the first benchmark line and stops
// there. Whole-file scan rather than a prefix read: a driver log can carry its
// readings thousands of lines in, and a prefix would reintroduce the silent
// direction this fix exists to close.
func carriesBenchmarkOutput(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	// Explicitly discarded rather than left bare: errcheck is in the gate's
	// golangci-lint set and this file reads the file only, so a Close error has
	// nothing to report and nowhere to report it. Caught by the judged run rather
	// than by me -- I ran go vet and gofmt locally and not golangci-lint.
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	// Archived gate logs carry single prose lines well past the 64KiB default.
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		if benchmarkLine.Match(sc.Bytes()) {
			return true, nil
		}
	}
	if err := sc.Err(); err != nil {
		return false, err
	}
	return false, nil
}

// The membership predicate's own control, on a fixture rather than on the real
// corpus — a predicate exercised only against input it already accepts cannot
// distinguish "included" from "not looked at". Both directions of the old glob's
// defect are arms here, plus the two edges that made it wrong structurally.
func TestBenchmarkLogMembershipIsByContent(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	const readings = "goos: linux\nBenchmarkSgemm/n=4096-8   \t     123\t   9876543 ns/op\nPASS\n"

	// INCLUDED: readings in a `.log`, which is exactly the 10 files the glob could
	// not see, and readings nested two deep, which its single `*/` could not reach.
	write("era/readings.log", readings)
	write("era/deep/nested/readings.txt", readings)
	// EXCLUDED: a prose note carrying no readings. region-probe.txt and
	// measurements.txt were this shape and were renamed to .md to appease the glob.
	write("era/notes.txt", "a probe of the region, by hand. No readings here.\n")
	// EXCLUDED, and distinct from the above on purpose: prose that CONTAINS the
	// word Benchmark unanchored. Every archived gate log does, so an unanchored
	// match would pull ~160 prose logs into a corpus that then re-derives 0
	// readings from each -- trading the silent direction for the loud one.
	write("era/gate.log", "   [keel-skx] Benchmark coverage for this host is stated below\n")
	// EXCLUDED, and this arm exists because its absence let a mutation through:
	// a line that STARTS with Benchmark but carries no iteration count. Without it
	// the anchor was exercised and the `\s+\d+\s` was not, so deleting the
	// iteration-count requirement left every arm green. Found by driving the
	// mutation, not by reading the fixture.
	write("era/handnote.log", "BenchmarkSgemm/n=4096 was re-run by hand; see the issue\n")
	// EXCLUDED: compressed evidence is not a parse target.
	write("era/squashed.txt.gz", readings)

	got, err := benchmarkLogsUnder(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		filepath.Join(root, "era", "deep", "nested", "readings.txt"),
		filepath.Join(root, "era", "readings.log"),
	}
	if len(got) != len(want) {
		t.Fatalf("membership = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("membership[%d] = %s, want %s", i, got[i], want[i])
		}
	}
}

func archivedLogs(t *testing.T) []string {
	t.Helper()
	logs, err := benchmarkLogsUnder(filepath.Join("..", "..", "archive"))
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) < archivedFilesAsOf20261011 {
		t.Fatalf("found %d archived log(s), want >= %d: rule 20's coverage paragraph "+
			"claims this corpus exercises the archived path", len(logs), archivedFilesAsOf20261011)
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
	if rows < archivedRowsAsOf20261011 {
		t.Errorf("re-derived %d reading(s) from %d file(s), want >= %d", rows, len(logs), archivedRowsAsOf20261011)
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
