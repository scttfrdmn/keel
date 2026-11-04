# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# Derives the V3 load probe (archive/v3-load/) mechanically from the shipped KernelPB5x16
# (internal/vec/gemm_neon.go), as archive/neon-raw/probe-gen.py did for raw 3x24/1. Each
# variant takes one class of load out of the K-loop and leaves the loop's bookkeeping alone:
#   Ctl  verbatim;  NA  the 5 A vectors loaded once before the loop;  NB  the 4 B vectors
#   loaded once before the loop;  FO  both (the loop is the 160 FMAs and its bookkeeping).
# None is a GEMM; they reuse operands, so they are timed, never checked.
#   usage: python3 probe-gen.py internal/vec/gemm_neon.go > internal/vec/loadprobe_arm64_test.go
import re, sys
src = open(sys.argv[1]).read()
i = src.index('func KernelPB5x16(')
body = src[i:src.index('\n}\n', i) + 3]
BL = re.compile(r'^\s*b0, b1, b2, b3 = ')
AL = re.compile(r'^(\s*)av = archsimd\.LoadFloat32x4Array\(\(\*\[4\]float32\)\(ap\[(\d+):(\d+)\]\)\)$')
def variant(name, na, nb):
    out = []
    for l in body.split('\n'):
        if l.startswith('func KernelPB5x16('):
            out.append(l.replace('KernelPB5x16', name)); continue
        if l.strip().startswith('for len(ap) >= 160') :
            if nb:
                out.append('\tb0, b1, b2, b3 = archsimd.LoadFloat32x4Array((*[4]float32)(bp[0:4])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[4:8])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[8:12])), archsimd.LoadFloat32x4Array((*[4]float32)(bp[12:16]))')
            if na:
                out.append('\ta0, a1, a2, a3, a4 := ' + ', '.join(f'archsimd.LoadFloat32x4Array((*[4]float32)(ap[{4*r}:{4*r+4}]))' for r in range(5)))
            out.append(l); continue
        if nb and BL.match(l):
            continue
        m = AL.match(l)
        if m and na:
            out.append(f'{m.group(1)}av = a{(int(m.group(2)) // 4) % 5}'); continue
        out.append(l)
    return '\n'.join(out)
doc = {'KernelQ5x16Ctl': 'the control: the shipped KernelPB5x16, verbatim.',
       'KernelQ5x16NA': 'the 5 pre-broadcast A vectors are loaded once before the loop, so the loop has no A loads.',
       'KernelQ5x16NB': 'the 4 B vectors are loaded once before the loop, so the loop has no B loads.',
       'KernelQ5x16FO': 'both hoisted: the loop is the 160 FMAs and its own bookkeeping.'}
print('''// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && arm64

package vec

// The V3 load probe (archive/v3-load/). Each KernelQ5x16* is the shipped KernelPB5x16 with
// one class of load hoisted out of the K-loop, emitted by archive/v3-load/probe-gen.py, so
// timing them side by side attributes the gap between the shipped kernel and the FMA-only
// rate to A loads, B loads, or neither. Test-only: they reuse operands and are not GEMMs.

import (
	"fmt"
	"math/rand"
	"testing"

	"simd/archsimd"
)
''')
for name, (na, nb) in [('KernelQ5x16Ctl', (0, 0)), ('KernelQ5x16NA', (1, 0)), ('KernelQ5x16NB', (0, 1)), ('KernelQ5x16FO', (1, 1))]:
    print(f'// {name} is a load probe (archive/v3-load/): {doc[name]}')
    print(variant(name, na, nb))
print('''var loadProbeSink float32

// BenchmarkLoadProbe times the four variants at three depths, L1- to L2-resident. GFLOP/s
// counts 2·5·16·kc per call for every variant, so each reads as its loop's FMA rate.
func BenchmarkLoadProbe(b *testing.B) {
	const mr, nr = 5, 16
	for _, kc := range []int{128, 384, 512} {
		for _, p := range []struct {
			name string
			f    func(kc int, a, b, c []float32, ldc int)
		}{
			{"Ctl", KernelQ5x16Ctl}, {"NA", KernelQ5x16NA}, {"NB", KernelQ5x16NB}, {"FO", KernelQ5x16FO},
		} {
			b.Run(fmt.Sprintf("%s/kc=%d", p.name, kc), func(b *testing.B) {
				rng := rand.New(rand.NewSource(1))
				ldc := nr + 16
				a, bp, c := make([]float32, 4*kc*mr), make([]float32, kc*nr), make([]float32, mr*ldc)
				for i := range a {
					a[i] = rng.Float32()*2 - 1
				}
				for i := range bp {
					bp[i] = rng.Float32()*2 - 1
				}
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					p.f(kc, a, bp, c, ldc)
				}
				b.StopTimer()
				loadProbeSink = c[0]
				b.ReportMetric(2*mr*nr*float64(kc)*float64(b.N)/b.Elapsed().Seconds()/1e9, "GFLOP/s")
			})
		}
	}
}''')
