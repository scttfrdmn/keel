import sys,re,statistics as st
d={}
for l in sys.stdin:
    p=l.split(); h,ph,name=p[0][5:],p[1],re.sub(r"-[0-9]+$","",p[2])
    u="Gops/s" if "Gops/s" in p else "GFLOP/s"
    d.setdefault((h,ph,name),[]).append(float(p[p.index(u)-1]))
hosts=["g3a","g3b","g3ea","g3eb","g3hpc"]
def med(h,ph,n): return st.median(d[(h,ph,n)])
PK="BenchmarkPeak/neon"
print("FO peak (core 2) vs loaded cores k; % of k=0")
print("k    "+"  ".join("%15s"%h for h in hosts))
for k in [0,1,3,7,15,31,63]:
    cells=[]
    for h in hosts:
        v=med(h,"curve-k%d"%k,PK); v0=med(h,"curve-k0",PK)
        cells.append("%6.2f (%5.1f%%)"%(v,100*v/v0))
    print("%-4d "%k+"  ".join(cells))
print("range at k=63:", {h:(min(d[(h,"curve-k63",PK)]),max(d[(h,"curve-k63",PK)])) for h in hosts})
print("clock idle0 / k=63 / idle1 / min over all k")
for h in hosts:
    mn=min(min(d[(h,"clock-k%d"%k,"BenchmarkClock")]) for k in [0,1,3,7,15,31,63])
    print("  %-6s %.3f / %.3f / %.3f / %.3f"%(h,med(h,"idle0","BenchmarkClock"),med(h,"clock-k63","BenchmarkClock"),med(h,"idle1","BenchmarkClock"),mn))
print("selectivity, 63 loaders: fo / x44 / shared")
for h in hosts:
    print("  %-6s "%h+" / ".join("%.2f"%med(h,"select63","BenchmarkCeiling/family/%s/threads=1"%x) for x in ["fo","x44","shared"]))
