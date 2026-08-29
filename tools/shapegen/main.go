// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

// Command shapegen is the microkernel shape generator: it emits candidate SGEMM
// microkernels, audits each one's compiled steady-state loop, and ranks them.
//
// # Why this is in the tree
//
// The sweep that chose keel's shipped tile shapes was a run artifact and was never
// committed (#107). Two live claims descend from it — gate-p2's SWEEP_BEST_IPF
// threshold and docs/spill-report.md's "no shape in the sweep clears 4.09", the
// sentence that makes 55%-of-peak unreachable by source-level shaping on Sapphire
// Rapids — so a vanished instrument was minting a gate threshold and supplying a
// go/no-go's evidence. Ruled in-tree 2026-08-18: the generator is the provenance
// of every shipped shape, it re-runs for the life of the project (the SPR campaign
// already forced one objective amendment, and arm64 will ask the same question),
// and a shape's pedigree should have history like everything else.
//
// It is apparatus, not library. tools/ is counted on the apparatus side of
// gate-docs.sh's ratio for that reason.
//
// # What it emits, and the one rule that governs it
//
// The emitted kernels are internal/vec-shaped simd code, which is the one class of
// code CLAUDE.md's prime directive says may never be written from memory. Every
// archsimd identifier in emit.go was copied from `go doc simd/archsimd` output on
// the toolchain in use, and the body idiom is internal/vec/gemm_amd64.go's. A
// reader checking this generator should re-read both, not trust this comment.
//
// # Modes
//
//	-emit SHAPE     print one candidate to stdout
//	-verify         emit the three shipped shapes and compare against the tree
//	-sweep          emit, audit and rank the whole shape space
//	-frontier       print just the best emittable zero-spill insns/FMA, for gate-p2
//	-uarch SPEC     score against NAME:WIDTH:PORTS:LATENCY (default skylake-x:4:2:4)
//
// A shape is written MRxNR/U, e.g. 2x32/4.
//
// The audit is a cross-compile for linux/amd64, so every mode runs on any host;
// no execution host is needed to produce the insns/FMA table. What needs a host is
// the GFLOP/s validation, which is P2's business and not this tool's.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/scttfrdmn/keel/internal/spill"
)

// shippedUArch is the microarchitecture the sweep reports against by default.
//
// Skylake-X numbers, and named rather than anonymous: 4-wide retire, 2 FMA pipes,
// 4-cycle FMA latency. These are vendor pipeline properties, not keel
// measurements. Change one only with its provenance beside it — which is why the
// other microarchitectures are NOT listed here from memory: -uarch takes them on the
// command line, so the SPR and arm64 re-sweeps record their constants in the run's
// own log next to whoever sourced them, rather than minting three integers in a file.
var shippedUArch = UArch{Name: "skylake-x", Width: 4, Ports: 2, Lat: 4}

// parseUArch reads NAME:WIDTH:PORTS:LATENCY.
func parseUArch(spec string) (UArch, error) {
	f := strings.Split(spec, ":")
	if len(f) != 4 || f[0] == "" {
		return UArch{}, fmt.Errorf("uarch %q is not NAME:WIDTH:PORTS:LATENCY", spec)
	}
	u := UArch{Name: f[0]}
	for i, p := range []*int{&u.Width, &u.Ports, &u.Lat} {
		if _, err := fmt.Sscanf(f[i+1], "%d", p); err != nil || *p <= 0 {
			return UArch{}, fmt.Errorf("uarch %q: field %d is not a positive integer", spec, i+2)
		}
	}
	return u, nil
}

func main() {
	var (
		emitFlag = flag.String("emit", "", "print one candidate: MRxNR/U")
		form     = flag.String("form", "broadcast", "accumulate form: broadcast | permute")
		verify   = flag.Bool("verify", false, "emit the shipped shapes and compare against internal/vec")
		sweep    = flag.Bool("sweep", false, "emit, audit and rank the whole shape space")
		frontier = flag.Bool("frontier", false, "print the best emittable zero-spill insns/FMA and nothing else")
		keep     = flag.String("keep", "", "if set, leave each emitted candidate in this directory")
		uarch    = flag.String("uarch", "skylake-x:4:2:4", "score against NAME:WIDTH:PORTS:LATENCY")
	)
	flag.Parse()

	u, err := parseUArch(*uarch)
	if err != nil {
		die(err)
	}
	shippedUArch = u

	switch {
	case *emitFlag != "":
		s, err := parseShape(*emitFlag, *form)
		if err != nil {
			die(err)
		}
		fmt.Print(s.Emit())
	case *verify:
		if !runVerify() {
			os.Exit(1)
		}
	case *sweep:
		if err := runSweep(*keep); err != nil {
			die(err)
		}
	case *frontier:
		if err := runFrontier(); err != nil {
			die(err)
		}
	default:
		flag.Usage()
		os.Exit(2)
	}
}

