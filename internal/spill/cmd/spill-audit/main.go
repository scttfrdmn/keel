// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

// Command spill-audit is the P2 gate's compile-time check (DESIGN.md §4/P2).
//
// It compiles a package for linux/amd64 with GOEXPERIMENT=simd, reads the
// assembly listing the compiler emits, finds the steady-state loop of each named
// function, and reports what is in it. Exit status 1 means a criterion failed,
// so the gate can use it directly.
//
// Two modes, because P2 has two compile-time properties to check:
//
//	-mode spill     no vector register may be moved to or from the stack inside
//	                the loop body, the body may contain no calls, and no branch
//	                out of it may lead to a runtime panic. This is the microkernel
//	                criterion: zero spills, no calls in the K-loop, and pre-sliced
//	                panels (i.e. bounds checks eliminated) — all three of P2's
//	                compile-time requirements, checked on the loop body rather
//	                than on the whole function, which is where they are stated.
//	-mode nomemory  the loop body may not reference memory at all. This is the
//	                peak kernel's property 1 (issue #11) — the one property of
//	                the percent-of-peak denominator that arithmetic cannot check.
//
// The target is always linux/amd64 with the experiment on, regardless of the
// host, because that is the object code the execution hosts in docs/hosts.md run.
// Auditing a darwin/arm64 build would audit a program with no vector
// instructions in it at all.
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

func main() {
	var (
		pkg     = flag.String("pkg", "", "package to compile and audit, e.g. ./internal/kern")
		funcs   = flag.String("func", "", "comma-separated function names (short names, not symbols)")
		mode    = flag.String("mode", "spill", "spill | nomemory")
		ssaDir  = flag.String("ssa", "", "if set, also write GOSSAFUNC html per function into this directory")
		verbose = flag.Bool("v", false, "print the steady-state loop body")
	)
	flag.Parse()

	if *pkg == "" || *funcs == "" {
		fmt.Fprintln(os.Stderr, "spill-audit: -pkg and -func are required")
		os.Exit(2)
	}
	if *mode != "spill" && *mode != "nomemory" {
		fmt.Fprintf(os.Stderr, "spill-audit: unknown -mode %q\n", *mode)
		os.Exit(2)
	}

	listing, err := compile(*pkg, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "spill-audit: %v\n", err)
		os.Exit(2)
	}
	fns, err := spill.Parse(bytes.NewReader(listing))
	if err != nil {
		fmt.Fprintf(os.Stderr, "spill-audit: parsing the listing: %v\n", err)
		os.Exit(2)
	}

	failed := false
	for _, name := range strings.Split(*funcs, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if !audit(fns, name, *mode, *verbose) {
			failed = true
		}
		if *ssaDir != "" {
			if err := emitSSA(*pkg, name, *ssaDir); err != nil {
				fmt.Fprintf(os.Stderr, "  %s: ssa.html not archived: %v\n", name, err)
				failed = true
			}
		}
	}
	if failed {
		os.Exit(1)
	}
}

// audit reports one function, returning whether it passed.
func audit(fns []spill.Func, name, mode string, verbose bool) bool {
	f, err := spill.Find(fns, name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", name, err)
		return false
	}
	loop, err := f.SteadyLoop()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return false
	}
	r := spill.Audit(f, loop)
	fmt.Println(r.Summary())

	ok := true
	switch mode {
	case "spill":
		if n := r.Spills(); n > 0 {
			ok = false
			fmt.Fprintf(os.Stderr, "%s: %d vector stack reference(s) in the steady-state K-loop:\n", name, n)
			for _, in := range r.VecStack {
				fmt.Fprintf(os.Stderr, "    %s\n", in)
			}
		}
		if n := len(r.Calls); n > 0 {
			ok = false
			fmt.Fprintf(os.Stderr, "%s: %d call(s) in the steady-state K-loop:\n", name, n)
			for _, in := range r.Calls {
				fmt.Fprintf(os.Stderr, "    %s\n", in)
			}
		}
		if n := len(r.PanicExits); n > 0 {
			ok = false
			fmt.Fprintf(os.Stderr, "%s: %d surviving bounds check(s) in the steady-state K-loop "+
				"(branch out of the body to a runtime panic):\n", name, n)
			for _, in := range r.PanicExits {
				fmt.Fprintf(os.Stderr, "    %s\n", in)
			}
		}
	case "nomemory":
		if mem := r.Memory(); len(mem) > 0 {
			ok = false
			fmt.Fprintf(os.Stderr, "%s: %d memory reference(s) in a loop that must be register-only:\n", name, len(mem))
			for _, in := range mem {
				fmt.Fprintf(os.Stderr, "    %s\n", in)
			}
		}
	}
	if verbose {
		for _, in := range loop.Insns {
			fmt.Printf("    %s\n", in)
		}
	}
	return ok
}

