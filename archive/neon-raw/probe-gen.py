import re,sys
src=open(sys.argv[1]).read()
body=src[src.index('func KernelR3x24U1'):]
def variant(name, pb, nb, fo):
    s=body.replace('KernelR3x24U1', name)
    lines=s.split('\n'); out=[]
    bload=[l for l in lines if l.strip().startswith('b0, b1, b2, b3, b4, b5 =')][0]
    for l in lines:
        t=l.strip()
        if t.startswith('ap := a[:kc*3]') and (pb):
            out.append('\tap := a[:kc*12] // A pre-broadcast: 4 copies of each element'); continue
        if t.startswith('for len(ap) >= 3'):
            if nb:  # B loaded once, outside the loop
                out.append(bload.replace('\t\t','\t',1))
            if fo:
                out.append('\ta0, a1, a2 := archsimd.BroadcastFloat32x4(ap[0]), archsimd.BroadcastFloat32x4(ap[1]), archsimd.BroadcastFloat32x4(ap[2])')
            out.append(l.replace('len(ap) >= 3', 'len(ap) >= 12') if pb else l); continue
        if t.startswith('b0, b1, b2, b3, b4, b5 =') and nb: continue
        m=re.match(r'\s*av = archsimd.BroadcastFloat32x4\(ap\[(\d)\]\)', l)
        if m:
            r=int(m.group(1))
            if fo: out.append(f'\t\tav = a{r}')
            elif pb: out.append(f'\t\tav = archsimd.LoadFloat32x4Array((*[4]float32)(ap[{4*r}:{4*r+4}]))')
            else: out.append(l)
            continue
        if t.startswith('ap, bp = ap[3:]') and pb:
            out.append(l.replace('ap[3:]','ap[12:]')); continue
        out.append(l)
    return '\n'.join(out)
doc={'KernelP3x24Ctl':'the control: raw 3x24/1 as emitted.',
 'KernelP3x24PB':'A arrives pre-broadcast, one 128-bit load per row instead of an element load, insert and VDUP.',
 'KernelP3x24NB':'B is loaded once before the loop, so the loop has no B loads.',
 'KernelP3x24PBNB':'both: pre-broadcast A loads and no B loads.',
 'KernelP3x24FO':'FMA only: A broadcasts and B loads both hoisted; the loop is the 18 FMAs and its own bookkeeping.'}
print('''// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

//go:build goexperiment.simd && arm64

package PKG

import "simd/archsimd"
''')
for name,(pb,nb,fo) in [('KernelP3x24Ctl',(0,0,0)),('KernelP3x24PB',(1,0,0)),('KernelP3x24NB',(0,1,0)),('KernelP3x24PBNB',(1,1,0)),('KernelP3x24FO',(0,1,1))]:
    print(f'// {name} is a limiter probe (#191): {doc[name]}')
    print(variant(name,pb,nb,fo))