func die(err error) {
	fmt.Fprintf(os.Stderr, "shapegen: %v\n", err)
	os.Exit(2)
}

// parseShape reads MRxNR/U.
func parseShape(spec, form string) (Shape, error) {
	var mr, nr, u int
	if n, err := fmt.Sscanf(spec, "%dx%d/%d", &mr, &nr, &u); n != 3 || err != nil {
		return Shape{}, fmt.Errorf("shape %q is not MRxNR/U", spec)
	}
	if nr%16 != 0 || nr == 0 {
		return Shape{}, fmt.Errorf("NR=%d is not a positive multiple of 16 (a Float32x16 is 16 lanes)", nr)
	}
	f := Broadcast
	switch form {
	case "broadcast":
	case "permute":
		f = Permute
	default:
		return Shape{}, fmt.Errorf("unknown form %q", form)
	}
	return Shape{MR: mr, V: nr / 16, U: u, Form: f}, nil
}

// repoRoot locates the module root, which every mode needs: candidates are
// compiled from a directory under internal/vec and the shipped kernels are read
// from the tree, so neither can be found relative to the caller's cwd.
//
// The marker is go.mod, not .git. This asked `git rev-parse --show-toplevel`
// until 2026-08-28, which finds the *repository* root — a different thing that
// merely coincides with the module root in a checkout, and is absent from a
// `git archive` export. gate-p5's -race leg and gate-p3's OpenBLAS harness both
// ship by that export (cgo forbids cross-building them), so the git form made
// three tests here fail with `exit status 128` on every benchmark host and left
// a named P5 criterion permanently UNMEASURED. Everything these tests read is in
// the export; only the lookup wasn't.
func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("reading the working directory: %v", err)
	}
	for d := dir; ; {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d, nil
		}
		parent := filepath.Dir(d)
		if parent == d {
			return "", fmt.Errorf("no go.mod in %s or any parent", dir)
		}
		d = parent
	}
}

// audit compiles one candidate and returns its steady-state loop report.
//
// The candidate is written to a directory under internal/vec whose name begins
// with a dot. Both halves of that are deliberate. Under internal/vec, because
// CLAUDE.md puts every simd import in that directory and a generated kernel is no
// exception. Dot-prefixed, because the go command ignores such directories when
// expanding ./... — so a crashed sweep that leaves one behind cannot silently join
// `make build` or a gate's package list. It is also gitignored.
func audit(root string, s Shape, keep string) (spill.Report, error) {
	dir, err := os.MkdirTemp(filepath.Join(root, "internal", "vec"), ".shapegen-")
	if err != nil {
		return spill.Report{}, err
	}
	// Reported rather than discarded, the same way and for the same reason as
	// spill-audit's scratch dir (issue #39, whose fix this file reintroduced the
	// defect against). Cleanup is not load-bearing — the caller reads the returned
	// report, never the directory — but `-sweep` calls audit once per candidate, so
	// a silent `_ =` accumulates dot-directories under internal/vec across 34
	// shapes, and failing the audit over a cleanup would let a chmod suppress a
	// report that was produced correctly. So: say it on stderr, keep the primary
	// error. The dot prefix (see above) is what keeps a leak out of ./..., not this.
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			fmt.Fprintf(os.Stderr, "shapegen: candidate dir %s left behind (%v); "+
				"it is gitignored and skipped by ./... and can be deleted\n", dir, err)
		}
	}()

	src := s.Emit()
	if err := os.WriteFile(filepath.Join(dir, "cand.go"), []byte(src), 0o644); err != nil {
		return spill.Report{}, err
	}
	if keep != "" {
		name := fmt.Sprintf("%dx%d-u%d-%s.go", s.MR, s.NR(), s.U, s.Form)
		if err := os.WriteFile(filepath.Join(keep, name), []byte(src), 0o644); err != nil {
			return spill.Report{}, err
		}
	}

	listing, err := compile("./" + filepath.Join("internal", "vec", filepath.Base(dir)))
	if err != nil {
		return spill.Report{}, err
	}
	fns, err := spill.Parse(bytes.NewReader(listing))
	if err != nil {
		return spill.Report{}, fmt.Errorf("parsing the listing: %v", err)
	}
	f, err := spill.Find(fns, s.Name())
	if err != nil {
		return spill.Report{}, err
	}
	loop, err := f.SteadyLoop()
	if err != nil {
		return spill.Report{}, err
	}
	return spill.Audit(f, loop), nil
}

