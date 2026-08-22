// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

// Command benchci summarizes a Go benchmark log the way `benchstat -format=csv`
// does, and differs from it in exactly one respect: the confidence interval is
// emitted at full float64 precision instead of rounded to a whole percent.
//
// WHY THIS EXISTS. benchstat's CSV `CI` column is a *display* string. In
// x/perf, `benchmath.Summary` carries `Lo`/`Hi` as float64, and
// `benchtab.ToCSV` (table.go:393) writes `cell.Summary.Center` with
// `fmt.Sprint` — full precision — beside `cell.Summary.PctRangeString()`,
// which is `fmt.Sprintf("%.0f%%", ...)`. So the machine-readable format loses
// precision the library retained, and `-format=csv` exists for programs.
//
// That rounding is not a cosmetic loss here. Every keel gate grades a
// *threshold net of the interval*, so a reported `0%` makes the bound equal the
// raw ratio — the degeneracy DESIGN.md's P4 clause exists to prevent. It
// decided a shipped verdict: janus Strsm flipped FAIL -> PASS between two runs
// on a 0.014% move in the point estimate (7.0098 -> 7.0101), because one arm's
// reported CI crossed 0.5% from `1%` to `0%` (issue #110, toolchain-notes T21).
//
// THE QUANTUM EXCEEDED EVERY MARGIN IT ADJUDICATED. One rounding step is worth
// 0.1386 on a 7.0 ratio; the margins being decided were 0.011 and 0.081. Scott's
// ruling (2026-08-19, #110 Q7): since the adjudicated margins are smaller than
// the quantum, *no arithmetic in the rounded domain* can make these verdicts
// measurement-decided — including the band-top correction, which is honest but
// leaves the verdict decided by the correction rather than by the measurement.
// The instrument must gain resolution, and the resolution already exists one
// field access away.
//
// Band-top survives for exactly one job, and `-bandtop` is it: re-reading a gate
// log whose raw samples no longer exist. Every judged run in this project's
// history is in that state, because BENCHLOG lived in a `mktemp -d` the gate's
// EXIT trap removed — so for history there is nothing to recompute from and the
// rounded reading is the only evidence there will ever be. `-bandtop` states each
// such row as the interval its rounding actually supports (a reported p% means
// true CI in [p, p+0.5)) and re-adjudicates at the pessimistic edge, which is the
// safe edge for a floor. Forward runs never use it: they have samples.
//
// FIDELITY IS MEASURED, NOT ASSERTED. `-verify` runs the pinned
// `go tool benchstat -format=csv` over the same input and requires that (a) every
// center matches bit for bit, and (b) rounding this tool's CI with `%.0f%%`
// reproduces benchstat's CI column exactly, cell for cell. The statistics are
// therefore benchstat's own — same projections, same confidence, same
// `benchmath.DefaultThresholds`, same per-unit assumption from the file's unit
// metadata — and the only difference is the formatting step this file replaces.
// Same shape as tools/shapegen's `-verify`, and for the same reason: a generator
// whose agreement with the shipped thing is asserted is a claim, not a check.
//
// WHAT IT DOES NOT DO. It does not compare two files: benchstat's `vs base`
// column, its p-values and its geomean row are absent, because no keel gate
// reads the CSV for those (`bench_compare` calls benchstat's *text* output,
// which is unaffected by this defect and stays pinned to benchstat). Handing
// this tool two files is an error rather than a silent single-file summary.
package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/perf/benchfmt"
	"golang.org/x/perf/benchmath"
	"golang.org/x/perf/benchproc"
)

// The projections benchstat parses from its own flag defaults: -table .config
// (with unit), -row .fullname, -col .file. Copied from cmd/benchstat/main.go
// rather than recalled, because a projection that differs from benchstat's
// groups the samples differently and would change the statistics while claiming
// only to reformat them.
const (
	tableProj = ".config"
	rowProj   = ".fullname"
	colProj   = ".file"
	// benchstat's -confidence default. Named here so -verify compares like with
	// like; a mismatch would show up as a CI disagreement and be blamed on the
	// formatting.
	confidence = 0.95
)

type cell struct {
	values  []float64
	summary benchmath.Summary
	sample  *benchmath.Sample
}

