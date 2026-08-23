// Copyright 2026 The keel Authors
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"bytes"
	"encoding/csv"
	"os"
	"testing"

	"golang.org/x/perf/benchmath"
)

// keel-zen5's 8-thread compute ceiling on 6ba6566, take three, verbatim. Ten
// samples, two of them low, and at n=10 benchstat's median CI is [x_(2), x_(9)] =
// [1995, 2292] about a center of 2291.5 — reaching DOWN 296.5 and UP 0.5.
//
// This is the fixture the whole of #116 rests on, so it is the real numbers rather
// than a constructed shape: a synthetic one-sided sample would demonstrate the
// arithmetic without pinning the case that actually shipped a fabricated
// denominator.
var zen5Ceiling = []float64{1900, 1995, 2291, 2291, 2291, 2292, 2292, 2292, 2292, 2295}

// The mirroring is REAL and is deliberately still here: ciFraction keeps
// reproducing benchstat's symmetric display half-width, because -verify compares
// against it and changing it would move historical verdicts under cover of a
// precision fix. What #116 changed is what the *consumer* divides by. This pins
// both halves of that: the symmetric fraction stays 12.939...%, and the honest Hi
// stays 2292 — the two must not be allowed to drift into agreement, because the
// gap between them IS the defect.
func TestCIFractionStaysSymmetricWhileBoundsStayHonest(t *testing.T) {
	s := benchmath.Summary{Center: 2291.5, Lo: 1995, Hi: 2292}
	got := ciFraction(s)
	// 296.5/2291.5 — the DOWNWARD half-width, which max() selects.
	const want = 12.93912284529784 / 100
	if got != want {
		t.Errorf("ciFraction = %v, want %v (max() must keep taking the larger, downward half-width)", got, want)
	}
	// The fabrication, stated as the arithmetic the gate used to perform.
	if fab := s.Center * (1 + got); fab <= 2295 {
		t.Errorf("center*(1+ci) = %v; the whole point is that it exceeds the sample maximum 2295", fab)
	}
	lo, hi := sampleRange(zen5Ceiling)
	if s.Lo < lo || s.Hi > hi {
		t.Errorf("interval [%v, %v] escapes the sample range [%v, %v]; a rank-window bound cannot", s.Lo, s.Hi, lo, hi)
	}
}

func TestSampleRange(t *testing.T) {
	lo, hi := sampleRange(zen5Ceiling)
	if lo != 1900 || hi != 2295 {
		t.Errorf("sampleRange = [%v, %v], want [1900, 2295]", lo, hi)
	}
	// Empty must not report a confident [0, 0] that a reader would take for data.
	if lo, hi := sampleRange(nil); lo != 0 || hi != 0 {
		t.Errorf("sampleRange(nil) = [%v, %v], want [0, 0]", lo, hi)
	}
}