// compile is spill-audit's compile step, with the same environment for the same
// reason: the audit must read the object code the execution hosts run, and a
// darwin/arm64 build has no vector instructions in it at all.
func compile(pkg string) ([]byte, error) {
	root, err := repoRoot()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("go", "build", "-gcflags=-S", "-o", os.DevNull, pkg)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"GOEXPERIMENT=simd",
		"GOOS=linux",
		"GOARCH=amd64",
		"CGO_ENABLED=0",
	)
	var errb bytes.Buffer
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go build -gcflags=-S %s: %v\n%s", pkg, err, errb.String())
	}
	return errb.Bytes(), nil
}

// shipped is the three shapes that exist in internal/vec/gemm_amd64.go, with the
// unroll each was written at. runVerify checks the emitter against them.
var shipped = []Shape{
	{MR: 2, V: 2, U: 4, Form: Broadcast},
	{MR: 4, V: 2, U: 1, Form: Broadcast},
	{MR: 6, V: 2, U: 4, Form: Broadcast},
}

// runVerify is this instrument's mint verification.
//
// #107's finding is that the vanished generator's output "cannot be re-checked at
// all" — there is not even a pin, because the tool is gone. For three shapes there
// is something better than a pin available, and it is used here: 2x32, 4x32 and
// 6x32 are in the tree, hand-written, differentially tested and benchmarked. If
// the generator re-emits them, then its emission is the code the project actually
// writes, and its readings for the other shapes are readings of the same kind of
// kernel.
//
// Two comparisons, and both are reported because they answer different questions:
//
//   - Text, against the function body in internal/vec/gemm_amd64.go with comments
//     and blank lines removed. This is the strong form. It can fail for reasons
//     that do not matter (a variable renamed in the tree), so a text mismatch
//     prints the diff and is a finding to read, not automatically a defect.
//   - The audit report, field for field, against a compile of the shipped kernel
//     itself. This is the form that binds: it says the emitted shape has the same
//     instruction count, FMA count, spill count, copy count and broadcast count as
//     what ships — every quantity the objective reads. Naming cannot affect it.
func runVerify() bool {
	root, err := repoRoot()
	if err != nil {
		die(err)
	}
	shippedSrc, err := os.ReadFile(filepath.Join(root, "internal", "vec", "gemm_amd64.go"))
	if err != nil {
		die(err)
	}

	// One compile of the shipped package serves every shape.
	listing, err := compile("./internal/vec")
	if err != nil {
		die(err)
	}
	fns, err := spill.Parse(bytes.NewReader(listing))
	if err != nil {
		die(err)
	}

	ok := true
	for _, s := range shipped {
		fmt.Printf("%s\n", s.Label())

		got := normalize(funcBody(s.Emit(), s.Name()))
		want := normalize(funcBody(string(shippedSrc), s.Name()))
		switch {
		case want == "":
			fmt.Printf("  text:  NOT FOUND in internal/vec/gemm_amd64.go — %s is not a shipped kernel\n", s.Name())
			ok = false
		case got == want:
			fmt.Printf("  text:  identical to the shipped body (%d lines, comments and blanks removed)\n",
				len(strings.Split(want, "\n")))
		default:
			fmt.Printf("  text:  DIFFERS from the shipped body\n")
			printDiff(want, got)
			ok = false
		}

		f, err := spill.Find(fns, s.Name())
		if err != nil {
			fmt.Printf("  audit: %v\n", err)
			ok = false
			continue
		}
		loop, err := f.SteadyLoop()
		if err != nil {
			fmt.Printf("  audit: %v\n", err)
			ok = false
			continue
		}
		wantR := spill.Audit(f, loop)
		gotR, err := audit(root, s, "")
		if err != nil {
			fmt.Printf("  audit: emitting and compiling the candidate: %v\n", err)
			ok = false
			continue
		}
		if d := reportDiff(wantR, gotR); d != "" {
			fmt.Printf("  audit: DIFFERS from the shipped kernel's report: %s\n", d)
			ok = false
		} else {
			fmt.Printf("  audit: identical report — %d insns, %d FMAs, %d spills, %d copies, %d broadcasts, %d nops (%.3f insns/FMA)\n",
				gotR.Insns, gotR.Arith, gotR.Spills(), gotR.VecCopies, gotR.Broadcasts, gotR.Nops,
				float64(gotR.Insns)/float64(gotR.Arith))
		}
	}
	if ok {
		fmt.Println("\nverify: the emitter reproduces all three shipped shapes, by text and by audit.")
	} else {
		fmt.Println("\nverify: FAILED — the emitter does not reproduce the tree. Its sweep readings are not trustworthy until it does.")
	}
	return ok
}