// key identifies one output cell. It carries no sort position: output order
// follows first observation and is tracked separately, because a position inside
// the map key makes two sightings of the same cell into two cells.
type key struct {
	unit string
	row  string
}

func main() {
	verify := flag.Bool("verify", false, "additionally run the pinned benchstat over the same input and require agreement on every center, and on every CI after rounding to %.0f%%")
	bandtop := flag.Bool("bandtop", false, "re-read a GATE log (not a benchmark log) whose raw samples are gone: treat each reported p% CI as the (p+0.5)% it actually bounds, and re-adjudicate every scaling row at that pessimistic edge")
	floor := flag.Float64("floor", 6.0, "scaling floor for -bandtop")
	strsmFloor := flag.Float64("strsm-floor", 7.0, "scaling floor for Strsm under -bandtop")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: benchci [-verify] <benchmark-log>\n")
		fmt.Fprintf(os.Stderr, "       benchci -bandtop [-floor F] [-strsm-floor F] <gate-log>\n\n")
		fmt.Fprintf(os.Stderr, "Writes benchstat's CSV shape to stdout with the CI at full precision.\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		// Two files is a comparison, which this tool deliberately does not do.
		os.Exit(2)
	}
	path := flag.Arg(0)

	if *bandtop {
		if *verify {
			// -verify compares against benchstat, which needs samples; -bandtop
			// exists precisely because there are none. Silently ignoring one of
			// the two would report a verified band-top reading, which is not a
			// thing that can exist.
			fmt.Fprintf(os.Stderr, "benchci: -verify needs the raw samples and -bandtop exists because they are gone; they cannot be combined\n")
			os.Exit(2)
		}
		if err := bandTop(path, *floor, *strsmFloor); err != nil {
			fmt.Fprintf(os.Stderr, "benchci: %s\n", err)
			os.Exit(1)
		}
		return
	}

	rows, warnings, err := summarize(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "benchci: %s\n", err)
		os.Exit(1)
	}
	// Warnings go to stderr and are never suppressed: benchstat's "need >= 6
	// samples" is exactly the kind of thing a gate must not swallow, and
	// scripts/bench.sh relays this stream.
	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "benchci: %s\n", w)
	}

	if *verify {
		if err := verifyAgainstBenchstat(path, rows); err != nil {
			fmt.Fprintf(os.Stderr, "benchci: -verify FAILED: %s\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "benchci: -verify ok: %d cell(s) agree with the pinned benchstat on every center, and on every CI after rounding to %%.0f%%%%\n", len(rows))
	}

	if err := writeCSV(os.Stdout, rows); err != nil {
		fmt.Fprintf(os.Stderr, "benchci: %s\n", err)
		os.Exit(1)
	}
}

type outRow struct {
	unit   string
	name   string
	center float64
	// ci is the half-width as a fraction of center, matching what
	// PctRangeString renders as a percent: max(|hi-center|, |center-lo|)/center.
	ci      float64
	bounded bool
}

// summarize reads path and returns one row per (unit, benchmark), in benchstat's
// output order: units in first-observed order, and within a unit, rows in
// first-observed order.
func summarize(path string) ([]outRow, []string, error) {
	filter, err := benchproc.NewFilter("*")
	if err != nil {
		return nil, nil, fmt.Errorf("constructing the identity filter: %w", err)
	}
	var parser benchproc.ProjectionParser
	tableBy, _, err := parser.ParseWithUnit(tableProj, filter)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing -table %s: %w", tableProj, err)
	}
	rowBy, err := parser.Parse(rowProj, filter)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing -row %s: %w", rowProj, err)
	}
	if _, err := parser.Parse(colProj, filter); err != nil {
		return nil, nil, fmt.Errorf("parsing -col %s: %w", colProj, err)
	}
	fields := tableBy.Fields()
	unitField := fields[len(fields)-1]
	if unitField.Name != ".unit" {
		return nil, nil, fmt.Errorf("the %s projection has no .unit field, so no row could be attributed to a unit", tableProj)
	}

	cells := map[key]*cell{}
	var order []key
	var warnings []string

	files := benchfmt.Files{Paths: []string{path}, AllowStdin: true, AllowLabels: true}
	for files.Scan() {
		switch rec := files.Result().(type) {
		case *benchfmt.SyntaxError:
			// Non-fatal, and reported rather than dropped: benchstat warns and
			// keeps going, and a log this tool silently half-read would be a
			// measurement missing rows with no line saying so.
			warnings = append(warnings, rec.Error())
		case *benchfmt.Result:
			tableKeys := tableBy.ProjectValues(rec)
			rowKey := rowBy.Project(rec)
			for i, tk := range tableKeys {
				k := key{unit: tk.Get(unitField), row: rowKey.StringValues()}
				c, ok := cells[k]
				if !ok {
					c = &cell{}
					cells[k] = c
					order = append(order, k)
				}
				c.values = append(c.values, rec.Values[i].Value)
			}
		}
	}
	if err := files.Err(); err != nil {
		return nil, nil, err
	}

	units := files.Units()
	thresholds := benchmath.DefaultThresholds
	var rows []outRow
	// `order` is already first-observation order, but the CSV groups by unit, so
	// rows are emitted unit-major with each unit's rows in observation order —
	// benchstat's own layout, which scripts/bench.sh's section parser depends on.
	sort.SliceStable(order, func(i, j int) bool {
		return unitSeq(order, order[i].unit) < unitSeq(order, order[j].unit)
	})
	for _, k := range order {
		c := cells[k]
		c.sample = benchmath.NewSample(c.values, &thresholds)
		// The conversion is required, not cosmetic: Files.Units() returns the bare
		// map type, and GetAssumption — which is how the *file's own* unit
		// metadata picks median-vs-mean — is defined on UnitMetadataMap. Reading
		// the assumption from the file is what makes these benchstat's statistics
		// rather than a second opinion about them.
		c.summary = benchfmt.UnitMetadataMap(units).GetAssumption(k.unit).Summary(c.sample, confidence)
		for _, w := range c.sample.Warnings {
			warnings = append(warnings, fmt.Sprintf("%s (%s): %s", k.row, k.unit, w))
		}
		for _, w := range c.summary.Warnings {
			warnings = append(warnings, fmt.Sprintf("%s (%s): %s", k.row, k.unit, w))
		}
		rows = append(rows, outRow{
			unit:    k.unit,
			name:    k.row,
			center:  c.summary.Center,
			ci:      ciFraction(c.summary),
			bounded: bounded(c.summary),
		})
	}
	return rows, warnings, nil
}

