# Execution hosts

keel is developed on `darwin/arm64`, where `simd/archsimd` does not exist
(docs/toolchain-notes.md T1). Anything the *compiler* decides is checked
locally by cross-compiling; anything the *CPU* decides has to run on amd64
hardware. This file records how that happens and on which machines.

## How remote execution works

`scripts/remote.sh` cross-compiles a package's test binary for `linux/amd64`
with `go test -c`, ships it over `scp`, and runs it. Two consequences worth
stating plainly:

- **Almost no toolchain is needed on the remote host.** `CGO_ENABLED=0` makes a
  static, pure-Go ELF binary, so for every criterion but one the host needs
  nothing but `sshd`, and the compiler in every published denominator is the dev
  host's. *Corrected 2026-08-28*, twice over, because this read "**No toolchain is
  installed on the remote host**" and ended "the compiler whose output gets
  executed is the same one the gate just version-checked, so there is no second
  toolchain to drift":
  - There **is** a second toolchain, and gate-p5's `-race` arm requires it.
    `go test -c -race` under `CGO_ENABLED=0` is refused outright — `go: -race
    requires cgo` — so that one arm ships `git archive HEAD` and builds on the
    host. `janus` and `antares` both carry `/usr/local/go` = `go1.27.0` as of
    2026-08-28, installed under #70's ruling with `go1.27rc3` *removed* rather than
    left beside it: an inadmissible toolchain on disk is a wrong-binary selection
    waiting for a PATH ambiguity, and since T23's rename 1.26 cannot compile this
    tree at all. `vesta` carries it too as of 2026-08-29 — same digest against
    `$KEEL_GO_SHA256`, `go version go1.27.0 linux/amd64` read back off the host,
    and corroborated independently by `gate-p5`'s own provenance stamp on all
    three `-race` rows. *This sentence said `vesta` was `unmeasured` because it
    "answered neither `vesta` nor `vesta.local`", and that was never what
    happened*: `provision-vesta-b5cef4f.log` reached `vesta.local` on the first
    try and read its state correctly (`go=none … governor=powersave`), then hit
    `confirm()` with no answerable terminal, because the run was detached. A
    launcher that cannot obtain consent reported as a host that cannot be
    reached — the instrument speaking in the subject's voice, and the reason the
    fix was `--yes`, not a network change. Detached provisioning needs `--yes`:
    `detach.sh` gives the run a tmux pane, so `confirm()`'s `/dev/tty` open
    *succeeds* and its `read` then hits EOF, which is the one branch whose
    message names a terminal rather than the absence of one.
    `janus` additionally keeps `~/sdk/go1.25.0` and `~/sdk/go1.26.5`, recorded
    rather than removed because the ruling named rc3. *This said they were
    "reachable only through their own `go1.X` shims"; re-probed 2026-08-31 (#134),
    there are no shims* — `~/go/bin/go1.*` is empty on all three hosts, and no
    dotfile, crontab or user unit names `sdk/go1`, so nothing but an absolute path
    reaches them. **The auto-selectable toolchains are somewhere else entirely.**
    `GOTOOLCHAIN` is `auto` on all three, which selects out of
    `$GOMODCACHE/golang.org/toolchain@*`, and `janus` holds five there — go1.24.0,
    go1.25.0, go1.25.11, go1.25.12, go1.27.0, four of them below the go1.27 floor.
    Inert for keel, whose `go.mod` asks for `go 1.26` against an installed 1.27.0,
    so `auto` has nothing to switch to; the policy question is #121's.
  - The drift argument was backwards. The version the gate printed was the
    **host's** (`gate-p5.sh:639`), which is the compiler that produces *nothing*
    except that `-race` arm — so the one binary whose provenance mattered was the
    one no check read. `builder_toolchain` (`scripts/remote.sh`, `b0e5b37`) is what
    makes the sentence's premise true for the first time: it reads `go version
    <ELF>` off the cross-compiled artifact, reporting compiler *and* GOEXPERIMENT
    (`go1.27.0-X:simd`), and prints on change so a mid-run move announces itself.
- **It works for benchmarks too.** `go test -c` includes `Benchmark*`, so
  P1's ≥4× check and P2's percent-of-peak measurement run the same way. What
  cannot cross the wire is anything needing local `perf`/`ssa.html`
  inspection, which is a compile-time artifact and stays local anyway.

Configure targets with `.keel-hosts` at the repo root (one host per line,
gitignored — see `.keel-hosts.example`) or `$KEEL_REMOTE_HOSTS`, which takes
precedence. Real hostnames are infrastructure, not source, so they are not
checked in.

**Name a LAN host by its `.local` form, always, and never by the bare name.**
The two are not synonyms here: measured 2026-08-29, every lab host resolves the
bare name to a Tailscale address and the `.local` name to a LAN one — `vesta`
→ `100.82.237.84`, `vesta.local` → `192.168.6.153`, and likewise `janus`
(`100.89.76.28` / `192.168.6.180`) and `antares` (`100.107.102.112` /
`192.168.6.176`). Two consequences, and the second is the one that costs
something:

- The trust state differs per form. `vesta`'s reimage invalidated the key
  `known_hosts` holds for its Tailscale address, so the bare form warns
  `REMOTE HOST IDENTIFICATION HAS CHANGED` while `.local` connects clean. It
  degrades to a warning only because pubkey auth still succeeds; under a
  stricter `StrictHostKeyChecking` the same host is up and refusing.
- **The form leaks into provenance, so one machine becomes two.** The hostname
  is column 5 of a witness row and is interpolated into the archive filename,
  and `build/witness-candidates-b5cef4f.tsv` holds proof: rows 2 and 4 are the
  same Ryzen AI MAX+ 395 under `antares` and `antares.local`, pointing at
  `bench-gate-p5-…-antares-…` and `bench-gate-p5-…-antares.local-…`. The
  registry key `(cpu_model, era)` is what saves this from being a correctness
  bug — it is the same on both rows, by design (`host-baselines.tsv`: the
  hostname is "provenance, never a key") — but a reader reconciling archives by
  host sees a fleet with one more member than it has.

## Requirements for a target

| Requirement | Why |
|---|---|
| amd64 | **keel's** constraint, not the toolchain's — corrected 2026-08-29, having read `amd64-only` since it was written against go1.26.5. On go1.27.0 `$GOROOT/src/simd/archsimd` ships 9 arm64 files (`ops_arm64.go`, `types_arm64.go`, `slice_gen_arm64.go`, …) and the portable `simd` package two more, all tagged `goexperiment.simd` alone. What is amd64-only is every backend in `internal/vec` (`//go:build goexperiment.simd && amd64`, 6 files, plus the `!amd64` scalar fallbacks). So a NEON port is a kernel this repo has not written, not a facility the experiment withholds — and the *blocking* half is neither: every judged bar (`CEIL_FRACTION`, `STRSM_FLOOR`, `SYRK_FLOOR`) was derived on amd64, so an arm64 host would be judged wholly outside its derivation set on every bar at once. DESIGN.md §5 rule 17 is the rule that already forbids this for the *share* criteria, host by host; extending it to the *ratio* criteria is legislated post-tag, from principle, and arm64 is gated behind that extension rather than behind a NEON kernel |
| AVX2 + FMA | the AVX2 backend; `archsimd.X86.AVX2() && .FMA()` |
| AVX-512 F/CD/BW/DQ/VL | the bundle `archsimd.X86.AVX512()` gates on — all five, or the backend does not register |
| key-based ssh from the dev host | `remote.sh` uses `BatchMode=yes`; it never prompts and never handles credentials |
| a clock established stable | for gate numbers, not correctness. DESIGN.md §5 rule 5 (as amended 2026-08-16) requires it of every measuring host, by whichever instrument the host has: the `performance` governor where `cpufreq` is readable, else `BenchmarkPeak` sampled at head/middle/tail. Both branches are live — `clock_gate`/`clock_head`/`clock_post`, and the guest branch is what the AWS fleet runs on, where `assert_governor` reads `absent` |
| an admission class | `evidentiary` (whole-socket, judged) or `correctness` (partial-size or unproven, reported); see the class table below. Unreadable is `unmeasured`, not `correctness`. Read by `host_admission` from the provenance line's `instance=`, `virt=`, `spawn=` and `governor=` fields — an approved type in `KEEL_EVIDENTIARY_SIZES` **that the launcher independently confirms, on-demand**, *or* bare metal (`virt=metal`, no `hypervisor` CPU flag) with `governor=performance` (`remote.sh`, #106, and the 2026-08-19 ruling); wired into every judged perf criterion: gate-p2's 5b, gate-p3's two, and gate-p5's scaling criterion. **Deliberately not** gate-p4's criterion 7 — see the scope note below |

**And two on the driver, which is a different list because they are not properties of the
machine being measured.** `spawn` on `$PATH` with `AWS_PROFILE=aws` credentials, and `jq` —
the only two things `spawn_probe` needs to produce the `spawn=` field admission now requires.
Absence of either is **not** an error: the field reads `?`, which withholds admission and
prints the reason, so a driver without them can still run every correctness gate and every
lab-fleet judged gate (bare metal takes the other route to the class, and never consults the
launcher). What it cannot do is judge a cloud host. `jq` is a soft dependency and the only
one in this repo; nothing else here parses JSON.

## Current targets

All three were verified on 2026-08-10 and all three run all three backends.

**Two tiers as of 2026-08-19, and the three hosts below are the dev tier.** The *judged*
tier — the one a published row comes from — is the AWS fleet: full-size on-demand instances
launched by `truffle`/`spawn`, because they supply **reproducibility by strangers**. Anyone
can rent a `c7a.48xlarge`; nobody can rent this lab. The lab keeps two things the cloud
cannot supply, which is why it is reclassified and not retired: real levers (boost-off
iso-frequency arms, governor control — #105) and **unrentable silicon**. antares in
particular survives as a **marked consumer-silicon row**, since consumer and mobile Zen 5 is
exactly what keel's Go audience runs. Lab rows stay publishable *labelled as what they are*
and never mixed into the citable set. These hosts still reach the `evidentiary` class, by
the bare-metal arm; the tier split is about which rows get cited, not about which gates run.

### Private cache sizes (read from sysfs 2026-08-12, not looked up)

| Host | µarch | L1d | L2 | L3 |
|---|---|---|---|---|
| vesta | Zen 4 | 32K | 1024K | 98304K (V-Cache) |
| janus | Skylake-X | 32K | 1024K | 22528K |
| antares | Zen 5 | **48K** | 1024K | 32768K |

This table is here because a reading of the #48 feed decomposition turned out to
need it, and getting it from memory would have been wrong: **Zen 5 has a 48 KB L1d
where Zen 4 and Skylake-X have 32 KB.** `remote_probe` now reports these from
`/sys/devices/system/cpu/cpu0/cache/index*` on every run, so the numbers arrive with
the measurement instead of being asserted next to it.

What they are used for: three of `BenchmarkFeed`'s arms reuse one packed panel pair
of `(MR+NR)·kk·4` bytes, which at NR=32 is 17/34/51/68 KB for `kc=128/256/384/512`
(2×32) and 18/36/54/72 KB (4×32). So the "reused pair" is L1-resident only at
`kc=128` on vesta and janus, and at `kc=128` *and* `256` on antares — and above that
threshold both panel arms feed from L2, where all four sizes fit on all three hosts.
The step between them is then a difference of locality *within* L2 rather than a
difference of level, which is a different measurement from the one the arm's name
suggests. The threshold moves with the host, which is exactly why the number has to
come from the host.

### Zen 4 — AMD Ryzen 9 7950X3D, 16C/32T, Linux 6.17, `performance`

The primary *development* target (the judged tier is the cloud fleet — see above).
DESIGN.md §4/P3 sizes its initial blocking parameters for
"a Zen4/Ice Lake-class target", so this is the machine those numbers are aimed
at. AVX-512 feature set is essentially complete for Zen 4 (F, DQ, IFMA, CD,
BW, VL, VBMI, VBMI2, VNNI, BITALG, VPOPCNTDQ, BF16, plus GFNI/VAES/VPCLMULQDQ).

keel uses none of that beyond F/DQ/CD/BW/VL. VNNI (int8 dot product) and BF16
are for quantized and reduced-precision inference, which DESIGN.md §2 lists as
non-goals for v0; they are noted here only so a future parking-lot item knows
the capability exists on this machine and not the other.

**The double-pumped datapath, measured rather than looked up.** Zen 4
implements AVX-512 on 256-bit datapaths: a 512-bit FMA is split into two
256-bit µops, so a core retires one 512-bit FMA per cycle, not two. That is now
this project's own measurement rather than a citation — `BenchmarkPeak` on
2026-08-11 returned **165.6 GFLOP/s at 512 bits and 165.5 at 256**, a width
ratio of **1.00×**. Two vector widths with identical float32 ceilings is what a
double-pumped datapath looks like from software, and it needs no clock estimate
or port count to interpret.

DESIGN.md §4/P2's original peak formula (`freq · 2 FMA ports · 16 lanes · 2
flops`) predicts 368.9 GFLOP/s here, **2.23× the measured ceiling**, which
would have turned the 55%-of-peak floor into a demand for ~123% of what this
hardware can do. That is why peak is now measured per host and the formula kept
only as a printed cross-check (issue #11).

A corollary that matters for P2's spill audit: on Zen 4, AVX-512 and AVX2 have
the *same* float32 FLOP ceiling. AVX-512's advantage there is 32 architectural
vector registers instead of 16, plus masking and fewer instructions per unit of
work — which is exactly the currency the spill audit deals in, so a Zen 4
AVX-512 kernel can still beat its AVX2 twin without either exceeding a shared
peak.

### Skylake-X — Intel Core i9-9960X, 16C/32T, RHEL 9, `performance`

The second data point, and a genuinely different machine rather than a
duplicate:

- True 512-bit datapath with two 512-bit FMA units — **measured width ratio
  2.12×** (215.9 GFLOP/s at 512 bits, 101.8 at 256), the opposite shape from
  vesta and the reason having both machines settles the question that either one
  alone could not.
- The narrow AVX-512 tier — F, DQ, CD, BW, VL only. Since that is exactly the
  bundle `archsimd.X86.AVX512()` requires, it is the floor of what keel's
  AVX-512 backend may assume. If a future kernel reaches for VBMI2 or VNNI, it
  fails here first, which is the point.
- Heavy AVX-512 code triggers a substantial frequency license drop on this
  microarchitecture, and the peak measurement shows it: 215.9 GFLOP/s at 512
  bits implies **3.37 GHz sustained** under an FMA-saturated load, against the
  4.4 GHz `cpuinfo_max_freq` the formula uses — which accounts for essentially
  all of the formula's 1.30× overstatement here. Nominal clock is the wrong
  denominator on this machine; measuring the ceiling directly sidesteps the
  question, which is the second independent reason peak is measured (issue #11).

### Zen 5 — AMD Ryzen AI MAX+ 395 (Strix Halo), 16C/32T, Linux 7.0, `performance`

The newest microarchitecture available, and the widest AVX-512 feature set of
the three: F, DQ, CD, BW, VL, IFMA, VBMI, VBMI2, VNNI, BITALG, VPOPCNTDQ, BF16,
and `avx512_vp2intersect` — the last being genuinely unusual, since Intel
shipped it on Tiger Lake and then dropped it. keel uses none of it beyond
F/DQ/CD/BW/VL; it is recorded because it makes this the second machine (with
vesta) that could host any future reduced-precision parking-lot work.

- **Full-width 512-bit datapath, and the fastest of the three.** AMD has
  shipped Zen 5 in both configurations — full 512-bit on some parts, 256-bit
  double-pumped on others — and which one an APU of this class uses was exactly
  the kind of detail that is easy to assert confidently and wrongly, so it was
  measured: **327.8 GFLOP/s at 512 bits against 164.0 at 256, a width ratio of
  2.00×**. This is a full-width part, and its measured ceiling is 1.98× vesta's
  despite being a mobile APU. The formula agrees here to within **1.01×**, which
  is worth noting precisely because it is the exception: the same formula is
  2.23× high on vesta and 1.30× high on janus.
- **The 43% confidence interval on this host has no explanation on record, and
  that is the honest state of it** (issue #44). On 2026-08-11 this machine was in
  `powersave` while the other two were in `performance`, and one gate run reported
  the scalar Sdot median with a **43% confidence interval** (point-estimate speedup
  16.03×, against 8.96× the next run). The governor was the obvious suspect and was
  written here as the cause. It has since been set to `performance` during the
  OpenBLAS provisioning campaign — every gate run from 2026-08-12 reads
  `governor=performance` from the machine itself — which removes the suspect
  without settling anything: the variance was measured under the old setting and
  has not been re-measured under the new one.

  **Re-measured under the asserted governor on 2026-08-12, and the governor was not
  the whole story.** Three independent runs of the same benchmark under the same
  methodology (`GateSdot` at n=4096, `-count=10 -benchtime=1s`, benchstat median):

  | run | scalar median | avx512 median | ratio | net of CI |
  |---|---|---|---|---|
  | 1 | 9.437e-07 s ± 8.0% | 9.505e-08 s ± 0.0% | 9.93× | 9.13× |
  | 2 | 1.039e-06 s ± 14.0% | 9.514e-08 s ± 1.0% | 10.92× | 9.29× |
  | 3 | 1.017e-06 s ± 3.0% | 9.519e-08 s ± 0.0% | 10.69× | 10.37× |

  Two readings, and both belong here:

  - **The 43% interval did not reproduce.** The widest scalar interval in three runs
    was 14%. That does not *exclude* a 43% outlier — three draws cannot rule out a
    rare event, and the original 43% was itself one run in three — so the honest form
    is a bound, not an absolution: under `performance`, three runs produced ≤14%.
  - **A scalar-path instability remains, and the location is established.** The
    scalar median's interval ranged 3–14% while the AVX-512 median's ranged 0–1% in
    the same runs, on the same host, minutes apart. That comparison is
    *differential* — two kernels co-measured under identical conditions — so an
    order of magnitude between them is a property of one kernel rather than ambient
    machine noise. The governor therefore cannot be credited with having caused what
    was never shown to be caused by it, and the 43% has **no identified cause**: it
    has a bound and a location.

  **Location identified, mechanism open, and the difference matters.** "The kernel
  that touches memory" is the tempting phrase and it does not discriminate: both
  kernels read the same two arrays over the same 32 KB working set. What actually
  differs between them is runtime per op (~10×), frequency sensitivity and issue
  character. So there are two candidate classes, and neither is favoured by the data
  in hand:

  1. **Clock-domain exposure.** A scalar loop spanning ten times the wall time per
     measurement samples ten times more boost and thermal wander, which on a mobile
     APU under a shared power budget is at least as available a mechanism as anything
     in the cache hierarchy.
  2. **Memory-path behaviour** at that working set — the reading the phrase above
     assumes, which would need evidence the two kernels' memory traffic differs in a
     way their identical arrays do not already fix.

  Both untested. This is the same *character* of finding as vesta's bimodal Sdot
  ratio and lives in the same place: issue #15, the run-to-run instability and
  pinning decision, which now carries both hosts' data. Keeping the mechanism open
  is the same discipline that reopened the 43% in the first place — a plausible
  phrase is not an explanation.

  What is not in question: the register-only peak measurement was unaffected (0.4%
  across three runs), so whatever this is, it appears only once a benchmark touches
  memory. The net-of-CI rule absorbed it correctly either way, reporting 9.13×,
  9.29× and 10.37× across the three runs above, which is the rule doing its job
  rather than a reason to trust the point estimate.

  > **Correction, 2026-08-16 (#79).** This paragraph previously read "reporting
  > 9.14×, 8.96× and 8.79× across three runs". That trio is not from these runs: it
  > was published on 2026-08-11 by `4643b63`, *before* the governor fix, and the
  > three runs tabled immediately above report 9.13× / 9.29× / 10.37×. The
  > misattribution also reversed the paragraph's own point — the trio it quoted
  > spans 4.0%, the runs it was describing span 13.6%, so it cited a tight spread as
  > evidence while the data in front of it was the loose one. The 2026-08-11 trio has
  > no surviving derivation at all (see the note at the summary table below); the
  > figures here are the ones this section actually measured.

Also worth noting for P3 rather than P2: this is an APU with unified LPDDR5X
rather than separate DIMM channels, so its bandwidth and cache hierarchy differ
structurally from the two desktop/HEDT parts. DESIGN.md §4/P3 sizes KC/MC/NC
against a cache hierarchy, so the blocking parameters that suit vesta should not
be assumed to transfer here. Measure per host.

## Two admission classes (ruled 2026-08-12 on #12; amended 2026-08-17 on #104 and 2026-08-19 on #106)

*Headed "Cloud hosts" until 2026-08-19. The scope was never the cloud — it is which hosts
may sign a perf verdict — and the misnomer is exactly how bare metal came to be classified
by a fallthrough written for guests.*

Two classes, and the distinction is what each one is allowed to produce:

| class | machines | produces | admission requires |
|---|---|---|---|
| **evidentiary** | a **full-size (whole-socket)** instance of an approved family **that `truffle`/`spawn` launched on-demand**, **or** a bare-metal host | judged perf verdicts; published rows; the stage-3 curves | whole-socket ownership **and** a passing preamble: clock stability established by §5 rule 5's instrument for the host it is. Two routes to the former, and the class names each — an approved instance type the launcher independently confirms, or `virt=metal` with `governor=performance` |
| **correctness** | any partial-size guest, any µarch; any host whose socket ownership is unproven | differential and correctness coverage; perf numbers **reported, never judged** | nothing beyond reachability |

**Amended 2026-08-19 (Scott's judged-tier directive): the launcher is a second witness, and
an approved type alone no longer admits.** Every other field in a provenance line is *the
host describing itself*, which is the circularity the rejected `assume_fleet` design was
refused for — every witness of a host's identity came from the host. `instance=` narrows it
(169.254.169.254 is the control plane answering, not the guest) but it is still a reading
taken inside the guest over a link the guest could serve itself. `spawn=` is read on the
**driver's** side of the wire, from the EC2 control plane under the driver's own
credentials, so it is the one statement about a host's identity the host cannot author.
Admission now needs both, and adds one conjunct the launcher alone can supply:

| `spawn=` | class | why |
|---|---|---|
| `id:type:ondemand`, type agreeing with `instance=` | **evidentiary** | two independent witnesses, and the market a judged run requires |
| `id:type:spot` | correctness | a reclaim mid-sweep converts a judged reading into a truncated one. **Interruption is the whole reason and cost is not** — spot is the exploration tier |
| type **disagreeing** with `instance=` | **`unknown` ⇒ unmeasured** | the identity is in dispute. Grading it `correctness` would report "not full size" when what is broken is the instrument |
| `none` | correctness | the launcher has no running record under this name, so the size rests on the guest's own testimony |
| `ambiguous` | correctness | two records answer to this name; a join that picked one would attribute a reading to a machine that may not have produced it |
| `?` | correctness | the launcher could not be consulted (no `spawn`, no `jq`, or no credentials) — unread is unmet |
| absent | correctness | a provenance line from before the launcher was a witness. Told apart from `?` because the remedies differ: upgrade the driver, versus install `jq` |

The market conjunct is a *criterion that became readable*, not a new rule. On-demand was
already required, but only **declared** — by `aws-fleet.sh`'s `KEEL_FLEET_MARKET`, a
variable the launcher set and no gate read back. `remote.sh`'s own test for an assumption
is "is there any mechanism by which this gate could read the precondition back? If yes, it
is a criterion this gate is missing", and `spawn list`'s `spot` boolean is that mechanism.

The join key is the **ssh alias, matched against `spawn`'s `name` exactly**, which is a
constraint on how the fleet is launched rather than a heuristic. Matching on a public
address instead would survive a rename and would also, on a reused address, join a reading
to a machine that did not produce it: a missed join reads `none` and withholds admission, a
wrong join would grant it.

**Amended 2026-08-19 (ruling on #106): bare metal reaches the evidentiary class by a named
arm, and the default stays restrictive.** As first written, `host_admission` read the class
from `instance=` alone, so a machine with no EC2 identity — which is what bare metal is —
fell through the `case` default to `correctness`. That is the right *default* and the wrong
*classification*: **bare metal is the limiting case of full size, not the absence of it**,
and a whole machine with no hypervisor owns its socket more completely than any instance
type can demonstrate. The repair adds an arm rather than widening the fallthrough, because
widening it would trade a false demotion for a false admission and invert the allowlist's
one safety property — a stale list may only withhold a judgement, never grant one. The
governor conjunct is what makes this the *pre-existing* §5 rule 5 instrument (the one that
admitted the lab fleet) rather than a new grant; a guest, which owns no governor, gets rule
5's substitute instrument instead. What the arm cannot see is stated where it is
implemented: the `hypervisor` flag is `CPUID.1:ECX.31`, which a hypervisor sets by
convention and may clear.

**Amended 2026-08-17 (ruling on #104): the evidentiary class is full-size, not metal.**
Scott's earlier ruling retired bare metal outright — *"no one will ever use this library
with bare metal"* — which left the class with no members and the harness judging perf on
whatever guest answered. The property #12 actually argued for was exclusive ownership of
the silicon under measurement, and a whole-socket instance has that while sharing the
deployment model a caller really uses. So the *size* carries the admission, and the
preamble carries the proof.

The reason for the split is that the two roles have different failure modes. A
correctness run either agrees bit-for-bit with the scalar spec or it does not, and a
noisy neighbour cannot change that answer; a throughput run on shared tenancy cannot
distinguish a noisy neighbour or an invisible frequency ceiling from a bad loop nest.

**A number from a correctness-class host is reported-not-judged no matter what it reads**,
high or low, which is the half that had been missing: #104 put a *low* reading from
`c7i.4xlarge` — 16 vCPU of a Sapphire Rapids socket, 8 physical cores — through a floor
written for a machine keel owned, and called the result a P2 STOP. Class is checked
before the number is trusted, not after it surprises someone. And the check fails closed:
an **unreadable** class is `unmeasured`, never "not judged" — otherwise the mechanism that
excuses a partial-size reading is also a mechanism for laundering a red.

**Which judged criteria consult admission, and the one that deliberately does not (§5 rule
12).** Wired: gate-p2's criterion 5b, gate-p3's two percent-of-peak criteria, and gate-p5's
scaling criterion. **Not** gate-p4's criterion 7, `SYRK_FLOOR=0.85` — and the omission is
reasoned, not pending. Every wired criterion divides a rate by something *outside* the
co-tenancy: a theoretical or measured peak, or a single-thread rate on fewer cores than the
numerator used. A noisy neighbour lands in the numerator alone, so the ratio is a claim
about silicon the run may not own, and that is what the class governs. Criterion 7 divides
Ssyrk by Sgemm **at the same thread count on the same host in the same run**, so co-tenancy
sits in both terms and largely divides out; what survives is a statement about two kernels'
relative cost, which is as true on a shared 4xlarge as on a whole socket. Wiring admission
into it would withhold a verdict the class has no bearing on. *This is a judgement, and it
is the one place a reader should look first if a `c7i.4xlarge` ever produces a criterion-7
result that a full-size host contradicts.*

What the evidentiary hosts are *for*: the ≥6× floor was written when the largest gate
host had 16 cores. 6× at 8 threads on a client part with client memory channels says
little about where the parallel nest actually stops scaling — packing-buffer contention
invisible at 16 threads is the whole show at 64. **The floor does not move**; the
full-size hosts add a wider curve (16/32/64 threads) reported beside the judged number,
and they must clear the same ≥6× every other host clears, so adding them can only
make the gate stricter.

Mechanics: `scripts/aws-fleet.sh up|wire|status|down`, launching through `spawn` under
`AWS_PROFILE=aws`, one entry per host with its class recorded, torn down at session end.
**Launched when there is something to measure** — a full-size instance running during stage
1 would bill for hours and measure nothing, which is a reason of measurement and not of
cost.

**Superseded 2026-08-19.** The rule here was: *"the standing grant (2026-08-17) covers
repeats and re-runs of an instance type already approved; a new type is Scott's call each
time."* It no longer holds. The judged-tier directive reads *"launch whatever the evidence
needs, sized and counted by the measurement's requirements alone — no per-launch approval,
no cost hedging; the session-end comment reports what ran, as record not as permission."*
So a new instance type is now a **measurement** decision, reported afterwards rather than
asked beforehand, and `KEEL_FLEET_DRYRUN=1` prices a launch before it happens for the
report rather than for an approval. What did **not** move is `KEEL_EVIDENTIARY_SIZES`: a
type is admitted to the judged class when a read-back on it justifies the addition, so
launching a type freely and *judging* rows measured on it remain two decisions.

## What P3 asks of every host, and of one

**The OpenBLAS reference is same-host, on all three** (ruling on issue #23).
There is no reference-host list and no golden machine. P3's second criterion is
≥60% of single-thread OpenBLAS at 2048³, and the only apples-to-apples version of
that ratio is same silicon, same thread count, same run — so each host is divided
by *its own* OpenBLAS, in one benchmark invocation, from a native build of the
`openblas`-tagged cgo harness shipped as `git archive HEAD`. A cross-host ratio
would compare two microarchitectures and attribute the difference to keel
(DESIGN.md §7 rule 7); this dev host would be the worst case of that, being
`darwin/arm64` where keel's AVX-512 path does not exist at all.

That makes the amd64 hosts the one place in this project that needs a Go
toolchain *and* a system library installed, rather than a cross-compiled static
binary. None of the three had either as of 2026-08-11:

| Host | distro ID | `go` on a non-login `ssh` PATH | `libopenblas.so` | governor |
|---|---|---|---|---|
| vesta | `ubuntu` | none | none | performance |
| janus | `rocky` | none | none | performance |
| antares | `ubuntu` | none | none | powersave → performance (set during provisioning; #44) |

(janus runs Rocky Linux 9, not RHEL proper — the `dnf` package name is the same.)

Provisioning is Scott's decision and Scott's `sudo`, so it lives in
`scripts/provision-openblas.sh`, which he runs; the gate connects with
`BatchMode=yes` and never handles a credential. A host that cannot produce a
reference **fails** the gate by name, with the exact commands for its own distro
printed. It does not fall back to percent-of-peak: CLAUDE.md's "the OpenBLAS
reference when available; otherwise say it isn't" is a rule about reporting
numbers, and using it to satisfy a criterion would be weakening a gate.

Three things get recorded per host, because each of them moves the ratio:

1. **the OpenBLAS version and build string** — what the denominator *is*;
2. **`OPENBLAS_NUM_THREADS=1` taking effect**, read back from
   `openblas_get_num_threads()` rather than assumed from the environment;
3. **the `DYNAMIC_ARCH`-selected kernel family**, from
   `openblas_get_corename()`, checked against an AVX2-or-better allowlist
   (`haswell skylakex cooperlake sapphirerapids zen`).

The third is the one that is easy to omit and the only one whose failure mode is
*in keel's favour*: a package that quietly selects a generic or pre-AVX2 kernel
on an AVX-512 host produces a reference that reads low, which inflates keel's
ratio while the version string, the thread count and the config line all still
look right. An unrecognized name fails too — missing knowledge should cost a
human one line of diff, not silently widen a bar.

**The allowlist bounds the class, and the pin — not the allowlist — is what
protects the denominator.** `OPENBLAS_CORETYPE` selects the family at runtime on a
`DYNAMIC_ARCH` build, and `gate-p3` sweeps it per host and pins the fastest family
it finds before measuring anything. On the AWS fleet at `ce43bca` (`OpenBLAS
0.3.26` Ubuntu package, 1 thread, best of 10 at `-benchtime=1s`) it pinned:

| host | CPU | `DYNAMIC_ARCH` picks unaided | gate pins | pin's margin | vs same-family drift |
|---|---|---|---|---|---|
| keel-zen4 | EPYC 9R14 (Genoa) | Cooperlake, 106.60 | **Haswell, 112.20** | **+5.3%** | 5.60 win vs 0.60 drift |
| keel-zen5 | EPYC 9R45 (Turin) | Cooperlake, 267.60 | SkylakeX, 268.90 | +0.5% | 1.30 win vs 1.10 drift |
| keel-gnr | Xeon 6975P-C (Granite Rapids) | Cooperlake, 210.30 | SkylakeX, 214.70 | +2.1% | 4.40 win vs 4.10 drift |

So the allowlist is a class check and nothing more, and it is not load-bearing for
any published ratio: the reference is the swept winner, and the gate weighs the
cross-family win against the sweep's own same-family drift before believing it —
which on gnr and zen5 is barely decisive (4.40 against 4.10, 1.30 against 1.10)
and says so in the log.

**VOID, 2026-08-20:** an earlier version of this section claimed the allowlist let
a `DYNAMIC_ARCH`-picked reference stand and inflated keel's ratio by up to 5.2%.
There is no such inflation — the gate never divides by the unaided pick. The claim
was published without running the instrument that already settled it (§5 rule 11),
and it was refuted by that instrument's own log two hours later. The measurement
behind it was sound and is a second derivation of the same quantity from a
different harness (keel's `n=2048` median-of-5 got Haswell +5.2% on Zen 4 against
the gate's +5.3% at best-of-10/1s); only the conclusion drawn beside it was wrong.

What does survive, and is corroborated by the gate's own sweep above: on Zen 4 the
fastest of the five families is the **AVX2** one, beating every AVX-512 option
(Haswell 112.20 against SkylakeX 107.90 and Cooperlake 106.30). Zen 4 executes
AVX-512 over a 256-bit datapath, so those kernels buy no throughput there and pay
their overhead anyway. "AVX2-or-better" is therefore an *ordering* that does not
hold on every host it grades — which is why the pin, not the allowlist, decides.

On an **issue-bound** host the denominator is
`min(same-host OpenBLAS, roofline × measured peak)` (the same ruling, citing
#17/#18): OpenBLAS's K-loop there is hand assembly folding accumulation and an
embedded broadcast into single FMAs, which the intrinsic layer provably cannot
emit (T12), so it sits *above* the front-end ceiling keel's kernels are capped
by. vesta and antares classify FMA-bound and therefore face the unmodified
criterion — which keeps at least one host where the comparison is completely
unassisted. Both ratios, amended and plain, are printed on every host.

Note what stays true regardless: the `openblas` tag keeps that harness out of the
module's dependency graph, so nothing keel ships ever links OpenBLAS.

*The dev host has OpenBLAS, and that is useful for exactly one thing.* Homebrew
OpenBLAS 0.3.34 is installed here, so the tagged harness is compile- and
run-verifiable locally — the cgo declarations, the linker flags, the provenance
plumbing and the thread pin were all checked this way before the gate ever shipped
them to a remote host (see toolchain note T13 for the file layout cgo forced).
What it cannot do is produce the criterion, and the measurements say why more
plainly than the architecture argument does:

| Config (n=1024, single sub-benchmark) | Rate |
|---|---|
| OpenBLAS, `OPENBLAS_NUM_THREADS=1 OMP_NUM_THREADS=1` | ~1410 GFLOP/s |
| OpenBLAS, unpinned (`threads=12`) | ~1400 GFLOP/s |
| keel, scalar backend (n=256) | ~6 GFLOP/s |

The pin works and is honest — `openblas_get_num_threads()` reports 1, and user CPU
time equals wall time to within a few percent, so one thread really is doing the
work. That one thread reaches roughly ten times what this core's NEON FMA units
can issue, and adding eleven more threads buys nothing, which together say the
reference is running on the M4's matrix unit rather than on its vector units
(config target `vortexm4`). Against that, keel's *scalar* path is the only thing
this host can run. A ratio of ~1:200 between a matrix coprocessor and a Go scalar
loop is not evidence about the AVX-512 kernel P3's criterion is actually about.

Two of the longer local runs of the *identical* pinned configuration also came in
at 119 and 712 GFLOP/s — up to 12× run-to-run — which is what a laptop with no
governor to pin and an aggressive QoS scheduler looks like. DESIGN.md §5 rule 5's
methodology exists to exclude exactly this, and it is a second, independent reason
no number from this machine is reportable.

**The throughput sentinel** is the one role P3 assigns to a single host, named in
`.keel-sentinel` or `$KEEL_SENTINEL_HOST` — same format as `.keel-hosts`, and
gitignored for the same reason. *(Both channels retired 2026-08-31; see the
correction below.)* The ruling
on issue #19 left P2's floor with a class-dependent denominator, and on an
issue-bound host that floor *rises* as the kernel's instruction count falls. The
same arithmetic makes such a host the one that notices a K-loop getting fatter,
which is the risk P3 carries — packing, edge handling and beta variants all add
code around the loop. So gate P3 re-runs P2's verdict there. janus is the
sentinel today: it is the host where instruction count binds (46.0% of peak,
94.6% of a 48.6% issue roofline). If the file is absent, every host is a
sentinel — missing configuration costs time, never coverage.

**janus keeps that role through P5** (confirmed 2026-08-12 by ruling). The
question raised was whether a host sitting at 31.9% of peak should go on
certifying phases, and the answer is that this is precisely the host worth
keeping: **as of 2026-08-12** the only Intel part and the only issue-bound one,
and the machine the roofline section's standing task names as the thing to
re-measure when the lowering improves. Because the amendment ratchets — the floor is monotone
non-increasing in `I_b` — the exception tightens automatically rather than
granting a permanent dispensation. A host whose low number is *explained*, by a
model that gets stricter as the explanation goes away, carries more information
per run than a host that simply passes.

Gate P3 classifies *every* host with that same measurement even when only one is
judged by it, because the amended denominator above applies only where the
classifier says issue-bound. A host whose classification could not be measured is
treated as FMA-bound and faces the unmodified bar, which is the strict direction.

**CORRECTION, 2026-08-31 (#146, ruled): the two sentences above naming how a sentinel
is selected are retired, and the "janus keeps that role through P5" clause is VOID for
any judged run.** The selection channels described above — `$KEEL_SENTINEL_HOST`, then
`.keel-sentinel` — outranked the configured fleet, so a gitignored, machine-local,
three-week-old file naming `janus.local` decided which host held a *judged* role. It
did so in the #113 re-measurement **and in the run that signed `v0.1.0-a2`**, where
`keel-skx`, `keel-zen4` and `keel-zen5` each printed *"not a sentinel, so P2's floor is
not judged here"*. That is the migration ruling (judged measurement on AWS; the lab is
the dev tier) violated through a channel nobody enumerated, and rule 21's total
restatement could not see it because a namespace clear reaches variables, not files.

What holds now: the sentinel set is **every fleet host**, derived from the same
`.keel-hosts`/`$KEEL_REMOTE_HOSTS` the rest of the gate reads. `.keel-sentinel` is not
a selection input at all, and if it exists the gate names it, with its mtime, as *not
read*. `$KEEL_SENTINEL_HOST` **fails** the gate rather than being silently ignored,
because a no-op override reads exactly like an honoured one. An out-of-fleet sentinel
is still available for deliberate characterization work, but only through
`$KEEL_SENTINEL_OUT_OF_FLEET`, which is a *union* with the fleet — no flag can subtract
a fleet witness — and which prints into the run's `.cmd` and into the log's declaration
row, so a certificate rendered with a lab sentinel says so on its face.

The widening from "a fleet host" to "every fleet host" is mine and is disclosed on
#146: any proper subset needs a tie-break among equals, which is the arbitrariness being
removed rather than a fix for it, and judging more hosts can only turn a green red.

janus's standing as *the* host where instruction count binds is unaffected as a
statement about the silicon — the table below still reports it — but it is no longer a
statement about who judges. **P2's floor was judged on the fleet for the first time on
2026-09-01** (`archive/pinned8/p2onfleet-afb108e.log`, `gate-p3: GREEN`, 52 PASS / 0
FAIL), and it holds on all three: `keel-zen5` 187.3 / 287.7 GFLOP/s = 65.1% and
`keel-zen4` 113.3 / 117.0 = 96.8%, both fma-bound against the flat 55% floor; `keel-skx`
88.79 / 185.6 = 47.8%, which is 96.9% of that shape's 49.3% issue roofline against the
90% bar. Both terms of each ratio are that host's own, from the *same* invocation —
`BenchmarkKernel` on the dispatched shape over `BenchmarkPeak/avx512`, medians of 30,
recomputed here from the run's tracked sample files rather than quoted, because the gate
log publishes the ratio and neither term. The peaks in the Sgemm section of the same log
(288.2 / 117.0 / 186.3) are a *different* invocation and are not these denominators. The
two fma-bound readings are also not comparable to each other as percentages: 96.8% of
117.0 is a smaller rate than 65.1% of 287.7. That run
was deliberately **not** a certificate: per DESIGN.md §5's *"its first firing should not
double as its first test"*, the declaration row and these three judgments had to be
rendered by some real gate before a **certificate** run rendered them.

**The uniqueness clause above went false without an edit, which is why it now carries
its date.** Six hosts have been classified since, four of them Intel, and one of the
new ones is issue-bound too — so "the only Intel part, the only issue-bound one" was a
count stated as a permanent property. The rows, reported and not judged (the class is
what selects P2's denominator, per §"Cause of the roofline"); peak and the
`2x32/avx512/kc=128` rate are each host's own measured GFLOP/s at 1 thread, AWS rows
from the wave-1 and wave-2 classification passes, `7ac592a`, 2026-08-21:

| host | µarch | measured peak | 2x32/avx512/kc=128 | % peak | class |
|---|---|---|---|---|---|
| keel-zen5 | Zen 5 (EPYC 9R45) | 288.6 | 153.5 | 53.2% | fma-bound |
| keel-zen4 | Zen 4 (EPYC 9R14) | 117.2 | 111.5 | 95.1% | fma-bound |
| keel-gnr | Granite Rapids (Xeon 6975P-C) | 245.4 | — | — | fma-bound |
| keel-icx | Ice Lake (Xeon Platinum 8375C) | 213.4 | 104.0 | 48.7% | fma-bound |
| keel-spr | Sapphire Rapids | 232.3 | 61.45 | 26.5% | fma-bound |
| keel-skx | Skylake-X (Xeon Platinum 8124M) | 192.9 | 88.91 | 46.1% | **issue-bound** |

Two readings, worth keeping apart. **Issue-boundedness tracks the microarchitecture,
not the machine**: janus (lab i9-9960X, bare metal, `performance`) reads 46.0% of its
measured peak and keel-skx (virtualized AWS, no governor at all) reads 46.1% of its
own — different SKUs, clocks and memory systems, same Skylake-X front end, same shipped
2×32. That is a second sample of the *claim* about Skylake-X and **not** a second
witness to the *classifier*: both numbers come from the same instrument, and §5 rule 10
counts independent derivations rather than agreeing sites. Separately, **ICX measured
fma-bound** (mixes diverge `1.103×`, wholly over the `1.10` bar), contradicting the
premise that SKX *and* ICX are issue-bound silicon — the instrument was run against the
reasoning that motivated launching it and refuted half of it (§5 rule 11). Nothing
downstream moved: icx is correctness-class and judged nothing.

**Settled 2026-08-31, by ruling on #146 — this paragraph asked for a ruling and got
one.** It read *"the sentinel rationale above rests on janus being the lone issue-bound
host, and it no longer is … whether the sentinel role moves, splits or stays is a ruling,
not a measurement."* The answer is that the role does not sit on any one host: the
sentinel set is every fleet host, so `keel-skx`'s issue-boundedness is now judged where
it lives rather than making janus's role awkward. The lone-host premise is retired along
with the selection channels that rested on it, and the first fleet pass to render that
(2026-09-01, above) judged both classes at once — two fma-bound hosts against the flat
55% floor and one issue-bound host against its roofline.

## Placement methodology: pinned since 2026-08-21

Every judged benchmark invocation on every host runs under a CPU affinity mask of eight
distinct physical cores inside one NUMA node — **one core per cache domain**, amended
2026-08-22 — and the ceiling arm runs under the identical mask. Adopted fleet-wide by ruling
on #6; the law, the enumeration and the falsification condition are DESIGN.md §5 rule 5.

The amendment is not a refinement of taste. The form adopted on 2026-08-21 took the first
eight cores in ascending order, which on EPYC 9R45 — whose `index3` `shared_cpu_list` is
exactly eight cores wide — is definitionally **one CCD**: the 8-thread stream ceiling it
measured sat 5.96× (dot) and 4.65× (axpy) below the same host measured one-core-per-CCD, and
*below free placement itself* by 1.69×. So the confined mask would have regenerated this
file's and the README's bandwidth-bound rows several-fold under what the silicon does, which
§5 rule 16 forbids in the same words it forbids overselling. On `keel-skx`, whose L3 is per
socket, the node is one domain and the mask is the same consecutive eight it always was.
Every archive from 2026-08-22 records the domain of each masked core and the count its node
offered, so the shape is checked off the artifact rather than trusted (`gate-p5.sh`,
`bench_pin_spread`). **Before this date every number in this file and in the README was
measured under free placement**, which matters when comparing across the boundary: the whole
judged fleet is 2-socket/2-NUMA (keel-skx 2×18×2SMT, keel-zen4 and keel-zen5 2×96), so an
unpinned goroutine could and did migrate across sockets mid-measurement.

What drove it was not a preference for tidiness but four independent readings that the free
instrument was reporting the *draw* rather than the code — zen4's `Strsm` verdict flipping red
then green on unchanged code, zen5's `Ssyrk` clearing a bar by 0.4 points where its derivation
set 2.6, skx's `Strsm` clearing 7.0 at its median and failing net of CI, and criterion 9's 5%
band coming in narrower than the spread of the rows it judges. The transition campaign runs
**both** arms and archives both, so the crossing is measurable rather than asserted, and any
verdict that changes colour is published with both readings side by side.

The boundary has a name the gates read: this is the **`pinned8` measurement era**, and
everything before it is `free-placement` (`scripts/measurement-eras.tsv`, DESIGN.md §5 rule
17 clause (d)). It is not a label — a registered baseline is scoped to its era, so a
free-placement reference cannot be applied to a pinned reading, and that is enforced by the
reader rather than by a reviewer noticing. Which is why every host's registry row is empty
today: no host has a baseline in this era yet, and the transition run is where they are born.

Two limitations sit inside the figures rather than beside them. The mask pins the
`threads=1` rows to a *node*, not to a core, so the ±0.11% a one-core probe read for skx's
1-thread `Sgemm` — against ±14.6% unpinned, in a run of `-count=20` against the gate's 10 — is
not what this weaker mask promises. And because Go reports the mask's width as `GOMAXPROCS`,
benchmark names carry a `-8` suffix where the free arm carried `-192` or `-72`; that is a
renames every row, which every comparison across the boundary has to cope with.

## What has been verified here

2026-08-10, gate P0: the differential suite ran on all three hosts, bit-exact
across all 14 shim ops on all three backends. This settled the `Max`/`Min`
operand-order question (issue #9) that disassembly could not: `x.Max(y)`
returns `y` for NaN and for `max(±0, ∓0)` on Zen 4, Zen 5 and Skylake-X alike,
which is what the scalar spec already claimed. Three independent
implementations of `VMAXPS` agreeing is the reason that claim is now stated as
fact rather than as a convention keel happens to follow.

2026-08-11, gate P1 under the revised measurement methodology (issues #11, #14):
each host's FMA ceiling measured by `bench.BenchmarkPeak`, single core,
`-count=10 -benchtime=1s`, benchstat medians.

| Host | µarch | 512-bit | 256-bit | width ratio | formula error | Sdot n=4096 speedup (net of CI, three runs) |
|---|---|---|---|---|---|---|
| vesta | Zen 4 | 165.6 | 165.5 | 1.00× | 2.23× high | 8.71× / 7.18× / 8.57× |
| janus | Skylake-X | 215.9 | 101.8 | 2.12× | 1.30× high | 7.55× / 7.48× / 7.53× |
| antares | Zen 5 | 327.8 | 164.0 | 2.00× | 1.01× high | 9.14× / 8.96× / 8.79× |

Those rows are dated, and one of them is dated for a reason: **antares was in
`powersave` when its figures were taken** and is in `performance` now (#44), so its
Sdot column in particular is not directly comparable with a re-run today. The
measurements stand as records of what was measured; they are not restated as
current.

> **Provenance of antares's Sdot trio, audited 2026-08-16 (#79).** Stronger than
> "dated": the trio is **unprovenanced**. No archived log in this tree contains
> `8.96×` as an Sdot ratio at all, so the derivation of at least one of its three
> entries no longer exists — DESIGN.md §5 rule 8's terminal case, a derived figure
> whose log is gone. Readings of the same quantity under the same methodology with
> the governor read back as `performance` are consistently higher: four archived
> `gate-p1` logs give 9.626× / 9.716× / 9.728× / 9.740× net of CI, and this
> document's own 2026-08-12 re-measurement gives 9.13× / 9.29× / 10.37×. Against
> those, the published `8.79×` is 9–10% low.
>
> The leading explanation for the trio's shape is arithmetic rather than physical:
> `9.14` is, to three significant figures, the **raw** ratio of the run whose
> net-of-CI is `8.79`, so a list captioned "net of CI" may hold one raw value and
> "three runs" may be two. That is a hypothesis, not a finding.
>
> The governor is **not** credited with causing the gap. #44 established that it was
> a suspect *removed*, not a cause *shown*, and inferring here what that
> investigation declined to infer from data would repeat the error one document
> later. The figures are marked, not rewritten (#79's ruling): the gap is real, the
> direction is consistent across seven independent readings, and the cause is
> unestablished.

GFLOP/s, float32. The width ratio is the useful column: it is a direct
measurement of the datapath, requiring no clock estimate, port count or
frequency-license assumption, because both halves were measured on the same host
under identical conditions. The measured ceilings reproduce to within 0.4%
between gate runs; the Sdot ratios do not (see below), which is the difference
between a register-only kernel and one that touches memory.

**One thing not yet settled: run-to-run drift on vesta, and it is bimodal.**
Three gate runs over a few hours reported the same Sdot ratio as **8.71×, 7.18×
and 8.57×** net of CI, while benchstat's within-run interval was 0–2% every
time. janus reproduced to 0.9% across the three (7.55×, 7.48×, 7.53×) and
antares to 4% (9.14×, 8.96×, 8.79× — but see the provenance note below; the
`performance`-era readings of that host span 13.6%, not 4%). So this is specific to
vesta, and the
*shape* is informative: two runs agree to 1.6% and one sits 17% below them.
Thermal drift would be monotonic in run order; a bimodal distribution with a
tight upper cluster is what core placement looks like. The 7950X3D has two CCDs,
one with 3D V-Cache and a lower clock ceiling, and nothing pins these
single-threaded benchmarks to either — the scheduler picks, and roughly a third
of the time here it picked the slower one.

Measured *peak* is unaffected (0.4% across the same three runs), because a
register-only kernel has nothing to be lucky about; the drift appears only once a
benchmark touches memory. A within-run confidence interval cannot see any of
this, so "cleared net of CI" is not by itself protection against it, and a P2
threshold on a memory-touching kernel should not be trusted to 17% on this host.
Settling it needs `taskset` pinning, which changes what the measurement *means*
— pin to the V-Cache CCD or the higher-clocked one? the honest answer differs
for an L1-resident kernel and for a blocked SGEMM — and so is a decision to take
deliberately rather than a fix to slip in.
