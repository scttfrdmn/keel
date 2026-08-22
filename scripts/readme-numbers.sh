#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# readme-numbers.sh -- regenerate README.md's keel-numbers block, and the caption region
# under it (the run's provenance, plus which ratios sit below the gate's floors), from a
# gate-p5 log. scripts/docs-gen.sh extracts both regions onto doc-site/numbers.md.
#
# THE LAW THIS ENFORCES IS docs-gen.sh'S OWN LAW, MOVED ONE FILE UPSTREAM.
# docs-gen.sh already refuses to let doc-site/numbers.md be a hand-copy of README's
# keel-numbers block. Until this script existed the block was itself hand-assembled
# from a gate log, so that law stopped one file short of the numbers and the last
# hand-maintained duplicate in the chain was the published one.
#
# What that cost, measured 2026-08-19 (ruling on #6, gate-p5 at 7ee750d): the block's
# own ratios put FIVE rows on THREE hosts below the gate's scaling floors, while the
# caption three lines below it said the floor "was missed ... on the two hosts that
# keep the most of their single-thread peak". Prose that decayed while the numbers it
# summarised sat in the same file. The caption is therefore generated here too, from
# the same parse, so it cannot describe a table it disagrees with.
#
# THE BLOCK IS A PUBLISHED, CITABLE ARTIFACT, so this generator is in-tree rather
# than a scratch derivation: a script that produces published numbers from outside
# version control is provenance living in a gitignore -- unreviewable, unreproducible
# by anyone else, and the hand-assembly hole with one extra step. Same reasoning as
# tools/shapegen, whose emitted kernels are checked against the shipped ones in-tree.
#
# It NEVER measures. It reads a log a gate produced and rewrites two marked regions
# of README.md in place, so README stays the single source docs-gen.sh extracts from
# and this adds no second home for a rate.
#
# Usage:
#   scripts/readme-numbers.sh build/gate-p5-<rev>.log      rewrite README.md in place
#   scripts/readme-numbers.sh build/gate-p5-<rev>.log -n   print the block, touch nothing
set -euo pipefail

NUM_BEGIN='<!-- keel-numbers: begin -->'
NUM_END='<!-- keel-numbers: end -->'
CAP_BEGIN='<!-- keel-caption: begin -->'
CAP_END='<!-- keel-caption: end -->'

# Kept in step with gate-p5.sh by the check at the bottom of this script, not by
# being remembered: a floor that drifts from the gate's would publish a disclosure
# about a bar nothing enforces.
#
# CEIL_FRACTION replaced SCALE_FLOOR on 2026-08-20 (#6): the judged class is compared to
# each host's own measured 8-thread ceiling. Ratified 2026-08-21 at 58.5, re-typed to 57.8
# that day when a repair restored the ceiling's interval to the site computing its input (#6
# Q2), then emptied the same day when the `pinned8` era boundary retired it. The empty-bar
# branch below is live, not dead code — it has served two deferred bars and will serve a third.
#
# 51.0 was TYPED 2026-08-22 from rows measured under the mask's confined first form, and
# SUSPENDED the same day when the spread amendment changed the instrument its denominator was
# measured with. STRSM_FLOOR is suspended beside it, made era-scoped by the same ruling.
# gate-p5.sh carries both derivations and is the authority; emptied here in the same commit
# because the check below reads those lines back verbatim, so this is a second edit and not a
# second decision. The published shares this caption governs are regenerated separately, as
# medians over this era's archives (#6): a bar and the rows it judges are not one act.
CEIL_FRACTION=
STRSM_FLOOR=
SCALE_FLOOR_RETIRED=6.0
ROUTINES='Sgemm Ssyrk Ssymm Strsm'

die() { echo "readme-numbers: $*" >&2; exit 1; }

LOG="${1:-}"
DRY="${2:-}"
[[ -n "$LOG" && -r "$LOG" ]] || die "usage: $0 <gate-p5 log> [-n]"