// reportDiff names every field on which two audit reports disagree.
func reportDiff(want, got spill.Report) string {
	var d []string
	cmp := func(name string, w, g int) {
		if w != g {
			d = append(d, fmt.Sprintf("%s %d vs %d", name, w, g))
		}
	}
	cmp("insns", want.Insns, got.Insns)
	cmp("arith", want.Arith, got.Arith)
	cmp("spills", want.Spills(), got.Spills())
	cmp("copies", want.VecCopies, got.VecCopies)
	cmp("broadcasts", want.Broadcasts, got.Broadcasts)
	cmp("nops", want.Nops, got.Nops)
	cmp("calls", len(want.Calls), len(got.Calls))
	cmp("bounds-check exits", len(want.PanicExits), len(got.PanicExits))
	cmp("other mem refs", len(want.OtherMem), len(got.OtherMem))
	return strings.Join(d, ", ")
}

// funcBody returns the text of one top-level function, brace to brace.
func funcBody(src, name string) string {
	i := strings.Index(src, "\nfunc "+name+"(")
	if i < 0 {
		return ""
	}
	rest := src[i+1:]
	end := strings.Index(rest, "\n}\n")
	if end < 0 {
		return ""
	}
	return rest[:end+3]
}

// normalize drops comment-only lines, blank lines and trailing space, so the
// comparison is about code rather than prose.
func normalize(body string) string {
	var out []string
	for _, ln := range strings.Split(body, "\n") {
		t := strings.TrimSpace(ln)
		if t == "" || strings.HasPrefix(t, "//") {
			continue
		}
		out = append(out, strings.TrimRight(ln, " \t"))
	}
	return strings.Join(out, "\n")
}

// printDiff prints the first few differing lines of two normalized bodies.
func printDiff(want, got string) {
	w, g := strings.Split(want, "\n"), strings.Split(got, "\n")
	shown := 0
	for i := 0; i < len(w) || i < len(g); i++ {
		var a, b string
		if i < len(w) {
			a = w[i]
		}
		if i < len(g) {
			b = g[i]
		}
		if a != b {
			fmt.Printf("    line %d\n      tree: %s\n      emit: %s\n", i+1, a, b)
			if shown++; shown >= 5 {
				fmt.Printf("    (further differences not shown; %d tree lines, %d emitted)\n", len(w), len(g))
				return
			}
		}
	}
}

// space is the shape space the sweep enumerates.
//
// Stated here rather than derived from the old sweep's count, and the difference
// matters: docs/spill-report.md:202 says the vanished sweep audited 115 shapes in
// 60 broadcast and 55 Permute, and nothing records what it enumerated to get
// those. This enumeration is chosen on its own grounds and its count is whatever
// it is. Matching 115 by tuning the ranges would manufacture agreement with a
// number whose derivation is lost, which is the opposite of what #107 asks for.
//
// The ranges: MR 1..8 because an accumulator count above 8 spills on go1.26.x
// (T10) and MR=1 is the degenerate row that shows the loop overhead undivided; V
// 1..4 for NR of 16..64, past which one B panel load no longer feeds the rows it
// costs; U 1,2,4,8 because the unroll interacts with register pressure only
// through hoisted A scalars, whose count is MR*U.
func space() []Shape {
	var out []Shape
	for _, form := range []Form{Broadcast, Permute} {
		for mr := 1; mr <= 8; mr++ {
			for v := 1; v <= 4; v++ {
				for _, u := range []int{1, 2, 4, 8} {
					s := Shape{MR: mr, V: v, U: u, Form: form}
					if form == Permute && !s.PermuteWindowExact() {
						continue
					}
					out = append(out, s)
				}
			}
		}
	}
	return out
}

