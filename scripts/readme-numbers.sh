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
# measured with. STRSM_FLOOR was suspended beside it, made era-scoped by the same ruling.
# BOTH ARE TYPED AGAIN 2026-08-22 from the founding campaign's take four, recomputed under
# #116: 44.2 from six admissible rows and 6.067x from three judged hosts. gate-p5.sh carries
# both derivations and is the authority; restated here in the same commit because the check
# below reads those lines back verbatim, so this is a second edit and not a second decision.
# 6.067x is not SCALE_FLOOR_RETIRED coming back — it lands within 1.1% of it by coincidence,
# and the caption below names the retired one AS retired, which is why the check reads all
# three. The published shares this caption governs are regenerated separately, as medians
# over this era's archives (#6): a bar and the rows it judges are not one act.
CEIL_FRACTION=44.2
STRSM_FLOOR=6.067
SCALE_FLOOR_RETIRED=6.0
ROUTINES='Sgemm Ssyrk Ssymm Strsm'

die() { echo "readme-numbers: $*" >&2; exit 1; }

# MORE THAN ONE LOG IS THE NORMAL CASE (#6): the published rows are rule-16 medians over an
# era's archives, so the estimator needs the whole era at once. One log still works and is
# still N=1 -- stated as N=1, because the point of naming the estimator per row is that a
# reader can tell one draw from a median without counting this script's arguments.
LOGS=(); DRY=""
for a in "$@"; do
  if [[ "$a" == "-n" ]]; then DRY="-n"; else LOGS+=("$a"); fi