// gateRate matches the per-host rate line gate-p5 prints, which for a
// samples-destroyed run is the entire surviving record of the measurement:
//
//	[janus.local] Strsm: 1 thread 26.62 GFLOP/s +/- 1.0%, 8 threads 186.6 GFLOP/s +/- 0.0%
//	[vesta.local] Sgemm: boost off — 1 thread 122.4 GFLOP/s +/- 0.0%, 8 threads 771.5 GFLOP/s +/- 1.0%
//
// The `+/- p%` values are integers by construction (benchstat rounded them before
// the gate ever saw them); bench.sh prints them with one decimal after dividing by
// 100 and multiplying back, so `1.0` here means benchstat said `1%`.
//
// The optional annotation before `1 thread` is CAPTURED AND PRINTED, not skipped.
// Ten of the sixteen archived logs carry `boost off — ` there, and it names a
// different measurement condition; a re-reading that silently dropped it would
// present two populations as one. It was found by this mode refusing those ten
// outright rather than by review, which is the fail-closed on `rows == 0` earning
// its place: a pattern narrower than its input greens exactly like a clean parse
// when the only report is a count.
var gateRate = regexp.MustCompile(`\[([a-zA-Z0-9_.-]+)\] ([A-Za-z]+): ([^,]*?)1 thread ([0-9.]+) GFLOP/s \+/- ([0-9.]+)%, ([0-9]+) threads ([0-9.]+) GFLOP/s \+/- ([0-9.]+)%`)

