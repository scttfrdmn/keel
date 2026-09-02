// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

// Independent witness for analyze147.py's pretty(): testing.prettyPrint is unexported,
// but BenchmarkResult.String() calls it, so Go itself can render the same values and the
// transcription is checked against the toolchain rather than against a second copy of my
// own reading. Prints "n elapsed rendered" for values straddling every format boundary.
package main

import (
	"fmt"
	"testing"
	"time"
)

func main() {
	// (n, elapsed_ns) pairs: each targets a boundary in prettyPrint's switch, plus the
	// 19845.4 case that broke the 4-sig-fig comparison on real data.
	cases := [][2]int64{
		{1, 0}, {10, 9994}, {10, 9995}, {10, 9999}, {10, 99994}, {10, 99995},
		{10, 999949}, {10, 999950}, {10, 9999499}, {10, 9999500},
		{5, 99227}, {225726, 4478400682}, {225445, 4319726863}, {10000, 1612530000},
		{1000000, 1328000000}, {30, 501}, {30, 3}, {1000, 7}, {100000, 3},
	}
	for _, c := range cases {
		r := testing.BenchmarkResult{N: int(c[0]), T: time.Duration(c[1])}
		fmt.Printf("%d %d %s\n", c[0], c[1], r.String())
	}
}
