// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package block

import (
	"testing"

	"github.com/scttfrdmn/keel/internal/kern"
)

// TestPlanKCDeep pins KCDeep's shape rule: a full-height call (m and n both at least one MC
// block) gets KCDeep; the thin rank updates Trsm issues (MB rows on the left, MB columns on
// the right) and pre-broadcast kernels keep KC; and KCDeep = 0 changes nothing.
func TestPlanKCDeep(t *testing.T) {
	defer func(kc, kd int) { KC, KCDeep = kc, kd }(KC, KCDeep)
	KC = 384
	kn := kern.Kernel{Name: "test", MR: 2, NR: 32}
	pb := kern.Kernel{Name: "test", MR: 5, NR: 16, ALanes: 4}
	cases := []struct {
		name    string
		kn      kern.Kernel
		deep    int
		m, n, k int
		want    int
	}{
		{"off: full-height keeps KC", kn, 0, 4096, 4096, 4096, 384},
		{"on: full-height takes KCDeep", kn, 1024, 4096, 4096, 4096, 1024},
		{"on: left Trsm rank update (MB rows) keeps KC", kn, 1024, MB, 4096, 4096, 384},
		{"on: right Trsm rank update (MB columns) keeps KC", kn, 1024, 4096, MB, 4096, 384},
		{"on: KCDeep still clamps to k", kn, 1024, 4096, 4096, 700, 700},
		{"on: pre-broadcast kernels keep their own KC", pb, 1024, 4096, 4096, 4096, KCPB},
	}
	for _, c := range cases {
		KCDeep = c.deep
		if kc, _, _ := plan(c.kn, c.m, c.n, c.k); kc != c.want {
			t.Errorf("%s: plan kc = %d, want %d", c.name, kc, c.want)
		}
	}
}