// bandTop re-adjudicates a gate log's scaling rows at the pessimistic edge of what
// its rounded intervals support. It is a READING of old evidence, never a
// measurement: it cannot make a row better than the log recorded, and it cannot
// turn a FAIL into a PASS — treating p% as (p+0.5)% only ever widens an interval,
// so every bound it prints is <= the bound the gate printed.
func bandTop(path string, floor, strsmFloor float64) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	// Read-only, so a close error carries no information the scan did not.
	defer func() { _ = f.Close() }()

	fmt.Printf("band-top re-reading of %s (issue #110)\n", path)
	fmt.Printf("A reported `p%%` CI means the true interval is in [p, p+0.5): benchstat rounds it\n")
	fmt.Printf("to a whole percent. These rows' raw samples are gone, so this is the strongest\n")
	fmt.Printf("statement the surviving evidence supports — not a re-measurement.\n\n")
	fmt.Printf("%-14s %-7s %6s %8s %9s %6s %8s %6s  %s\n",
		"host", "routine", "bar", "raw", "as-judged", "", "band-top", "", "note")

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<20)
	rows, moved := 0, 0
	for sc.Scan() {
		m := gateRate.FindStringSubmatch(stripANSI(sc.Text()))
		if m == nil {
			continue
		}
		host, routine := m[1], m[2]
		note := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(m[3]), "—"))
		a, aci := atof(m[4]), atof(m[5])/100
		b, bci := atof(m[7]), atof(m[8])/100
		bar := floor
		if routine == "Strsm" {
			bar = strsmFloor
		}
		// bench_ratio_lo's formula, verbatim: numerator down by its interval,
		// denominator up by its own.
		asJudged := (b * (1 - bci)) / (a * (1 + aci))
		// The same formula at the top of each rounding band.
		bt := (b * (1 - (bci + 0.005))) / (a * (1 + (aci + 0.005)))
		rows++
		flag := ""
		if (asJudged >= bar) != (bt >= bar) {
			moved++
			flag = "  <== VERDICT MOVES, and only ever toward FAIL"
		}
		fmt.Printf("%-14s %-7s %6.2f %8.4f %9.4f %6s %8.4f %6s  %s%s\n",
			host, routine, bar, b/a, asJudged, verdict(asJudged, bar), bt, verdict(bt, bar), note, flag)
	}
	if err := sc.Err(); err != nil {
		return err
	}
	if rows == 0 {
		// A silent zero here would read as "every row is fine".
		return fmt.Errorf("no scaling rate lines matched in %s, so nothing was re-read; this mode wants a gate-p5 log, not a benchmark log", path)
	}
	fmt.Printf("\n%d row(s) re-read, %d verdict(s) move under the correction.\n", rows, moved)
	return nil
}

func verdict(v, bar float64) string {
	if v >= bar {
		return "PASS"
	}
	return "FAIL"
}

func atof(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		// Unreachable: the regexp already constrained these to numeric shapes.
		panic(fmt.Sprintf("benchci: %q matched a numeric group but did not parse: %v", s, err))
	}
	return v
}

var ansi = regexp.MustCompile(`\033\[[0-9;]*m`)

func stripANSI(s string) string { return ansi.ReplaceAllString(s, "") }

// unitSeq is the position at which a unit was first observed. Used as the sort
// key so the emitted CSV is unit-major without disturbing observation order
// inside a unit (sort.SliceStable).
func unitSeq(order []key, unit string) int {
	for i, k := range order {
		if k.unit == unit {
			return i
		}
	}
	return len(order)
}

// ciFraction is PctRangeString's arithmetic with the formatting removed. Read
// from x/perf's Summary.PctRangeString rather than derived: the quantity the
// gates have always divided by 100 is this one, so changing it would change
// verdicts under cover of a precision fix.
func ciFraction(s benchmath.Summary) float64 {
	if s.Center == 0 {
		return 0
	}
	d := max(s.Hi-s.Center, s.Center-s.Lo)
	return d / s.Center
}

func bounded(s benchmath.Summary) bool {
	return !isInf(s.Lo) && !isInf(s.Hi) && s.Center != 0
}

func isInf(f float64) bool { return f > 1e308 || f < -1e308 }