# The floors are read back out of the gate rather than trusted here. A published
# disclosure naming 6.0x while the gate enforces something else is exactly the
# caption-drift defect this script exists to end, one level up. The RETIRED constant
# is read back too, because the caption names it as retired: if it were deleted from
# the gate, or worse revived there as a live comparison, this disclosure would be
# describing the wrong one of two 6.0s.
for pair in "CEIL_FRACTION=$CEIL_FRACTION" "STRSM_FLOOR=$STRSM_FLOOR" "SCALE_FLOOR_RETIRED=$SCALE_FLOOR_RETIRED"; do
  grep -qxF "${pair%%=*}=${pair#*=}" scripts/gate-p5.sh \
    || die "${pair%%=*} here is ${pair#*=} but scripts/gate-p5.sh disagrees; the disclosure would name a bar the gate does not enforce"
done

BLOCK="$(awk -v routines="$ROUTINES" -v cf="$CEIL_FRACTION" -v tf="$STRSM_FLOOR" \
         -v retired="$SCALE_FLOOR_RETIRED" -v src="$(basename "$LOG")" '
  function strip(s) { gsub(/\033\[[0-9;]*m/, "", s); return s }

  # Every line the gate emits per host is tagged [host.local]; the provenance line
  # is the one whose first field after the tag is the CPU model.
  {
    line = strip($0)
    # Run-wide provenance, before the host-tag guard or the guard skips it. The sha comes off
    # a sample path: "green on this commit (sha)" prints only when GREEN, so reds lost the rev.
    if (match(line, /bench-gate-p5-[0-9a-f]{7,40}-/)) { rev = substr(line, RSTART + 14, RLENGTH - 15) }
    if (match(line, /^-- scaling at 8 cores on [0-9]+\^3/)) { nsz = substr(line, RSTART + 25, RLENGTH - 27) }
    if (match(line, /\[[a-zA-Z0-9_.-]+\]/) == 0) next
    host = substr(line, RSTART + 1, RLENGTH - 2)
    rest = substr(line, RSTART + RLENGTH + 1)

    # provenance: MODEL | instance=... | virt=... | ... | governor=... | ...
    if (rest ~ /\| instance=/ && !(host in model)) {
      m = rest; sub(/ *\|.*$/, "", m)
      model[host] = m
      if (match(rest, /governor=[a-z_]+/)) { gov[substr(rest, RSTART + 9, RLENGTH - 9)]++ }
      if (!(host in seen)) { order[++nh] = host; seen[host] = 1 }
    }

    # rates: ROUTINE: 1 thread X GFLOP/s +/- p%, 8 threads Y GFLOP/s +/- q%
    if (match(rest, /^[A-Za-z]+: 1 thread /)) {
      r = rest; sub(/:.*$/, "", r)
      if (match(rest, /1 thread [0-9.]+/))  { t1 = substr(rest, RSTART + 9, RLENGTH - 9) }
      if (match(rest, /8 threads [0-9.]+/)) { t8 = substr(rest, RSTART + 10, RLENGTH - 10) }
      one[host, r] = t1; eight[host, r] = t8
    }

    # The scaling verdict is EXTRACTED, never recomputed. The first draft of this
    # script re-derived it from the point-estimate ratio b/a and disagreed with the
    # gate on 2 of 12 rows, because the gate judges net of CI -- so the caption would
    # have named 3 shortfalls where the gate fails 5. Checking that the floor
    # CONSTANTS matched gate-p5.sh did not catch it: the constants agreed and the
    # PREDICATE did not. The instrument adjudicates (DESIGN.md §5 rule 11), and a
    # disclosure that re-implements the criterion it discloses is the drift this
    # script exists to end, one level in. Two shapes, because Strsm omits the
    # thread-count clause:
    #   [h] Sgemm scales only 5.802x at 8 threads, 5.744x net of CI (< 6.0x)
    #   [h] Strsm scales 7.010x, 6.940x net of CI (< 7.0x, ...)
    if (match(rest, /^[A-Za-z]+ scales /) && rest ~ /net of CI/) {
      r = rest; sub(/ scales .*$/, "", r)
      pt = rest; sub(/^.*scales (only )?/, "", pt); sub(/x.*$/, "", pt)
      # Anchored on "x net of CI", not on the last comma: the Strsm lines carry a
      # comma INSIDE their trailing parenthetical ("(< 7.0x, the regression bar
      # ratified for this class)"), so a greedy comma split reads that prose as
      # the ratio. The Sgemm/Ssymm lines have no such comma and parsed correctly,
      # which is why only the two Strsm rows -- the near-margin ones this caption
      # exists to name -- came out wrong.
      ci = ""
      if (match(rest, /[0-9.]+x net of CI/)) { ci = substr(rest, RSTART, RLENGTH); sub(/x net of CI$/, "", ci) }
      verdict[host, r] = (line ~ /^ *FAIL/) ? "FAIL" : (line ~ /^ *PASS/) ? "PASS" : "OTHER"
      # Both ratios must parse as numbers. A ratio that came out as prose would be
      # published as prose, and the row it describes is the one a reader checks.
      if (pt !~ /^[0-9]+\.?[0-9]*$/ || ci !~ /^[0-9]+\.?[0-9]*$/)
        { printf "readme-numbers: [%s] %s verdict line did not yield two ratios (point=%s, net=%s)\n", host, r, pt, ci > "/dev/stderr"; bad = 1 }
      ptr[host, r] = pt; cir[host, r] = ci
    }

    # The JUDGED class changed shape with the criterion (#6, 2026-08-20): it no longer
    # "scales Nx" against a cross-host floor, it "reaches F% of" its own measured
    # ceiling, and the ratio rides along as reported context. The apostrophe in the
    # gate wording is deliberately not matched -- this awk program is inside a
    # single-quoted shell string, so the discriminator is the thread-count clause.
    #   [h] Sgemm reaches 65.9% of this ... measured 8-thread ceiling (852.1 GFLOP/s), scaling 5.792x / 5.744x net of CI -- ...
    #   [h] Sgemm reaches only 65.9% of this ... 8-thread ceiling (852.1 GFLOP/s) (< 90%), scaling ...
    if (match(rest, /^[A-Za-z]+ reaches /) && rest ~ /measured [0-9]+-thread ceiling/) {
      r = rest; sub(/ reaches .*$/, "", r)
      fr = ""; if (match(rest, /reaches (only )?[0-9.]+%/)) { fr = substr(rest, RSTART, RLENGTH); sub(/^reaches (only )?/, "", fr); sub(/%$/, "", fr) }
      ce = ""; if (match(rest, /ceiling \([0-9.]+ GFLOP\/s\)/))  { ce = substr(rest, RSTART, RLENGTH); sub(/^ceiling \(/, "", ce); sub(/ GFLOP\/s\)$/, "", ce) }
      pt = ""; if (match(rest, /scaling [0-9.]+x/))              { pt = substr(rest, RSTART + 8, RLENGTH - 9) }
      ci = ""; if (match(rest, /[0-9.]+x net of CI/))            { ci = substr(rest, RSTART, RLENGTH); sub(/x net of CI$/, "", ci) }
      verdict[host, r] = (line ~ /^ *FAIL/) ? "FAIL" : (line ~ /^ *PASS/) ? "PASS" : "OTHER"
      if (match(rest, /measured [0-9]+-thread/)) { nt = substr(rest, RSTART + 9, RLENGTH - 16) }
      # Same refusal as above, extended to the two new numbers: a fraction or a
      # ceiling that came out as prose would be published as prose.
      if (fr !~ /^[0-9]+\.?[0-9]*$/ || ce !~ /^[0-9]+\.?[0-9]*$/ || pt !~ /^[0-9]+\.?[0-9]*$/ || ci !~ /^[0-9]+\.?[0-9]*$/)
        { printf "readme-numbers: [%s] %s ceiling verdict line did not yield four numbers (frac=%s, ceiling=%s, point=%s, net=%s)\n", host, r, fr, ce, pt, ci > "/dev/stderr"; bad = 1 }
      frr[host, r] = fr; ptr[host, r] = pt; cir[host, r] = ci
      # One ceiling per host, printed once per judged routine: three printings of one
      # measurement are one witness (§5 rule 10), so they cross-check and never corroborate.
      if ((host in ceil8) && ceil8[host] != ce)
        { printf "readme-numbers: [%s] prints two 8-thread ceilings in one run, %s and %s\n", host, ceil8[host], ce > "/dev/stderr"; bad = 1 }
      ceil8[host] = ce
    }

    # peak, printed once per routine per host, beside the 8-thread percent the gate computes
    if (match(rest, /% of 8x the single-thread avx512 peak \([0-9.]+ GFLOP\/s\)/)) {
      r = rest; sub(/:.*$/, "", r)
      pc = rest; sub(/^.*: /, "", pc); sub(/%.*$/, "", pc)
      pk = rest; sub(/^.*peak \(/, "", pk); sub(/ GFLOP.*$/, "", pk)
      # Four printings of one measurement are ONE witness (DESIGN.md §5 rule 10), so
      # they are used as a consistency check and never as corroboration.
      if ((host in peak) && peak[host] != pk)
        { printf "readme-numbers: [%s] prints two single-thread peaks in one run, %s and %s\n", host, peak[host], pk > "/dev/stderr"; bad = 1 }
      peak[host] = pk; gpc[host, r] = pc
    }
  }

  END {
    if (bad) exit 3
    n = split(routines, R, " ")
    if (nh == 0) { print "readme-numbers: the log names no host with a provenance line" > "/dev/stderr"; exit 3 }

    print "| CPU | benchmark | threads | GFLOP/s | denominator |"
    print "| --- | --- | --- | --- | --- |"
    for (i = 1; i <= nh; i++) {
      h = order[i]
      if (!(h in peak)) { printf "readme-numbers: [%s] has no measured single-thread peak\n", h > "/dev/stderr"; exit 3 }
      p1 = peak[h] + 0; p8 = p1 * 8
      for (j = 1; j <= n; j++) {
        r = R[j]
        a = one[h, r]; b = eight[h, r]
        # A thin block is not a table with one row fewer; it is a published table
        # that silently dropped a host. Same refusal the docs-gen.sh extractions make.
        if (a == "" || b == "") { printf "readme-numbers: [%s] %s has no 1-thread/8-thread pair in this log\n", h, r > "/dev/stderr"; exit 3 }

        # The 1-thread percent is computed because the gate does not print it. The
        # 8-thread percent the gate DOES print, so it is recomputed and checked --
        # this script auditing its own denominator against the one the instrument used.
        c8 = (b + 0) / p8 * 100
        if ((h, r) in gpc) {
          d = c8 - (gpc[h, r] + 0); if (d < 0) d = -d
          if (d > 0.15) { printf "readme-numbers: [%s] %s 8-thread share computes to %.1f%% but the gate printed %s%%\n", h, r, c8, gpc[h, r] > "/dev/stderr"; exit 3 }
        }
        printf "| %s | %s | 1 | %s | %.1f%% of %s GFLOP/s, the 1-thread avx512 microkernel peak measured in the same run |\n", \
          model[h], r, a, (a + 0) / p1 * 100, peak[h]
        printf "| %s | %s | 8 | %s | %.1f%% of %.1f GFLOP/s, that same peak x 8 cores |\n", \
          model[h], r, b, c8, p8

        # A row with no verdict is not a passing row. Fail closed: the caption would
        # otherwise be silent about exactly the row the gate could not judge.
        # A log from before the #6 ruling has rates and no ceiling verdict, and lands
        # here rather than being republished under a criterion it was never judged by.
        # Re-adjudicating those runs is its own deliverable working from the archived
        # samples, not something this generator may do by silently keeping the old bar.
        if (!((h, r) in verdict)) { printf "readme-numbers: [%s] %s has rates but no scaling verdict in this log, so its disclosure cannot be derived (a pre-2026-08-20 log has no ceiling verdict to publish)\n", h, r > "/dev/stderr"; exit 3 }
        if (verdict[h, r] == "FAIL") {
          # THREE KINDS NOW, because the two classes are judged by different
          # instruments. The judged class is compared net of CI against its own
          # measured ceiling, and that comparison yields ONE fraction -- there is no
          # point-estimate-vs-CI distinction to draw, so a shortfall there is just a
          # shortfall, named as a share of the ceiling and NOT as that ceiling in
          # GFLOP/s: this string lands in the caption, which is outside the block
          # criterion 9 re-measures, so a rate here would be a claim (§7 rule 7).
          # It was one, latently -- unreachable only because CEIL_FRACTION ships
          # deferred-empty, so the day a fraction is ratified and any row missed it,
          # criterion 9 would have gone red for a reason unrelated to the shortfall.
          if (r != "Strsm") {
            short[++ns] = sprintf("%s %s at %s%% of its own measured %s-thread ceiling", model[h], r, frr[h, r], (nt == "" ? "8" : nt))
          } else {
            # Strsm keeps the ratio bar and so keeps the distinction, which asks for
            # different things: a point estimate already under the floor is a
            # shortfall, while one that clears the floor and fails only net of CI is a
            # verdict decided by the noise the measurement itself carries, and the
            # remedy for that is precision (DESIGN.md §4, line 130) rather than a
            # discussion about the nest.
            if (ptr[h, r] + 0 >= tf + 0) near[++nn] = sprintf("%s %s (%sx, %sx net of CI)", model[h], r, ptr[h, r], cir[h, r])
            else low[++nl] = sprintf("%s %s at %sx (%sx net of CI)", model[h], r, ptr[h, r], cir[h, r])
          }
        }
      }
    }
    # The caption, derived from the same parse that produced the rows above it.
    printf "\n%s\n", "CAPTION:" > "/dev/stderr"
    # The run that produced the rows, named from the run rather than from memory: the
    # hand-written form of this sentence still said rev 083cbdb four runs later. One
    # governor is printed only if all hosts agree, because "performance governor" over a
    # mixed fleet is the caption-drift defect with a smaller blast radius. Idleness is
    # NOT claimed: the gate witnesses the governor and not the load, and a provenance
    # sentence may only assert what its log shows.
    # TWO QUANTITIES, TWO DENOMINATORS (DESIGN.md §7 rule 7). The table has nh*n*2 rows
    # and they form nh*n ratios, because a ratio consumes the 1-thread and 8-thread row
    # of one routine. The first draft published "All 12 rows" above a 24-row table and
    # "4 of 12 rows" for a count of ratios, in the same paragraph. The conflation is
    # older than this script -- the issue-6 comment says "5 of 12 published README rows"
    # about the same 12 ratios -- so naming both denominators is the fix, not renaming one.
    nr = nh * n; nrow = nr * 2
    g = ""; for (k in gov) g = (g == "" ? k : "mixed")
    # "rev `<sha>`", not "commit `<sha>`": docs-gen.sh:92 parses this exact sentence for
    # /rev `[0-9a-f]{7,40}`/ to date doc-site/numbers.md, and dies with "the page has no
    # run to date itself by" if it cannot find one. The generated sentence has to satisfy
    # the parser that was already reading the hand-written one.
    printf "All %d rows come from one run — `scripts/gate-p5.sh` at rev `%s`, log in `build/%s` — at n=%s square, `GOMAXPROCS` pinned to the threads column, %s. The 1-thread and 8-thread rows for a routine are the two arms of that run'"'"'s scaling ratio, so they are directly comparable to each other; rows from different CPUs are not, because the peaks differ.\n\n", \
      nrow, (rev == "" ? "unrecorded" : rev), src, (nsz == "" ? "4096" : nsz), \
      (g == "" ? "governor unrecorded" : g == "mixed" ? "governors differing between hosts (see the log)" : "`" g "` governor on every host") > "/dev/stderr"
    # THE COLUMN DENOMINATOR IS NOT THE CRITERION DENOMINATOR, and after #6 that has
    # to be said in the caption rather than inferred from the column. "8x the 1-thread
    # peak" is a true statement about what the percent divides by and an unattainable
    # ceiling by construction -- every part here loses several hundred MHz going from
    # one active core under a 512-bit license to eight, for reasons having nothing to
    # do with the parallel nest. The gate judges against a ceiling measured AT that
    # thread count instead, which is the number named in the bars sentence below.
    if (nt != "") {
      printf "The 8-thread rows divide by 8x the 1-thread peak, which no host can reach: the clock drops with core count, so that share is a floor on how well the nest did and not a score. The bar below divides by a ceiling measured at %s threads on the host itself, where the droop is inside the reading.\n\n", nt > "/dev/stderr"
      # THE CRITERION'"'"'S OWN READINGS BELONG ON THE NUMBERS PAGE. Without this the
      # caption publishes the share it just called "not a score" and withholds every
      # share the gate actually judges by. The per-host gap between the two
      # denominators is the measured form of #6'"'"'s argument: 8x-the-1-thread-peak
      # overstates the attainable ceiling by a DIFFERENT factor on each host, which is
      # what makes a fixed cross-host ratio rank them wrongly -- an argument that
      # arrived as a rank inversion and is now a spread.
      cl = ""; gp = ""
      for (i = 1; i <= nh; i++) {
        h = order[i]; if (!(h in ceil8) || !(h in peak)) continue
        lo = ""; hi = ""; nj = 0
        for (j = 1; j <= n; j++) if ((h, R[j]) in frr) {
          v = frr[h, R[j]] + 0; nj++
          if (lo == "" || v < lo) lo = v; if (hi == "" || v > hi) hi = v
        }
        if (lo == "") continue
        cl = cl (cl == "" ? "" : "; ") sprintf("%s %.1f-%.1f%%", model[h], lo, hi)
        gp = gp (gp == "" ? "" : ", ") sprintf("%.0f%%", 100 * (ceil8[h] + 0) / (8 * peak[h]))
      }
      sub(/, ([^,]*)$/, " and&", gp); sub(/ and, /, " and ", gp)
      if (cl != "")
        printf "Measured this run, as a share of each host'"'"'s own %s-thread ceiling: %s. Those ceilings are %s of 8x each host'"'"'s own 1-thread peak -- a different factor per host, which is why the retired %sx cross-host ratio could rank a host that kept more of its own silicon below one that kept less. The ceilings'"'"' own rates are deliberately not republished here: nothing in the table above re-measures them, and a rate no instrument re-checks is a claim rather than a measurement (§7 rule 7, and criterion 9 is what noticed). They are in the gate log this caption names, and publishing them here means first making them re-measured rows.\n\n", (nt == "" ? "8" : nt), cl, gp, retired > "/dev/stderr"
    }
    # The bars in words, not as a ">= %s" with a hole in it. Either may be deferred to
    # its own measurement and both have been, at different times and for the same
    # reason (#37 then #6), so "deferred" is a state this sentence can be in.
    jb = (cf == "" \
      ? sprintf("the judged routines are reported against each host'"'"'s own measured %s-thread ceiling with no fraction in force (#6)", (nt == "" ? "8" : nt)) \
      : sprintf("the judged routines must reach %s%% of each host'"'"'s own measured %s-thread ceiling (#6)", cf, (nt == "" ? "8" : nt)))
    # The same hole, in the same sentence, for the same reason: the bar for this class has been
    # deferred once (#37) and suspended once (the 2026-08-22 spread amendment), so ">= x"
    # with nothing in it is a state this printf could reach and did not handle.
    tb = (tf == "" \
      ? "Strsm is reported against its own 1-thread rate with no floor in force (#37)" \
      : sprintf("Strsm must scale >= %sx (#37)", tf))
    if (cf == "" && tf == "" && nl + nn + ns > 0) {
      # The log disagrees with this tree about whether a bar exists. The constants readback
      # above compares this script with gate-p5.sh and cannot see this: the rows come from a
      # LOG, and a log written when a bar was in force carries FAIL verdicts no suspended bar
      # could have produced. Refuse rather than publish either sentence -- "none was judged"
      # over evidence of judging is the worse of the two lies available here.
      printf "readme-numbers: both bars are suspended in this tree, but %s shortfall verdict(s) appear in %s -- that log was judged by bars this tree does not have, so no caption over it can be true\n", nl + nn + ns, FILENAME > "/dev/stderr"
      exit 3
    }
    if (cf == "" && tf == "") {
      # BOTH bars suspended: "clears the bars" would be vacuously true and would read as a
      # pass, which is the one sentence this caption may not print -- a check that could not
      # have come out otherwise is not evidence (DESIGN.md §5 rule 8). No row can fail here,
      # so the shortfall clauses below are unreachable and are not the reason none printed.
      #
      # UNEXERCISED BY ANY ARCHIVED LOG, stated rather than implied (§5 rule 12): every log in
      # build/ predates the suspension and carries a shortfall, so each one now takes the
      # refusal above instead. This string was rendered against one of them before that
      # refusal landed, which proves the formatting and not the reachability. The run this
      # branch exists for is the era-founding one, and that log will not reach it either until
      # a BASELINE verdict line can be parsed: those lines carry no scaling clause, so this
      # program refuses the whole log, which is the blocker on regenerating the README as
      # medians over this era (#6). Found 2026-08-22 by driving this branch, not by reading it.
      printf "NONE of the %d routine-host pairs those %d rows form was judged: both bars scripts/gate-p5.sh would enforce are suspended for re-derivation from this era (%s; %s), so every number above is REPORTED and the absence of a shortfall below is not a pass. The %sx cross-host scaling floor these numbers were once judged against is retired -- it was rank-ordered against per-core efficiency, refusing the host that kept the most of its core peak.\n", nr, nrow, jb, tb, retired > "/dev/stderr"
    } else if (nl + nn + ns == 0) {
      printf "Every one of the %d routine-host pairs those %d rows form clears the bars scripts/gate-p5.sh enforces, net of confidence intervals: %s, and %s. The %sx cross-host scaling floor these numbers were once judged against is retired -- it was rank-ordered against per-core efficiency, refusing the host that kept the most of its core peak.\n", nr, nrow, jb, tb, retired > "/dev/stderr"
    } else {
      printf "%d of the %d routine-host pairs those %d rows form %s not clear the bars scripts/gate-p5.sh enforces (%s; %s; both judged net of confidence intervals). ", nl + nn + ns, nr, nrow, (nl + nn + ns == 1 ? "does" : "do"), jb, tb > "/dev/stderr"
      if (ns > 0) {
        s = ""; for (k = 1; k <= ns; k++) s = s (k > 1 ? "; " : "") short[k]
        printf "%d of the judged routines %s short of %s own host'"'"'s ceiling: %s. ", ns, (ns == 1 ? "falls" : "fall"), (ns == 1 ? "its" : "their"), s > "/dev/stderr"
      }
      if (nl > 0) {
        s = ""; for (k = 1; k <= nl; k++) s = s (k > 1 ? "; " : "") low[k]
        printf "%d %s below it outright: %s. ", nl, (nl == 1 ? "sits" : "sit"), s > "/dev/stderr"
      }
      if (nn > 0) {
        s = ""; for (k = 1; k <= nn; k++) s = s (k > 1 ? "; " : "") near[k]
        printf "%d %s it on the point estimate and %s only net of CI, which is a verdict decided by the measurement precision rather than by the parallel nest: %s. ", nn, (nn == 1 ? "clears" : "clear"), (nn == 1 ? "misses" : "miss"), s > "/dev/stderr"
      }
      print "These are published shortfalls against bars checked on every gate run, not regressions against an earlier reading." > "/dev/stderr"
    }
  }
' "$LOG" 2>build/.readme-numbers.cap)" || { cat build/.readme-numbers.cap >&2; die "the log did not yield a complete block; README.md is untouched"; }

CAPTION="$(awk 'f {print} /^CAPTION:$/ {f=1}' build/.readme-numbers.cap)"
[[ -n "$CAPTION" ]] || die "no caption was derived, and a block without its disclosure is the drift this script exists to end"

if [[ "$DRY" == "-n" ]]; then
  printf '%s\n%s\n%s\n\n%s\n%s\n%s\n' "$NUM_BEGIN" "$BLOCK" "$NUM_END" "$CAP_BEGIN" "$CAPTION" "$CAP_END"
  exit 0
fi

for m in "$NUM_BEGIN" "$NUM_END" "$CAP_BEGIN" "$CAP_END"; do
  grep -qF "$m" README.md || die "README.md has no \`$m\` marker, so there is nothing to replace and a blind append would publish a second table"
done

splice() { # splice BEGIN END BODY-FILE
  # The body arrives as a FILE, not as -v: this awk rejects a literal newline in a -v
  # assignment ("awk: newline in string"), so the -n path printed a 24-row block while
  # the in-place path could not write one. The `||` cleanup matters for the same reason
  # the `&&` does -- a half-written README.md.new left beside README.md is the next
  # reader's ambiguity, and truncating the published table is the one outcome here
  # worse than failing.
  awk -v b="$1" -v e="$2" -v f="$3" '
    index($0, b) { print; while ((getline l < f) > 0) print l; close(f); inb = 1; next }
    index($0, e) { inb = 0 }
    !inb { print }' README.md > README.md.new && mv README.md.new README.md \
    || { rm -f README.md.new; die "splicing $1 failed; README.md is unchanged"; }
}
printf '%s\n' "$BLOCK"   > build/.readme-numbers.block
printf '%s\n' "$CAPTION" > build/.readme-numbers.caption
splice "$NUM_BEGIN" "$NUM_END" build/.readme-numbers.block
splice "$CAP_BEGIN" "$CAP_END" build/.readme-numbers.caption
# Counted on the threads column, not on a leading "| [A-Z]", which also matches the
# "| CPU | benchmark |" header and reported 25 rows for 24.
echo "readme-numbers: README.md rewritten from $LOG ($(printf '%s\n' "$BLOCK" | grep -cE '^\| .* \| [0-9]+ \| ') rows)"
