// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package keel

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/scttfrdmn/keel/internal/block"
)

// Coverage recording for P4's five routines, and the markers scripts/gate-p4.sh
// parses out of the test log.
//
// # Why a recorder instead of printing the lattice
//
// P3's markers restate the sweep's constants: the test declares
// `sweepAlphas = {0, 1, -0.75}` and the marker prints that list. It works because
// Sgemm has three flag dimensions. P4 has five routines carrying side, uplo,
// trans, diag and two strides between them, and each flag selects a different
// code path rather than a different number — so the interesting failure is no
// longer "the sweep is too small", it is "the sweep declares a lattice and skips
// part of it".
//
// So the numbers here are counted, not declared. dim() records the enumerated set
// for one flag; combo() records one tuple that was actually reached, in a set, so
// the count is of distinct combinations run. The gate multiplies the enumerated
// sets and requires the product to equal the count. A `continue` in the middle of
// the sweep, a rotation that quietly collapses a dimension, or a marker that
// claims a dimension it varied at one value all come out as a mismatch — and none
// of them would have made a test fail.
//
// Everything below is written from the sequential sweeps only. No P4 test runs in
// parallel (they reassign package-level dispatch state, like l1_test.go's), so
// the maps need no synchronization.

// p4dim is one flag dimension: the key the gate looks for and the values swept,
// in the order they were declared rather than sorted, so the marker reads like
// the source.
type p4dim struct {
	key  string
	vals []string
}

// p4verify is one size's oracle comparison: how many entries were checked, and
// whether that was every entry of the case or a seeded sample of them.
type p4verify struct {
	exact   bool
	entries int
	seed    int64
}

type p4cov struct {
	routine  string
	dims     []p4dim
	combos   map[string]bool
	sizes    map[int]bool
	verify   map[int]p4verify
	backends map[string]bool
	extras   []string
	config   string
}

var (
	p4covs  = map[string]*p4cov{}
	p4order []string
)

// p4 returns the recorder for one routine, creating it on first use. The order of
// first use is the order the markers print in, which is the order the sweeps run.
func p4(routine string) *p4cov {
	if c, ok := p4covs[routine]; ok {
		return c
	}
	c := &p4cov{
		routine:  routine,
		combos:   map[string]bool{},
		sizes:    map[int]bool{},
		verify:   map[int]p4verify{},
		backends: map[string]bool{},
	}
	p4covs[routine] = c
	p4order = append(p4order, routine)
	return c
}

// dim declares the set swept for one flag. Idempotent: the sweeps declare their
// lattice on every pass through the runner loop, and a redeclaration with
// different values is a bug in the test rather than a coverage fact, so it
// panics instead of quietly widening the lattice the gate then checks against.
func (c *p4cov) dim(key string, vals ...string) {
	for i, d := range c.dims {
		if d.key != key {
			continue
		}
		if strings.Join(d.vals, ",") != strings.Join(vals, ",") {
			panic(fmt.Sprintf("p4 coverage: %s redeclares dimension %s as %v, was %v",
				c.routine, key, vals, d.vals))
		}
		c.dims[i] = p4dim{key, vals}
		return
	}
	c.dims = append(c.dims, p4dim{key, vals})
}

// combo records one flag tuple as having run. The parts are the values in the
// order the dimensions were declared; only their joint identity matters.
func (c *p4cov) combo(parts ...string) { c.combos[strings.Join(parts, "/")] = true }

func (c *p4cov) size(n int)          { c.sizes[n] = true }
func (c *p4cov) backend(name string) { c.backends[name] = true }

// verified records how one size was compared against the oracle, keeping the
// strongest claim made for that size.
//
// Several tests reach the same size — the lattice sweep, the rectangular shapes,
// the multi-block case — and they check different numbers of entries, because a
// 3×16 rectangle and a 16×16 square are both "size 16" to the sizes marker. Taking
// the last call would make the reported count depend on test order, and taking the
// first would report a rectangle's 48 entries for a size the sweep checked all 256
// of. So the maximum wins, with exact breaking a tie: the marker then states a
// lower bound on what was compared, which is the only reading of it that stays true
// as tests are added.
func (c *p4cov) verified(size int, v p4verify) {
	if old, ok := c.verify[size]; ok {
		if old.entries > v.entries || (old.entries == v.entries && old.exact) {
			return
		}
	}
	c.verify[size] = v
}

// exactMode and sampledMode name the two kinds of comparison at the call site, so
// a sweep says which one it did rather than assembling a marker string.
func exactMode(entries int) p4verify { return p4verify{exact: true, entries: entries} }

func sampledMode(entries int, seed int64) p4verify {
	return p4verify{entries: entries, seed: seed}
}

// mode formats one verification for the marker. The shape the gate parses — mode=,
// plus n= and, for a sample, the seed= it can be replayed with — lives here only.
func (v p4verify) mode() string {
	if v.exact {
		return fmt.Sprintf("mode=exact n=%d", v.entries)
	}
	return fmt.Sprintf("mode=sampled n=%d seed=%#x", v.entries, v.seed)
}

// extra records one edge case beyond the lattice, deduplicated: several of them
// run once per backend or per size.
func (c *p4cov) extra(name string) {
	for _, e := range c.extras {
		if e == name {
			return
		}
	}
	c.extras = append(c.extras, name)
}