// The seven-column shape, and the invariant that matters most about it: ci, lo and
// hi go unbounded TOGETHER. A row whose ci read ∞ while lo/hi looked usable would
// let bench_ratio_lo proceed where it has always bailed, so the new columns would
// have widened what the gates judge instead of only making an existing judgement
// honest.
func TestWriteCSVColumnsAndUnboundedTogether(t *testing.T) {
	rows := []outRow{
		{unit: "GFLOP/s", name: "Judged-8", center: 2291.5, ci: 0.1293912284529784,
			lo: 1995, hi: 2292, sampleMin: 1900, sampleMax: 2295, bounded: true},
		{unit: "GFLOP/s", name: "Unbounded-8", center: 100, ci: 0,
			lo: 90, hi: 110, sampleMin: 80, sampleMax: 120, bounded: false},
	}
	f, err := os.CreateTemp(t.TempDir(), "csv")
	if err != nil {
		t.Fatal(err)
	}
	if err := writeCSV(f, rows); err != nil {
		t.Fatalf("writeCSV: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	cr := csv.NewReader(bytes.NewReader(b))
	// Ragged by design and unchanged by #116: a unit header is two fields and a
	// data row is seven, which is benchstat's own layout. parseBenchstatCSV sets
	// the same thing for the same reason.
	cr.FieldsPerRecord = -1
	recs, err := cr.ReadAll()
	if err != nil {
		t.Fatalf("the emitted CSV does not parse: %v", err)
	}
	// unit header, then two data rows.
	if len(recs) != 3 {
		t.Fatalf("got %d record(s), want 3: %v", len(recs), recs)
	}
	if recs[0][0] != "" || recs[0][1] != "GFLOP/s" {
		t.Errorf("unit header = %v, want [\"\", GFLOP/s]; bench.sh keys on the leading comma", recs[0])
	}
	// The first three fields must stay byte-identical to the pre-#116 shape:
	// every archived CSV and every awk -F, consumer reads by position.
	want := []string{"Judged-8", "2291.5", "12.93912284529784%", "1995", "2292", "1900", "2295"}
	for i, w := range want {
		if recs[1][i] != w {
			t.Errorf("judged row field %d = %q, want %q", i, recs[1][i], w)
		}
	}
	for _, i := range []int{2, 3, 4} {
		if recs[2][i] != "∞" {
			t.Errorf("unbounded row field %d = %q, want ∞: ci/lo/hi must go unbounded together", i, recs[2][i])
		}
	}
	// The range is still real on an unbounded row: it is what was observed, not
	// what was inferred, so it survives the interval being unestablished.
	if recs[2][5] != "80" || recs[2][6] != "120" {
		t.Errorf("unbounded row range = [%q, %q], want [80, 120]", recs[2][5], recs[2][6])
	}
}

// A configuration line benchstat quotes because it contains a comma is not a
// data row. `keel-pin:` is the case that caught this: split on commas it yields
// sixteen fields since the 2026-08-22 spread amendment doubled its comma-bearing
// fields (mask and doms), eight before that, either way breaking the len < 3
// guard and landing in want as a cell no
// summarizer can reproduce — so -verify failed on every pinned host for a
// metadata line. The other two shapes here (one comma, and a doubled-quote
// nesting) split into exactly two fields and were absorbed by that guard, which
// is why the guard looked correct for as long as it did.
func TestParseBenchstatCSVSkipsConfigLines(t *testing.T) {
	const in = `"keel-pin: mask=0,8,16,24,32,40,48,56 width=8 doms=0,8,16,24,32,40,48,56 nodedoms=12"
keel-bench-cpu: AMD EPYC 9R14
"keel-bench-clock-mhz: 2600-3780 (snapshot, not sustained)"
"keel-bench-ceiling: name=Ceiling/stream/axpy/threads=1 sizing=""256 MB floor"" rfo=""write-allocate, i.e. x1.33"""
,archive.txt,
,GFLOP/s,CI
Scale/Sgemm/n=4096/threads=8-8,704.35,0%
geomean,704.35,
`
	want, err := parseBenchstatCSV([]byte(in))
	if err != nil {
		t.Fatalf("parseBenchstatCSV: %v", err)
	}
	if len(want) != 1 {
		t.Errorf("got %d cell(s), want 1 (the config lines are not cells): %v", len(want), want)
	}
	cell, ok := want["GFLOP/s\x00Scale/Sgemm/n=4096/threads=8-8"]
	if !ok {
		t.Fatalf("the one real cell is missing: %v", want)
	}
	if cell.center != "704.35" || cell.ci != "0%" {
		t.Errorf("cell = %+v, want center 704.35 ci 0%%", cell)
	}
}

// Unparsed input must not green like clean input: a CSV this tool cannot read is
// a failure to agree with benchstat, not an agreement over zero cells.
func TestParseBenchstatCSVFailsClosed(t *testing.T) {
	if _, err := parseBenchstatCSV([]byte(`,unit,CI` + "\n" + `name,1"2,0%` + "\n")); err == nil {
		t.Error("malformed CSV parsed without error; the verifier would green on it")
	}
}

// Both directions of the differential still bite after the extraction.
func TestDiffAgainstBenchstatBothDirections(t *testing.T) {
	want := map[string]bsRef{
		"GFLOP/s\x00Agree-8":         {center: "100", ci: "1%"},
		"GFLOP/s\x00OnlyBenchstat-8": {center: "50", ci: "2%"},
	}
	rows := []outRow{
		{unit: "GFLOP/s", name: "Agree-8", center: 100, ci: 0.014, bounded: true},
		{unit: "GFLOP/s", name: "OnlyThisTool-8", center: 7, ci: 0.01, bounded: true},
	}
	bad := diffAgainstBenchstat(want, rows)
	if len(bad) != 2 {
		t.Fatalf("got %d disagreement(s), want 2: %v", len(bad), bad)
	}
	if bad[0] != "OnlyBenchstat-8 (GFLOP/s): benchstat summarized it, this tool did not" {
		t.Errorf("reverse direction: %q", bad[0])
	}
	if bad[1] != "OnlyThisTool-8 (GFLOP/s): this tool summarized it, benchstat did not" {
		t.Errorf("forward direction: %q", bad[1])
	}
}
