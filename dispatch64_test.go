// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package keel

import (
	"testing"

	"github.com/scttfrdmn/keel/internal/kern"
)

// TestKernBackendResolvesKEEL_FORCE pins the shared resolver both microkernel
// selections go through: unset picks the widest backend, a present name is
// honoured, and a name with no microkernel (avx2) is a ceiling at scalar.
func TestKernBackendResolvesKEEL_FORCE(t *testing.T) {
	names := []string{kern.AVX512, kern.AVX512, kern.Scalar, kern.Scalar}
	for _, c := range []struct{ force, want string }{
		{"", kern.AVX512},
		{kern.Scalar, kern.Scalar},
		{kern.AVX512, kern.AVX512},
		{"avx2", kern.Scalar},
	} {
		t.Setenv(envForce, c.force)
		if got := kernBackend(names); got != c.want {
			t.Errorf("KEEL_FORCE=%q: backend %q, want %q", c.force, got, c.want)
		}
	}
}

// TestSelectKern64 holds the float64 selection to the float32 one's rules: under
// KEEL_FORCE=scalar it runs scalar; otherwise, where a float64 vector kernel
// exists, the backend matches the float32 Level-3 backend and the shape is the
// one Preferred64 picks for the host's class.
func TestSelectKern64(t *testing.T) {
	t.Setenv(envForce, kern.Scalar)
	if k := selectKern64(); k.Name != kern.Scalar {
		t.Errorf("KEEL_FORCE=scalar selected %s", k.ID())
	}
	t.Setenv(envForce, "")
	k := selectKern64()
	var vec []kern.Kernel64
	for _, c := range kern.Kernels64() {
		if c.Name != kern.Scalar {
			vec = append(vec, c)
		}
	}
	if len(vec) == 0 {
		if k.Name != kern.Scalar {
			t.Errorf("no float64 vector kernel on this build, but %s was selected", k.ID())
		}
		t.Logf("float64 dispatch: scalar (%s); no float64 vector kernel on this build", k.ID())
		return
	}
	want, _ := kern.Preferred64(activeKernClass, vec)
	if k.ID() != want.ID() {
		t.Errorf("selected %s, Preferred64 for class %s picks %s", k.ID(), activeKernClass, want.ID())
	}
	if k32 := selectKern(); k32.Name != k.Name {
		t.Errorf("float32 Level 3 dispatches %s but float64 dispatches %s on the same host", k32.Name, k.Name)
	}
	t.Logf("float64 dispatch: %s on a %s host", k.ID(), activeKernClass)
}
