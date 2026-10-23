// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

package gonumblas

import "github.com/scttfrdmn/keel"

// Direct keel calls for TestRoutesToKeel, written against keel's own API so they
// share no code path with the adapter under test.

func keelSdot(n int, x, y []float32) float32 { return keel.Sdot(n, x, 1, y, 1) }
func keelSasum(n int, x []float32) float32   { return keel.Sasum(n, x, 1) }
func keelSnrm2(n int, x []float32) float32   { return keel.Snrm2(n, x, 1) }
func keelSgemv(a, x, y []float32)            { keel.Sgemv(keel.NoTrans, 33, 33, 1.5, a, 33, x, 1, 0.5, y, 1) }
func keelSgemm(a, b, c []float32) {
	keel.Sgemm(keel.NoTrans, keel.Trans, 33, 33, 33, 1.5, a, 33, b, 33, 0.5, c, 33)
}
func keelSsyrk(a, c []float32) { keel.Ssyrk(keel.Upper, keel.NoTrans, 33, 33, 1.5, a, 33, 0.5, c, 33) }
func keelSsymm(a, b, c []float32) {
	keel.Ssymm(keel.Left, keel.Upper, 33, 33, 1.5, a, 33, b, 33, 0.5, c, 33)
}
func keelStrsm130(a, c []float32) {
	keel.Strsm(keel.Left, keel.Lower, keel.NoTrans, keel.NonUnit, 130, 7, 1.5, a, 130, c, 7)
}
