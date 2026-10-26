import re,statistics as st,math,collections
R=collections.defaultdict(list)
for l in open('sweep.full'):
    m=re.match(r'p\d (Benchmark\S+?)(?:-\d+)?\s+\d+\s+\d+ ns/op\s+([\d.]+) GFLOP/s',l)
    if m: R[m.group(1)].append(float(m.group(2)))
def stats(k): v=R[k]; return st.median(v),min(v),max(v),len(v)
kern=['2x16/avx512','4x16/avx512']
# Blocking
inc='kc=384/mc=144/nc=2048'
cfgs=sorted({k.split('/',3)[3] for k in R if k.startswith('BenchmarkBlocking64/')})
rows=[]
for c in cfgs:
    ok=True; meds=[]
    for kn in kern:
        a=f'BenchmarkBlocking64/{kn}/{c}'; b=f'BenchmarkBlocking64/{kn}/{inc}'
        if a not in R: ok=False; break
        am,amin,amax,_=stats(a); bm,bmin,bmax,_=stats(b); meds.append(am)
        if not (am>bm and amin>bmax): ok=False
    if len(meds)==2: rows.append((math.sqrt(meds[0]*meds[1]),c,ok,meds))
rows.sort(reverse=True)
for kn in kern:
    m,lo,hi,n=stats(f'BenchmarkBlocking64/{kn}/{inc}'); print(f"incumbent {kn} {inc}: median {m:.2f} [{lo:.2f},{hi:.2f}] n={n}")
print("top 8 by geomean:")
for g,c,ok,meds in rows[:8]:
    print(f"  {c:28s} geo {g:.2f}  2x16 {meds[0]:.2f}  4x16 {meds[1]:.2f}  qualifies={ok}")
q=[r for r in rows if r[2]]
print("QUALIFYING:", len(q), (q[0][1] if q else "none -> keep incumbent"))
# MB
print("Trsm MB:")
for kn in kern:
  for side in 'LR':
    line=[]
    for mb in [32,48,64,96,128]:
        k=f'BenchmarkTrsmMB64/{kn}/side={side}/mb={mb}'
        if k in R: m,lo,hi,_=stats(k); line.append(f"mb={mb}:{m:.2f}[{lo:.1f},{hi:.1f}]")
    print(f"  {kn} {side}: "+"  ".join(line))
mbq=[]
for mb in [32,48,96,128]:
    ok=True
    for kn in kern:
        for side in 'LR':
            a=f'BenchmarkTrsmMB64/{kn}/side={side}/mb={mb}'; b=f'BenchmarkTrsmMB64/{kn}/side={side}/mb=64'
            am,amin,amax,_=stats(a); bm,bmin,bmax,_=stats(b)
            if not (am>bm and amin>bmax): ok=False
    if ok: mbq.append(mb)
print("MB QUALIFYING:", mbq or "none -> keep 64")
