# The re-keying attempt that the witness mechanism correctly refused — 2026-10-05

`42 PASS / 10 FAIL / 1 UNMEASURED / 2 BASELINE` → RED. **I predicted 2 FAILs and got 10.**
The gate was right and the prediction was wrong, for a reason worth keeping.

## What I expected

`fd5195a` keyed `scale/*` and `share/*` on the dispatched shape, as `peak/*` is (#167). I
expected the six rows to render `BASELINE` — first sight under new keys — and the gate to
emit correctly-keyed candidates I could then land.

## What actually happens, and why it is correct

Rule 17(a)'s three states are decided from **tracked state**, and the witness is keyed on
**CPU model + era**, not on the criterion key:

> unregistered with no witness … is genuine newness, so the verdict is `BASELINE`;
> unregistered **with** a witness is not newness but an obligation someone did not land,
> so it is `FAIL` naming the unmet registration

`scripts/judged-runs.tsv` still says this silicon was judged in era `pinned8`. So re-keying
made the *registry* lookup miss while the *witness* stayed, and all six rows became `owing`
— eight FAILs, plus the two stale README rows.

**That is the mechanism working.** A re-key cannot launder a registered obligation into
newness, which is precisely the move it would otherwise permit: change a criterion's key and
every bar you were failing becomes a first sight. I refused that move explicitly on #177 and
then made it structurally an hour later; the witness is what caught me.

## What the fix actually requires

Re-keying mid-era orphans the registry, so the only legitimate path is an **era boundary**,
and §5 rule 17(d) prices it: *"an era exists only via a dated §5 or §7 amendment plus a
both-arms transition archive, recorded per era in `scripts/measurement-eras.tsv`"*, with
each host getting exactly one `BASELINE` per era. That is a contract amendment plus a
transition campaign — not a key string.

So `fd5195a`'s gate change is **reverted**. The finding it was built on stands and is
recorded on #177: the bare `scale/*` and `share/*` keys do carry a bar across kernel
changes, and `scale/Strsm`'s registered 7.921×/7.902× were measured at `8x8/neon`. Fixing it
needs the era boundary, deliberately, with the amendment it requires.