// l3config records what a derived Level-3 routine inherited from Sgemm. The
// values come from block.Params and the live dispatch rather than from constants
// here: gate-p4.sh criterion 5 compares them against the keel-sgemm-* markers
// printed by the same process, and two independently restated constants would
// agree with each other while both being wrong.
func (c *p4cov) l3config(path string) {
	kc, mc, nc := block.Params(activeKern)
	c.config = fmt.Sprintf("kern=%s kc=%d mc=%d nc=%d path=%s",
		ActiveKernTile()+"/"+ActiveKernBackend(), kc, mc, nc, path)
}

// l2config records what a Level-2 routine's inner loop inherited from P1.
func (c *p4cov) l2config(path string) {
	c.config = fmt.Sprintf("l1=%s path=%s", ActiveL1Backend(), path)
}

// printP4Markers emits what scripts/gate-p4.sh parses. Called from TestMain in
// l1_test.go — one TestMain per package — and unconditionally, so a failing run
// still reports what it covered. A routine whose sweep panicked before recording
// anything prints no marker at all, and the gate treats a missing marker as a
// failure in its own right rather than as coverage it can assume.
func printP4Markers() {
	for _, name := range p4order {
		c := p4covs[name]
		var dims []string
		for _, d := range c.dims {
			dims = append(dims, d.key+"="+strings.Join(d.vals, ","))
		}
		fmt.Printf("keel-p4-lattice: routine=%s %s combos=%d\n",
			c.routine, strings.Join(dims, " "), len(c.combos))

		sizes := make([]int, 0, len(c.sizes))
		for n := range c.sizes {
			sizes = append(sizes, n)
		}
		sort.Ints(sizes)
		strs := make([]string, len(sizes))
		for i, n := range sizes {
			strs[i] = strconv.Itoa(n)
		}
		fmt.Printf("keel-p4-sizes: routine=%s sizes=%s\n", c.routine, strings.Join(strs, ","))
		for _, n := range sizes {
			if v, ok := c.verify[n]; ok {
				fmt.Printf("keel-p4-verify: routine=%s size=%d %s\n", c.routine, n, v.mode())
			}
		}

		bs := make([]string, 0, len(c.backends))
		for b := range c.backends {
			bs = append(bs, b)
		}
		sort.Strings(bs)
		fmt.Printf("keel-p4-backends: routine=%s backends=%s\n", c.routine, strings.Join(bs, ","))
		fmt.Printf("keel-p4-extra: routine=%s extras=%s\n", c.routine, strings.Join(c.extras, ","))
		fmt.Printf("keel-p4-config: routine=%s %s\n", c.routine, c.config)
	}
}

// ------------------------------------------------------------ shared constants

// The P4 size sweep, duplicated in scripts/gate-p4.sh for the reason stated
// there: a test that shrank its own sweep would still report "ok".
//
// 1..17 covers every remainder against both shipped MR values and the Level-1
// kernels' unroll; 31/32/33 and 63/64/65 straddle NR = 32 and two of it, which is
// where a fringe tile and a mask boundary can coincide; 500 is past MC = 144 and
// KC = 384, so the loop nest runs more than one block in every direction.
var p4Sizes = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 31, 32, 33, 63, 64, 65, 500}

const (
	// p4ExactMax is the largest size compared entry by entry.
	p4ExactMax = 65
	// p4SampleN entries per case above p4ExactMax, when a routine samples. The
	// gate demands at least 256.
	p4SampleN = 256
	// p4Seed seeds every data fill and every sampler, so any failure replays.
	p4Seed = 0x7034
)

// The scalar lattices. Each covers both special-cased values and one general
// value, for the reason P3's does: a lattice of {0, 1} tests every shortcut and no
// arithmetic.
var (
	p4Alphas = []float32{0, 1, -0.75}
	p4Betas  = []float32{0, 1, 0.5}
	// The stride lattice: unit (the kernel path), wider than one, and negative
	// (the vector runs backwards from its far end). Reference BLAS defines
	// negative strides for exactly the Level-2 vectors, and offset() is the only
	// place that convention lives.
	p4Incs = []int{1, 2, -1}
)

// f32set and intSet format a lattice for the marker. 'g' with -1 precision so
// -0.75 prints as itself and the gate's numeric comparison sees 0 and 1 exactly.
func f32set(vs []float32) []string {
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = strconv.FormatFloat(float64(v), 'g', -1, 32)
	}
	return out
}

func intSet(vs []int) []string {
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = strconv.Itoa(v)
	}
	return out
}

func f32str(v float32) string { return strconv.FormatFloat(float64(v), 'g', -1, 32) }

// uploFlag, sideFlag and diagFlag are the marker spellings of the three flags,
// and also the values the gate's lattice_req names. One function per flag so the
// spelling cannot drift between the sweep and the marker.
func uploStr(lower bool) string {
	if lower {
		return "L"
	}
	return "U"
}

func sideStr(left bool) string {
	if left {
		return "L"
	}
	return "R"
}

func diagStr(unit bool) string {
	if unit {
		return "U"
	}
	return "N"
}

func transStr(trans bool) string {
	if trans {
		return "T"
	}
	return "N"
}

func uploFlag(lower bool) Uplo {
	if lower {
		return Lower
	}
	return Upper
}

func sideFlag(left bool) Side {
	if left {
		return Left
	}
	return Right
}

func diagFlag(unit bool) Diag {
	if unit {
		return Unit
	}
	return NonUnit
}