type row struct {
	Shape  Shape
	Report spill.Report
	Score  Score
	Err    error
}

func runSweep(keep string) error {
	root, err := repoRoot()
	if err != nil {
		return err
	}
	if keep != "" {
		if err := os.MkdirAll(keep, 0o755); err != nil {
			return err
		}
	}
	all := space()
	fmt.Printf("shapegen sweep: %d shapes (%d broadcast, %d permute), uarch %s (width %d, %d FMA ports, %d-cycle latency)\n",
		len(all), countForm(all, Broadcast), countForm(all, Permute),
		shippedUArch.Name, shippedUArch.Width, shippedUArch.Ports, shippedUArch.Lat)
	fmt.Printf("objective: cycles = max(insns/width, FMAs/ports, (FMAs/chains)*latency); dependency-bound iff chains < ports*latency = %d\n\n",
		shippedUArch.Ports*shippedUArch.Lat)

	rows := auditAll(root, keep)
	for _, r := range rows {
		if r.Err != nil {
			fmt.Fprintf(os.Stderr, "  %s: %v\n", r.Shape.Label(), r.Err)
		}
	}

	fmt.Printf("%-20s %6s %5s %6s %6s %7s %8s %12s %9s\n",
		"shape", "insns", "nops", "FMAs", "chains", "spills", "insns/FMA", "bound", "flops/cyc")
	for _, r := range rows {
		if r.Err != nil {
			fmt.Printf("%-20s %s\n", r.Shape.Label(), "UNAUDITED (see stderr)")
			continue
		}
		fmt.Printf("%-20s %6d %5d %6d %6d %7d %8.3f %12s %9.2f\n",
			r.Shape.Label(), r.Report.Insns, r.Report.Nops, r.Report.Arith, r.Score.Accs,
			r.Report.Spills(), r.Score.InsnsPerFMA(), r.Score.Bound, r.Score.FlopsPerCycle())
	}

	summarize(rows)
	return nil
}

// auditAll audits the whole space, recording rather than judging failures: a shape
// that will not compile becomes a row with Err set, and what that means is the
// caller's policy. A sweep prints it and carries on; the frontier refuses to state
// a figure over it, because the shape that would not compile may be the shape that
// would have won.
func auditAll(root, keep string) []row {
	var rows []row
	for _, s := range space() {
		r, err := audit(root, s, keep)
		if err != nil {
			rows = append(rows, row{Shape: s, Err: err})
			continue
		}
		rows = append(rows, row{Shape: s, Report: r, Score: shippedUArch.Score(s, r.Insns)})
	}
	return rows
}

// contender reports whether a swept row may set the frontier: audited, and
// zero-spill. Emittability is upstream of this — space() never enumerates a shape
// whose A-window load cannot be covered, and Err catches one that will not compile
// — so "emittable, zero-spill", the terms gate-p2's SWEEP_BEST_IPF is defined in,
// is exactly this predicate over that enumeration.
func contender(r row) bool { return r.Err == nil && r.Report.Spills() == 0 }

// best returns the top row under an ordering, and false when there were none.
//
// False rather than a zero row on purpose: an empty contender set must not read as
// a frontier of 0.000 insns/FMA, which is the most permissive figure gate-p2's
// shape guard could possibly be handed — a broken enumeration would silently widen
// the very threshold it is supposed to pin.
func best(rows []row, better func(a, b row) bool) (row, bool) {
	var top row
	found := false
	for _, r := range rows {
		if !found || better(r, top) {
			top, found = r, true
		}
	}
	return top, found
}

func fewestInsnsPerFMA(a, b row) bool { return a.Score.InsnsPerFMA() < b.Score.InsnsPerFMA() }
func mostFlopsPerCycle(a, b row) bool { return a.Score.FlopsPerCycle() > b.Score.FlopsPerCycle() }