// compile builds pkg for the execution hosts' target and returns the assembly
// listing the compiler wrote to stderr.
//
// -o os.DevNull keeps the object out of the build cache's way and makes it
// obvious that the artifact wanted here is the listing, not a binary.
func compile(pkg string, extra []string) ([]byte, error) {
	args := append([]string{"build", "-gcflags=-S", "-o", os.DevNull}, extra...)
	args = append(args, pkg)
	cmd := exec.Command("go", args...)
	cmd.Env = append(os.Environ(),
		"GOEXPERIMENT=simd",
		"GOOS=linux",
		"GOARCH=amd64",
		"CGO_ENABLED=0",
	)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go build -gcflags=-S %s: %v\n%s", pkg, err, errb.String())
	}
	return errb.Bytes(), nil
}

// emitSSA writes the compiler's own SSA dump for one function.
//
// This is the artifact that explains *why* a spill happened, which the listing
// cannot: it shows the value's live range and the allocator's decision. The gate
// requires it to exist for every audited function, so that a red gate always
// comes with the evidence needed to write docs/spill-report.md.
func emitSSA(pkg, fn, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	// GOSSAFUNC writes ssa.html into the build's working directory, so the build
	// has to run somewhere disposable. That somewhere has to be *inside the
	// module*: `go build` resolves the module from its own working directory even
	// when the package is named by absolute path, so a scratch dir under
	// os.TempDir() fails with "go.mod file not found in current directory or any
	// parent directory". Nesting it under dir (build/ssa, gitignored) keeps the
	// dump out of the repo root and keeps the module in a parent.
	// Absolute, because GOCACHE below is required to be an absolute path and dir
	// arrives from a flag that is naturally written relative (build/ssa).
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(absDir, "scratch-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	abs, err := filepath.Abs(pkg)
	if err != nil {
		return err
	}
	cmd := exec.Command("go", "build", "-o", os.DevNull, abs)
	cmd.Env = append(os.Environ(),
		"GOEXPERIMENT=simd",
		"GOOS=linux",
		"GOARCH=amd64",
		"CGO_ENABLED=0",
		"GOSSAFUNC="+fn,
		// GOSSAFUNC is read by the compiler but is *not* part of the action ID
		// the go command caches on, so an otherwise-identical build is a cache
		// hit, the compiler never runs, and no ssa.html appears — while the
		// cached compiler stderr, including "dumped SSA for <fn> to ./ssa.html",
		// is replayed (docs/toolchain-notes.md T11). GOCACHE *is* in the lookup,
		// so a private one guarantees the compile actually happens. Measured at
		// 0.63s and 18MB per function for ./internal/vec; both are discarded
		// with tmp.
		"GOCACHE="+filepath.Join(tmp, "cache"),
	)
	cmd.Dir = tmp
	var errb bytes.Buffer
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("GOSSAFUNC=%s go build: %v\n%s", fn, err, errb.String())
	}
	// Check for the file rather than trusting the exit status or the message,
	// for the reason in T11: this is what caught the cache hit.
	src := filepath.Join(tmp, "ssa.html")
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("no ssa.html produced for %s (is the name right?): %v", fn, err)
	}
	return os.WriteFile(filepath.Join(dir, fn+".html"), data, 0o644)
}