func writeCSV(f *os.File, rows []outRow) error {
	w := csv.NewWriter(f)
	unit := ""
	for _, r := range rows {
		if r.unit != unit {
			unit = r.unit
			// benchstat's unit section header: an empty first field, then the
			// unit. scripts/bench.sh keys on the leading comma, so this shape is
			// load-bearing and not decoration.
			if err := w.Write([]string{"", unit}); err != nil {
				return err
			}
		}
		ci := "∞"
		if r.bounded {
			// Full precision, which is the whole point of this program. %g
			// rather than %f: a 0.0003 interval must not print as 0.000.
			ci = fmt.Sprintf("%g%%", 100*r.ci)
		}
		if err := w.Write([]string{r.name, fmt.Sprint(r.center), ci}); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

// bsRef is one cell of benchstat's own summary, kept as the text it printed.
type bsRef struct {
	center string
	ci     string
}

// parseBenchstatCSV reads `benchstat -format=csv` into cells keyed unit\x00name.
//
// benchstat writes this CSV with encoding/csv, so it is read back with the same
// package rather than split on commas. A configuration line is a single quoted
// field, and `keel-pin: mask=0,1,2,3,4,5,6,7 width=8` split into eight columns
// that read as a data row named `"keel-pin: mask=0`. Every earlier comma-bearing
// config line happened to split into exactly two columns and was absorbed by the
// len < 3 guard below, so that guard was passing by luck, not by design.
func parseBenchstatCSV(out []byte) (map[string]bsRef, error) {
	cr := csv.NewReader(strings.NewReader(string(out)))
	cr.FieldsPerRecord = -1
	recs, err := cr.ReadAll()
	// Fail rather than skip: unparsed input greens exactly like clean input.
	if err != nil {
		return nil, fmt.Errorf("parsing the pinned benchstat's CSV: %w", err)
	}
	want := map[string]bsRef{}
	unit := ""
	for _, rec := range recs {
		if len(rec) < 2 {
			continue
		}
		if rec[0] == "" {
			unit = rec[1]
			continue
		}
		if len(rec) < 3 {
			continue
		}
		// benchstat emits a geomean row this tool deliberately omits.
		if rec[0] == "geomean" {
			continue
		}
		want[unit+"\x00"+rec[0]] = bsRef{center: rec[1], ci: rec[2]}
	}
	return want, nil
}

// verifyAgainstBenchstat is the differential test. It does not check that this
// tool is *right* — it checks that it is benchstat, plus resolution.
func verifyAgainstBenchstat(path string, rows []outRow) error {
	out, err := exec.Command("go", "tool", "benchstat", "-format=csv", path).Output()
	if err != nil {
		return fmt.Errorf("running the pinned benchstat: %w", err)
	}
	want, err := parseBenchstatCSV(out)
	if err != nil {
		return err
	}
	if bad := diffAgainstBenchstat(want, rows); len(bad) > 0 {
		return fmt.Errorf("%d disagreement(s) with the pinned benchstat:\n  %s", len(bad), strings.Join(bad, "\n  "))
	}
	if len(rows) == 0 {
		return fmt.Errorf("no cells were summarized, so agreement with benchstat is vacuous")
	}
	return nil
}

// diffAgainstBenchstat compares both directions and returns every disagreement.
func diffAgainstBenchstat(want map[string]bsRef, rows []outRow) []string {
	var bad []string
	for _, r := range rows {
		w, ok := want[r.unit+"\x00"+r.name]
		if !ok {
			bad = append(bad, fmt.Sprintf("%s (%s): this tool summarized it, benchstat did not", r.name, r.unit))
			continue
		}
		if got := fmt.Sprint(r.center); got != w.center {
			bad = append(bad, fmt.Sprintf("%s (%s): center %s, benchstat %s", r.name, r.unit, got, w.center))
		}
		got := "∞"
		if r.bounded {
			got = fmt.Sprintf("%.0f%%", 100*r.ci)
		}
		if got != w.ci {
			bad = append(bad, fmt.Sprintf("%s (%s): CI %s rounds to %s, benchstat %s", r.name, r.unit, fmt.Sprintf("%g%%", 100*r.ci), got, w.ci))
		}
	}
	// The reverse direction too: a cell benchstat summarized and this tool did
	// not is a row silently dropped from every criterion that reads it.
	for k := range want {
		parts := strings.SplitN(k, "\x00", 2)
		found := false
		for _, r := range rows {
			if r.unit == parts[0] && r.name == parts[1] {
				found = true
				break
			}
		}
		if !found {
			bad = append(bad, fmt.Sprintf("%s (%s): benchstat summarized it, this tool did not", parts[1], parts[0]))
		}
	}
	sort.Strings(bad)
	return bad
}