// runFrontier prints the single figure gate-p2's SWEEP_BEST_IPF states, and nothing
// else, so that constant can be reconciled against a live derivation on every gate
// run instead of trusted (#107). It is computed across both forms rather than the
// broadcast form alone: Permute contributes no zero-spill shape today, and stating
// the frontier over the whole space is what keeps that a finding rather than an
// assumption baked into the gate.
//
// A full audit is ~7s, so the gate re-derives rather than caching. Any shape that
// fails to compile is fatal here.
func runFrontier() error {
	root, err := repoRoot()
	if err != nil {
		return err
	}
	rows := auditAll(root, "")
	var zero []row
	for _, r := range rows {
		if r.Err != nil {
			return fmt.Errorf("%s did not audit (%v); the frontier cannot be stated over an incomplete sweep", r.Shape.Label(), r.Err)
		}
		if contender(r) {
			zero = append(zero, r)
		}
	}
	top, ok := best(zero, fewestInsnsPerFMA)
	if !ok {
		return fmt.Errorf("no emittable zero-spill shape among %d audited: there is no frontier to state", len(rows))
	}
	// Figure, contender count, then the label — the label is several words, so it
	// goes last and a shell reader can take it as the remainder of the line.
	fmt.Printf("%.3f %d %s\n", top.Score.InsnsPerFMA(), len(zero), top.Shape.Label())
	return nil
}

func countForm(all []Shape, f Form) int {
	n := 0
	for _, s := range all {
		if s.Form == f {
			n++
		}
	}
	return n
}

// summarize reports each form's frontier under both objectives, and says plainly
// where the corrected one disagrees with the old one. The disagreement is the
// result: if ranking by chain-aware cycles picked the same shape as ranking by
// insns/FMA, the amendment would have been unnecessary.
func summarize(rows []row) {
	for _, form := range []Form{Broadcast, Permute} {
		var zero []row
		for _, r := range rows {
			if contender(r) && r.Shape.Form == form {
				zero = append(zero, r)
			}
		}
		total := 0
		for _, r := range rows {
			if r.Shape.Form == form {
				total++
			}
		}
		fmt.Printf("\n%s: %d shapes, %d zero-spill\n", form, total, len(zero))
		if len(zero) == 0 {
			if form == Permute {
				fmt.Printf("  none can be, and the constraints that exclude them are arithmetic rather than empirical:\n")
				fmt.Printf("  one 16-lane A-window load is in bounds only when MR*U >= 16 and covers every index the\n")
				fmt.Printf("  body needs only when MR*U <= 16, so MR*U == 16 — and the form hoists one index vector\n")
				fmt.Printf("  per row per k-step, putting exactly 16 of them live at once against the 15 SIMD values\n")
				fmt.Printf("  go1.26.x offers (T10). Emittable and in-budget are mutually exclusive under the shipped\n")
				fmt.Printf("  A-panel layout, so this form has no zero-spill shape at all.\n")
				fmt.Printf("  This is what retired gate-p2's SWEEP_BEST_IPF=4.438, which docs/spill-report.md:206\n")
				fmt.Printf("  attributed to `2x64 u=2` in this form: that shape guarantees only 4 A floats, so whatever\n")
				fmt.Printf("  kernel produced 4.438 did not read its A panel the way the shipped kernels read theirs.\n")
				fmt.Printf("  Ruled 2026-08-18 (#107): the threshold is now 4.625, the best figure an emittable\n")
				fmt.Printf("  zero-spill shape reaches, and both gates reconcile it against -frontier rather than\n")
				fmt.Printf("  reading it. See #107.\n")
			}
			continue
		}
		// len(zero) > 0 is established above, so both are found. Same two orderings
		// -frontier uses, from the same definitions: the figure gate-p2 reconciles
		// against must be the figure this table publishes.
		old, _ := best(zero, fewestInsnsPerFMA)
		new_, _ := best(zero, mostFlopsPerCycle)
		fmt.Printf("  old objective (insns/FMA only):      %s at %.3f insns/FMA\n",
			old.Shape.Label(), old.Score.InsnsPerFMA())
		fmt.Printf("    %s\n", old.Score.Derivation(shippedUArch))
		fmt.Printf("  corrected objective (chain-aware):   %s at %.2f flops/cycle\n",
			new_.Shape.Label(), new_.Score.FlopsPerCycle())
		fmt.Printf("    %s\n", new_.Score.Derivation(shippedUArch))
		fmt.Printf("  %s\n", Ceiling(shippedUArch, new_.Score, new_.Report.Nops))
		if old.Shape.Label() == new_.Shape.Label() {
			fmt.Printf("  the two objectives agree on this form's optimum\n")
		} else {
			fmt.Printf("  THE TWO OBJECTIVES DISAGREE: the old optimum is %s-bound, at %.2f flops/cycle against %.2f\n",
				old.Score.Bound, old.Score.FlopsPerCycle(), new_.Score.FlopsPerCycle())
		}
	}
}
