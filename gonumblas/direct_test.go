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

func keelStrmm130(a, b []float32) {
	keel.Strmm(keel.Left, keel.Lower, keel.NoTrans, keel.NonUnit, 130, 9, 1.5, a, 130, b, 9)
}

func keelSsyr2k70(a, b, c []float32) {
	keel.Ssyr2k(keel.Upper, keel.NoTrans, 70, 70, 1.5, a, 70, b, 70, 0.5, c, 70)
}

func keelStrmv(n int, a, x []float32) {
	keel.Strmv(keel.Upper, keel.NoTrans, keel.NonUnit, n, a, n, x, 1)
}
func keelStrsv(n int, a, x []float32) {
	keel.Strsv(keel.Lower, keel.NoTrans, keel.NonUnit, n, a, n, x, 1)
}
func keelSsymv(n int, a, x, y []float32) {
	keel.Ssymv(keel.Upper, n, 1.5, a, n, x, 1, 0.5, y, 1)
}

func keelStbmv(n, k int, a, x []float32) {
	keel.Stbmv(keel.Upper, keel.NoTrans, keel.NonUnit, n, k, a, k+1, x, 1)
}
func keelStpmv(n int, ap, x []float32) {
	keel.Stpmv(keel.Upper, keel.NoTrans, keel.NonUnit, n, ap, x, 1)
}
func keelSspmv(n int, ap, x, y []float32) {
	keel.Sspmv(keel.Upper, n, 1.5, ap, x, 1, 0.5, y, 1)
}
func keelSsbmv(n, k int, a, x, y []float32) {
	keel.Ssbmv(keel.Upper, n, k, 1.5, a, k+1, x, 1, 0.5, y, 1)
}
