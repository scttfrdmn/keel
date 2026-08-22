// Copyright 2026 The keel Authors
// SPDX-License-Identifier: BSD-3-Clause

package main

import "testing"

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
