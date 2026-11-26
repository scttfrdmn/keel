# Reduces run-eb76812.log: per host x side x threads, the solve time share at each MB and the full-time optimum.
import sys,re,statistics as st
d={}
for l in open(sys.argv[1]):
    m=re.match(r'^keel-(\w+) t(\d) c\d (BenchmarkTrsmMB/\S+)\s+\d+\s+([0-9.]+) ns/op',l)
    if not m: continue
    h,t,name,ns=m.group(1),int(m.group(2)),re.sub(r'-\d+$','',m.group(3)),float(m.group(4))
    p=name.split('/'); side=p[4][5:]; mb=int(p[5][3:]); arm=p[6]
    d.setdefault((h,side,t,mb,arm),[]).append(ns)
hosts=['skx','zen5','zen4','gvt3','gvt4','gvt5']; MBS=[32,48,64,96,128,192,256]
for side in 'LR':
  for t in (1,8):
    print(f"== side={side} threads={t}: solve share at MB=64 | full-time best MB (vs MB=64, median; [range-based bound]) ")
    for h in hosts:
        full={mb:d[(h,side,t,mb,'full')] for mb in MBS}; solve={mb:d[(h,side,t,mb,'solve-only')] for mb in MBS}
        share=st.median(solve[64])/st.median(full[64])
        best=min(MBS,key=lambda mb:st.median(full[mb]))
        g=1-st.median(full[best])/st.median(full[64]); gnet=1-max(full[best])/min(full[64])
        shares=' '.join(f"{mb}:{100*st.median(solve[mb])/st.median(full[mb]):.0f}%" for mb in MBS)
        print(f"  {h:5} share@64 {100*share:5.1f}%  best MB={best:<3} saves {100*g:5.1f}% [net {100*gnet:+5.1f}%]   shares {shares}")

print("\n== full time at each MB relative to MB=64 (median; negative = faster), threads=8 and 1")
for t in (8,1):
  for side in 'LR':
    print(f"side={side} threads={t}   " + "  ".join(f"{mb:>5}" for mb in MBS))
    for h in hosts:
        f64=st.median(d[(h,side,t,64,'full')])
        print(f"  {h:5}           " + "  ".join(f"{100*(st.median(d[(h,side,t,mb,'full')])/f64-1):+5.1f}" for mb in MBS))