done
(( ${#LOGS[@]} > 0 )) || die "usage: $0 <gate-p5 log> [<gate-p5 log> ...] [-n]"
for l in "${LOGS[@]}"; do [[ -r "$l" ]] || die "cannot read $l"; done

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
         -v retired="$SCALE_FLOOR_RETIRED" '
  function strip(s) { gsub(/\033\[[0-9;]*m/, "", s); return s }
  function bn(p) { sub(/^.*\//, "", p); return p }
  # The estimator, named in the cell beside the rate it produced. N=1 says "one draw" in
  # words rather than "median of 1", which is true but reads as a pooled figure.
  function est(k) { return (k == 1 ? "one draw (N=1)" : sprintf("median of N=%d archives", k)) }

  # Rule-16 median over the era'"'"'s archives, pooled over the FILE LIST rather than over a
  # draw counter: the array is keyed (..., FILENAME) throughout, so the pool is exactly the
  # set of logs that carried the key and a log missing a row contributes nothing instead of
  # shifting every later draw'"'"'s index. N is 2 at the founding, so an insertion sort is the
  # whole algorithm; the even case averages the two middles rather than silently picking one,
  # which is the difference between a median and a preference for whichever log was typed
  # first. Every published rate goes through here even at N=1 -- one code path, so there is
  # no "single log" special case to diverge from the pooled one. Sets MEDN so a caller can
  # state the estimator'"'"'s N beside the number instead of assuming it.
  function med(a, kp,   i, j, c, s, t) {
    c = 0
    for (i = 1; i <= nf; i++) if ((kp, flist[i]) in a) { s[++c] = a[kp, flist[i]] + 0 }
    MEDN = c
    if (c == 0) return ""
    for (i = 2; i <= c; i++) { t = s[i]; for (j = i - 1; j >= 1 && s[j] > t; j--) s[j + 1] = s[j]; s[j + 1] = t }
    return (c % 2) ? s[(c + 1) / 2] : (s[c / 2] + s[c / 2 + 1]) / 2
  }

  # A verdict line'"'"'s PREFIX and its PROSE can disagree about whether anything judged the
  # row, and the prefix is the one that lies. Take four (rev 6ba6566) prints
  #   PASS [keel-zen5] Sgemm reaches 48.0% ... -- measured and REPORTED, NO FRACTION IN FORCE (#6)
  # where PASS means only that nothing failed, and nothing failed because no bar existed:
  # 44.2% and 6.067x were derived FROM that run. So the prose outranks the prefix. Reading
  # the prefix alone republished those 9 rows as clearing bars that postdate them -- the
  # mirror of the case guarded below, which is bars empty here and shortfalls in the log.
  # One function for BOTH verdict shapes, because a fix to one of two classifiers is a fix
  # to half the logs. Positive control: the phrase appears 10x in take four and 0x in the
  # confirmation log, so this cannot move a judged run'"'"'s caption, and does not.
  function vclass(line, rest) {
    if (rest ~ /NO FRACTION IN FORCE|NO FLOOR IN FORCE/) return "REPORTED"
    if (line ~ /^ *BASELINE/) return "BASELINE"
    if (line ~ /^ *FAIL/)     return "FAIL"
    if (line ~ /^ *PASS/)     return "PASS"
    return "OTHER"
  }

  # Every line the gate emits per host is tagged [host.local]; the provenance line
  # is the one whose first field after the tag is the CPU model.
  # PER-FILE, not per-run: with two logs every array keyed only by (host, routine) has the
  # second file overwriting the first, so provenance would depend on argument order and the
  # rule-10 identity checks below would compare a value against one from another machine-day.
  FNR == 1 { if (!(FILENAME in fseen)) { fseen[FILENAME] = 1; flist[++nf] = FILENAME } }

  {
    line = strip($0)
    # Run-wide provenance, before the host-tag guard or the guard skips it. The sha comes off
    # a sample path: "green on this commit (sha)" prints only when GREEN, so reds lost the rev.
    if (match(line, /bench-gate-p5-[0-9a-f]{7,40}-/)) { frev[FILENAME] = substr(line, RSTART + 14, RLENGTH - 15) }
    if (match(line, /^-- scaling at 8 cores on [0-9]+\^3/)) { nsz = substr(line, RSTART + 25, RLENGTH - 27) }
    if (era == "" && match(line, /in era [a-z0-9]+/)) { era = substr(line, RSTART + 7, RLENGTH - 7) }
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
      # CLEARED FIRST. t1/t8 were assigned only on a match and never reset, so a line whose
      # "8 threads" clause failed to parse published the 8-thread rate of the PREVIOUS routine
      # as this one -- a wrong number with no missing-data symptom anywhere. Reachable by any
      # future edit to the rate line the gate emits, which is exactly the change nothing here
      # would catch. Now a failed parse refuses the log instead of borrowing the rate above it.
      t1 = ""; t8 = ""
      if (match(rest, /1 thread [0-9.]+/))  { t1 = substr(rest, RSTART + 9, RLENGTH - 9) }
      if (match(rest, /8 threads [0-9.]+/)) { t8 = substr(rest, RSTART + 10, RLENGTH - 10) }
      if (t1 == "" || t8 == "")
        { printf "readme-numbers: [%s] %s rate line yielded no %s-thread rate in %s\n", host, r, (t1 == "" ? "1" : "8"), bn(FILENAME) > "/dev/stderr"; bad = 1 }
      one[host, r, FILENAME] = t1; eight[host, r, FILENAME] = t8
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
      vc = vd[host, r, FILENAME] = vclass(line, rest)
      if (vc == "PASS" || vc == "FAIL") hasjudged[FILENAME] = 1
      # Both ratios must parse as numbers. A ratio that came out as prose would be
      # published as prose, and the row it describes is the one a reader checks.
      if (pt !~ /^[0-9]+\.?[0-9]*$/ || ci !~ /^[0-9]+\.?[0-9]*$/)
        { printf "readme-numbers: [%s] %s verdict line did not yield two ratios (point=%s, net=%s)\n", host, r, pt, ci > "/dev/stderr"; bad = 1 }
      ptr[host, r, FILENAME] = pt; cir[host, r, FILENAME] = ci
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
      vc = vd[host, r, FILENAME] = vclass(line, rest)
      if (vc == "PASS" || vc == "FAIL") hasjudged[FILENAME] = 1
      if (match(rest, /measured [0-9]+-thread/)) { nt = substr(rest, RSTART + 9, RLENGTH - 16) }
      # Same refusal as above, extended to the two new numbers: a fraction or a
      # ceiling that came out as prose would be published as prose. A BASELINE row
      # carries only the first two -- nothing judged it, so it has no ratio to print --
      # and demanding four refused the WHOLE log, which is why this generator could not
      # read the era it was written to publish. Predicted in this file 2026-08-22 and
      # confirmed by running it against the founding confirmation log 2026-08-23.
      want = (vc == "BASELINE") ? 2 : 4
      if (fr !~ /^[0-9]+\.?[0-9]*$/ || ce !~ /^[0-9]+\.?[0-9]*$/ || (want == 4 && (pt !~ /^[0-9]+\.?[0-9]*$/ || ci !~ /^[0-9]+\.?[0-9]*$/)))
        { printf "readme-numbers: [%s] %s ceiling verdict line did not yield %d numbers (frac=%s, ceiling=%s, point=%s, net=%s)\n", host, r, want, fr, ce, pt, ci > "/dev/stderr"; bad = 1 }
      frr[host, r, FILENAME] = fr; ptr[host, r, FILENAME] = pt; cir[host, r, FILENAME] = ci
      # One ceiling per host, printed once per judged routine: three printings of one
      # measurement are one witness (§5 rule 10), so they cross-check and never corroborate.
      # SCOPED TO THE FILE, because across archives two ceilings for one host is the normal
      # case and not a defect -- zen5 measured 2291 in take four and 2292 in the confirmation
      # run. Left unscoped, the check that exists to catch one run contradicting itself would
      # have failed the era for the ordinary fact that a measurement moved between days.
      if (((host, FILENAME) in ceil8) && ceil8[host, FILENAME] != ce)
        { printf "readme-numbers: [%s] prints two 8-thread ceilings in one run (%s), %s and %s\n", host, bn(FILENAME), ceil8[host, FILENAME], ce > "/dev/stderr"; bad = 1 }
      ceil8[host, FILENAME] = ce
    }

    # peak, printed once per routine per host, beside the 8-thread percent the gate computes
    if (match(rest, /% of 8x the single-thread avx512 peak \([0-9.]+ GFLOP\/s\)/)) {
      r = rest; sub(/:.*$/, "", r)
      pc = rest; sub(/^.*: /, "", pc); sub(/%.*$/, "", pc)
      pk = rest; sub(/^.*peak \(/, "", pk); sub(/ GFLOP.*$/, "", pk)
      # Four printings of one measurement are ONE witness (DESIGN.md §5 rule 10), so
      # they are used as a consistency check and never as corroboration. Scoped to the file
      # for the same reason as the ceiling above: within a run this must hold, across the
      # era it must not be required to.
      if (((host, FILENAME) in peak) && peak[host, FILENAME] != pk)
        { printf "readme-numbers: [%s] prints two single-thread peaks in one run (%s), %s and %s\n", host, bn(FILENAME), peak[host, FILENAME], pk > "/dev/stderr"; bad = 1 }
      peak[host, FILENAME] = pk; gpc[host, r, FILENAME] = pc
    }
  }

  END {
    if (bad) exit 3
    n = split(routines, R, " ")
    if (nh == 0) { print "readme-numbers: the log names no host with a provenance line" > "/dev/stderr"; exit 3 }

    # WHICH LOG RENDERED THE VERDICTS. Rates pool across the era; verdicts do not, because
    # a verdict belongs to the run whose gate rendered it and averaging two is meaningless.
    # Exactly one input may be a judged log. Zero is legal -- the founding run is judged by
    # nothing and the caption has a branch saying so -- but TWO is refused rather than
    # resolved: with two, whichever file awk read last would silently own the disclosure,
    # and "the verdicts came from the log you happened to type second" is not provenance.
    # This is the file'"'"'s own rule at the constants: a bar and the rows it judges are not
    # one act.
    jf = ""; njf = 0
    for (i = 1; i <= nf; i++) if (flist[i] in hasjudged) { njf++; jf = flist[i] }
    if (njf > 1) {
      printf "readme-numbers: %d of the %d logs carry bar verdicts, so the disclosure has no single provenance; pass one judged log plus any number of unjudged archives\n", njf, nf > "/dev/stderr"
      exit 3
    }
    # No judged log: the verdict classes are still needed (REPORTED/BASELINE both live in
    # the caption), so read them from the last archive, which is the only one there is to
    # read them from. Named here rather than defaulted silently.
    if (jf == "") jf = flist[nf]
    # The verdict AND the figures that state it. frr/ptr/cir are the numbers the shortfall
    # strings quote, so they belong to the same run as the verdict quoting them -- pooling
    # those would print a median ratio inside a sentence about one run'"'"'s FAIL.
    for (kk in vd)  { split(kk, K, SUBSEP); if (K[3] == jf) verdict[K[1], K[2]] = vd[kk] }
    for (kk in frr) { split(kk, K, SUBSEP); if (K[3] == jf) fr1[K[1], K[2]] = frr[kk] }
    for (kk in ptr) { split(kk, K, SUBSEP); if (K[3] == jf) pt1[K[1], K[2]] = ptr[kk] }
    for (kk in cir) { split(kk, K, SUBSEP); if (K[3] == jf) ci1[K[1], K[2]] = cir[kk] }

    print "| CPU | benchmark | threads | GFLOP/s | denominator |"
    print "| --- | --- | --- | --- | --- |"
    for (i = 1; i <= nh; i++) {
      h = order[i]
      p1 = med(peak, h)
      if (p1 == "") { printf "readme-numbers: [%s] has no measured single-thread peak\n", h > "/dev/stderr"; exit 3 }
      p8 = p1 * 8
      for (j = 1; j <= n; j++) {
        r = R[j]
        a = med(one, h SUBSEP r); an = MEDN
        b = med(eight, h SUBSEP r); bn8 = MEDN
        # A thin block is not a table with one row fewer; it is a published table
        # that silently dropped a host. Same refusal the docs-gen.sh extractions make.
        if (a == "" || b == "") { printf "readme-numbers: [%s] %s has no 1-thread/8-thread pair in these logs\n", h, r > "/dev/stderr"; exit 3 }

        # The 1-thread percent is computed because the gate does not print it. The
        # 8-thread percent the gate DOES print, so it is recomputed and checked --
        # this script auditing its own denominator against the one the instrument used.
        # CHECKED PER ARCHIVE, against that archive'"'"'s own peak and its own printed share:
        # the pooled median matches no single log by construction, so comparing the pooled
        # share against one log'"'"'s printed share would fire on every honest N>1 pool. Same
        # instrument, moved inside the loop that has a denominator it can be right about.
        for (q = 1; q <= nf; q++) {
          f = flist[q]
          if (!((h, r, f) in gpc) || !((h, f) in peak) || !((h, r, f) in eight)) continue
          fc8 = (eight[h, r, f] + 0) / (peak[h, f] * 8) * 100
          d = fc8 - (gpc[h, r, f] + 0); if (d < 0) d = -d
          if (d > 0.15) { printf "readme-numbers: [%s] %s 8-thread share computes to %.1f%% in %s but that run printed %s%%\n", h, r, fc8, bn(f), gpc[h, r, f] > "/dev/stderr"; exit 3 }
        }
        c8 = (b + 0) / p8 * 100
        # N rides on every row, per the ratified repair (#6): the estimator is part of the
        # number. "median of 2" and a lone draw are different claims and a reader cannot tell
        # them apart from the rate alone -- which is the whole defect that repair addressed.
        printf "| %s | %s | 1 | %.4g | %.1f%% of %.4g GFLOP/s, the 1-thread avx512 microkernel peak; %s |\n", \
          model[h], r, a, (a + 0) / p1 * 100, p1, est(an)
        printf "| %s | %s | 8 | %.4g | %.1f%% of %.1f GFLOP/s, that same peak x 8 cores; %s |\n", \
          model[h], r, b, c8, p8, est(bn8)

        # A row with no verdict is not a passing row. Fail closed: the caption would
        # otherwise be silent about exactly the row the gate could not judge.
        # A log from before the #6 ruling has rates and no ceiling verdict, and lands
        # here rather than being republished under a criterion it was never judged by.
        # Re-adjudicating those runs is its own deliverable working from the archived
        # samples, not something this generator may do by silently keeping the old bar.
        if (!((h, r) in verdict)) { printf "readme-numbers: [%s] %s has rates but no scaling verdict in this log, so its disclosure cannot be derived (a pre-2026-08-20 log has no ceiling verdict to publish)\n", h, r > "/dev/stderr"; exit 3 }
        # Counted, not skipped: a BASELINE pair is measured and unjudged, and the caption
        # below subtracts it from the population it says "clears the bars" rather than
        # letting it ride inside that claim. REPORTED is the second way to be unjudged and
        # is counted apart from it, because the two have different causes and a reader owed
        # the reason cannot get it from a merged count: BASELINE means this silicon had no
        # reference artifact in the era, REPORTED means the bar itself did not exist yet.
        if (verdict[h, r] == "BASELINE") nb++
        if (verdict[h, r] == "REPORTED") nu++
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
            short[++ns] = sprintf("%s %s at %s%% of its own measured %s-thread ceiling", model[h], r, fr1[h, r], (nt == "" ? "8" : nt))
          } else {
            # Strsm keeps the ratio bar and so keeps the distinction, which asks for
            # different things: a point estimate already under the floor is a
            # shortfall, while one that clears the floor and fails only net of CI is a
            # verdict decided by the noise the measurement itself carries, and the
            # remedy for that is precision (DESIGN.md §4, line 130) rather than a
            # discussion about the nest.
            if (pt1[h, r] + 0 >= tf + 0) near[++nn] = sprintf("%s %s (%sx, %sx net of CI)", model[h], r, pt1[h, r], ci1[h, r])
            else low[++nl] = sprintf("%s %s at %sx (%sx net of CI)", model[h], r, pt1[h, r], ci1[h, r])
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
    # The judged run leads, and not for style: docs-gen.sh:92 takes the FIRST match and exits,
    # so whichever rev is named first is the one that as-of dates the whole numbers page. That
    # is the judged run, because the verdicts on the page are its verdicts.
    src = ""; jrev = (jf in frev ? frev[jf] : "")
    src = sprintf("`build/%s`", bn(jf))
    for (q = 1; q <= nf; q++) if (flist[q] != jf) {
      src = src sprintf(", `build/%s`", bn(flist[q]))
      others = others (others == "" ? "" : ", ") sprintf("`%s`", (flist[q] in frev ? frev[flist[q]] : "unrecorded"))
    }
    # ONE RUN vs AN ERA is a different provenance claim, so it is a different sentence rather
    # than the same sentence with a plural. At N=1 the old wording is still exactly right and
    # still printed; the pooled wording names the estimator, both revs, and -- the part a
    # reader cannot recover from the table -- that the verdicts come from ONE of the archives
    # while the rates come from all of them.
    if (nf == 1) {
      printf "All %d rows come from one run — `scripts/gate-p5.sh` at rev `%s`, log in %s — at n=%s square, `GOMAXPROCS` pinned to the threads column, %s. The 1-thread and 8-thread rows for a routine are the two arms of that run'"'"'s scaling ratio, so they are directly comparable to each other; rows from different CPUs are not, because the peaks differ.\n\n", \
        nrow, (jrev == "" ? "unrecorded" : jrev), src, (nsz == "" ? "4096" : nsz), \
        (g == "" ? "governor unrecorded" : g == "mixed" ? "governors differing between hosts (see the log)" : "`" g "` governor on every host") > "/dev/stderr"
    } else {
      printf "All %d rows are per-row medians over the %d archived runs of one era — `scripts/gate-p5.sh` at rev `%s` (the judged run, which dates this page) and %s, logs in %s — at n=%s square, `GOMAXPROCS` pinned to the threads column, %s. Each cell names its own N. The 1-thread and 8-thread rows for a routine pool the same archives, so their ratio is a ratio of like estimators; rows from different CPUs are not comparable, because the peaks differ. The verdicts below are the judged run'"'"'s alone — a verdict belongs to the gate that rendered it and two cannot be averaged.\n\n", \
        nrow, nf, (jrev == "" ? "unrecorded" : jrev), others, src, (nsz == "" ? "4096" : nsz), \
        (g == "" ? "governor unrecorded" : g == "mixed" ? "governors differing between hosts (see the log)" : "`" g "` governor on every host") > "/dev/stderr"
    }
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
        h = order[i]
        # The ceiling and the peak are both pooled, and the RATIO of the two is what this
        # prints -- so both must come from the same pool or the gap between the denominators
        # would be a gap between eras. med() over the same file list gives that for free.
        hc = med(ceil8, h); hp = med(peak, h)
        if (hc == "" || hp == "") continue
        lo = ""; hi = ""
        # The shares are the JUDGED run'"'"'s, not the pool'"'"'s: they are the readings the bar
        # compared against, and a median share would not be the number anything judged.
        for (j = 1; j <= n; j++) if ((h, R[j]) in fr1) {
          v = fr1[h, R[j]] + 0
          if (lo == "" || v < lo) lo = v; if (hi == "" || v > hi) hi = v
        }
        if (lo == "") continue
        cl = cl (cl == "" ? "" : "; ") sprintf("%s %.1f-%.1f%%", model[h], lo, hi)
        gp = gp (gp == "" ? "" : ", ") sprintf("%.0f%%", 100 * hc / (8 * hp))
      }
      sub(/, ([^,]*)$/, " and&", gp); sub(/ and, /, " and ", gp)
      if (cl != "")
        # "In the judged run", not "this run": these shares are the readings the bar compared
        # against, so they come from one archive while the table above pools every archive.
        # Under pooling "this run" has no referent, and the shares would read as medians.
        printf "Measured in the judged run, as a share of each host'"'"'s own %s-thread ceiling: %s. Those ceilings are %s of 8x each host'"'"'s own 1-thread peak -- a different factor per host, which is why the retired %sx cross-host ratio could rank a host that kept more of its own silicon below one that kept less. The ceilings'"'"' own rates are deliberately not republished here: nothing in the table above re-measures them, and a rate no instrument re-checks is a claim rather than a measurement (§7 rule 7, and criterion 9 is what noticed). They are in the gate %s this caption names, and publishing them here means first making them re-measured rows.\n\n", (nt == "" ? "8" : nt), cl, gp, retired, (nf == 1 ? "log" : "logs") > "/dev/stderr"
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
    # ONE SPELLING OF EACH UNJUDGED DISCLOSURE, shared by every branch that can reach it.
    # The era is read off the log rather than named here: a generator that hardcodes the
    # era would keep publishing "pinned8" into the era after it.
    bc = (nb == 0 ? "" : sprintf(" A further %d of those pairs %s RECORDED as a candidate baseline in era %s and judged by nothing, so those rows are published as measurements and not as passes (#6).", nb, (nb == 1 ? "is" : "are"), (era == "" ? "the one this log names no id for" : era)))
    # Second unjudged class, second sentence: these rows ran BEFORE the bars existed, so no
    # verdict is available to publish at any strictness. Saying so is not optional -- their
    # gate lines are prefixed PASS, and that prefix is what made this caption claim they
    # cleared bars derived from them.
    bc = bc (nu == 0 ? "" : sprintf(" A further %d %s measured under no bar at all -- that run predates the bars it would be judged by, which were derived from it -- so %s no verdict.", nu, (nu == 1 ? "pair was" : "pairs were"), (nu == 1 ? "it is REPORTED and carries" : "they are REPORTED and carry")))
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
    } else if (nl + nn + ns == 0 && nr - nb - nu == 0) {
      # NOTHING WAS JUDGED, yet bars are in force in this tree -- so the branch above cannot
      # fire and the "N of M clear" branches would headline "0 of the 12 pairs clear the bars",
      # which reads as total failure where in truth nothing was tested. A zero numerator over
      # an untested population is not a result, and printing it as one is the same false
      # attribution this commit removes, one branch over. Reachable with the founding run as
      # sole input, which is exactly how it was found.
      printf "NONE of the %d routine-host pairs those %d rows form was judged against the bars scripts/gate-p5.sh now enforces (%s; %s), so no number above is a pass and the absence of a shortfall is not one either.%s The %sx cross-host scaling floor these numbers were once judged against is retired -- it was rank-ordered against per-core efficiency, refusing the host that kept the most of its core peak.\n", nr, nrow, jb, tb, bc, retired > "/dev/stderr"
    } else if (nl + nn + ns == 0 && nb == 0 && nu == 0) {
      printf "Every one of the %d routine-host pairs those %d rows form clears the bars scripts/gate-p5.sh enforces, net of confidence intervals: %s, and %s. The %sx cross-host scaling floor these numbers were once judged against is retired -- it was rank-ordered against per-core efficiency, refusing the host that kept the most of its core peak.\n", nr, nrow, jb, tb, retired > "/dev/stderr"
    } else if (nl + nn + ns == 0) {
      # EVERY JUDGED PAIR CLEARED, and some pairs were not judged at all. "Every one of the
      # N pairs clears the bars" over a population containing BASELINE rows is the vacuous
      # pass §5 rule 8 forbids -- it would read as 12 verdicts where the gate rendered 9.
      # So the judged count is the subject and the recorded count is named beside it, in
      # the SAME words the shortfall branch below uses, because two spellings of one
      # disclosure is the drift this whole script exists to end.
      printf "%d of the %d routine-host pairs those %d rows form clear the bars scripts/gate-p5.sh enforces, net of confidence intervals: %s, and %s.%s The %sx cross-host scaling floor these numbers were once judged against is retired -- it was rank-ordered against per-core efficiency, refusing the host that kept the most of its core peak.\n", nr - nb - nu, nr, nrow, jb, tb, bc, retired > "/dev/stderr"
    } else {
      # nr, not nr - nb: "the N pairs those M rows form" is a structural claim about the
      # table -- 24 rows form 12 pairs however many of them anything judged -- so subtracting
      # the baselines here would make the phrase false while the branch above kept it true,
      # which is one caption reading two denominators out of the same six words.
      printf "%d of the %d routine-host pairs those %d rows form %s not clear the bars scripts/gate-p5.sh enforces (%s; %s; both judged net of confidence intervals).%s ", nl + nn + ns, nr, nrow, (nl + nn + ns == 1 ? "does" : "do"), jb, tb, bc > "/dev/stderr"
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
' "${LOGS[@]}" 2>build/.readme-numbers.cap)" || { cat build/.readme-numbers.cap >&2; die "the logs did not yield a complete block; README.md is untouched"; }

CAPTION="$(awk 'f {print} /^CAPTION:$/ {f=1}' build/.readme-numbers.cap)"
[[ -n "$CAPTION" ]] || die "no caption was derived, and a block without its disclosure is the drift this script exists to end"

# A HOST LEAVES THE BLOCK WITHOUT ANY ROW FAILING. Rows are medians over one era's
# archives, so a host with no archive in that era is not thin -- it is absent, and
# absence is the one change 24 green rows cannot report. Derived against the LAST
# PUBLISHED block, not the working copy: comparing to README.md on disk would print
# this on the first run and drop it on the second, which is the silent loss it guards.
# Fails closed -- an unreadable HEAD says so rather than saying nobody left.
models() { awk -F' *\\| *' '/^\| .* \| [0-9]+ \| /{print $2}' | sort -u; }
if OLD_README="$(git show HEAD:README.md 2>/dev/null)" && [[ -n "$OLD_README" ]]; then
  GONE="$(comm -23 <(printf '%s\n' "$OLD_README" | models) <(printf '%s\n' "$BLOCK" | models) \
    | awk '{s = s (NR > 1 ? "; " : "") $0} END {printf "%s", s}')"
  [[ -z "$GONE" ]] || CAPTION="$CAPTION"$'\n\n'"In the previously published block and not in this one: $GONE. A host with no archive in an era cannot be a median over it, so it leaves the table rather than being marked inside it; the era's stated exclusions are in \`scripts/measurement-eras.tsv\`."
else
  CAPTION="$CAPTION"$'\n\n'"Whether a host left the block since the last published one is undetermined here: \`git show HEAD:README.md\` was unreadable."
fi

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
echo "readme-numbers: README.md rewritten from ${#LOGS[@]} log(s) — ${LOGS[*]} ($(printf '%s\n' "$BLOCK" | grep -cE '^\| .* \| [0-9]+ \| ') rows)"
