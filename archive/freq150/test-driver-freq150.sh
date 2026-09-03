#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# Exercises driver-freq150.sh's decision procedures before it is launched. Five groups, each for a
# different reason:
#
#   1. THE PEDESTAL, CROSS-CHECKED IN TWO LANGUAGES. The guard's bound was derived by
#      derive-pedestal.py in python; the live guard recomputes the same exponentially weighted
#      average in awk. Two implementations of one quantity agree by luck until something compares
#      them, and this one compares them on the tracked core148 timeline instant by instant. Note
#      what is NOT retyped: the log parsing and the arm timeline come from the python module, so the
#      only thing under test is the arithmetic.
#   2. THE QUIETNESS GATE, DRIVEN RED. A precondition only ever seen to pass is an unread witness.
#      Every refusal branch fires here, including the two failure-to-read branches no healthy run
#      reaches, and including the case the guard this one replaces would have PASSED -- a cold host
#      at 1.01, which is the whole point of subtracting a pedestal.
#   3. THE SAMPLER CPU CHOICE, against topologies janus does not have, including the one where there
#      is nowhere left to put the sampler.
#   4. THE SAMPLER ITSELF, ON THE HOST. Started, its trace measured for growth and format, its
#      process counted with the bracket pattern (1 while running, 0 after the kill). This is the
#      instrument's own applied-witness exercised on purpose, before a 3.5-hour run depends on it.
#   5. THE ARM TABLE AND SEQUENCES, cross-checked against predictions-freq150.py. The registration
#      is the authority for what this run measures; the driver is a second copy of it.
#
# Run: bash archive/freq150/test-driver-freq150.sh
set -uo pipefail
cd "$(dirname "$0")/../.." || exit 3

# Overridable so the cross-checks can be driven RED against a deliberately perturbed copy. A
# comparison that has only ever agreed is not known to be a comparison.
DRIVER="${DRIVER:-archive/freq150/driver-freq150.sh}"
REG="${REG:-archive/freq150/predictions-freq150.py}"
HOST="${HOST:-janus.local}"
pass=0; fail=0
ok()  { pass=$((pass+1)); printf '  PASS %s\n' "$1"; }
bad() { fail=$((fail+1)); printf '  FAIL %s\n' "$1"; }

# Functions taken from the driver by extraction rather than retyped: a retyped copy tests the copy.
extract() {
  awk -v f="$1" '
    $0 ~ "^" f "\\(\\)[ ]*\\{" { if ($0 ~ /\}[ ]*$/) { print; exit } inb = 1 }
    inb { print }
    inb && /^\}$/ { exit }
  ' "$DRIVER"
}
SELF_N=0.98
QUIET_FOREIGN_MAX=0.27
OWN_WINDOWS=""
T0=0
eval "$(extract own_add)"
eval "$(extract pedestal_at)"
eval "$(extract quiet_violation)"
eval "$(extract expand_online)"
eval "$(extract pick_scpu)"
eval "$(extract env_for)"
eval "$(extract mask_for)"
eval "$(extract cores_for)"
for f in own_add pedestal_at quiet_violation expand_online pick_scpu env_for mask_for cores_for; do
  declare -F "$f" >/dev/null || { echo "FATAL: could not extract $f() from $DRIVER"; exit 2; }
done
echo "extracted 8 functions from $DRIVER"

echo
echo "== 1. the pedestal, bash-vs-python on the tracked core148 timeline =="
# The python side supplies the timeline AND its own answer; bash recomputes only the average.
CASES="$(python3 - <<'PY'
import importlib.util as u
s = u.spec_from_file_location("d", "archive/freq150/derive-pedestal.py")
m = u.module_from_spec(s); s.loader.exec_module(m)
samples, arms = m.parse("archive/core148/core148-97a21f4.log")
kept, _ = m.dedup(samples)
t0 = min(x["t"] for x in samples)
print("T0 %d" % t0)
for _, a, b in arms:
    print("W %d %d" % (a, b))
for k in kept:
    print("C %d %.4f %s" % (k["t"], 0.98 * m.ewma(k["t"], arms, t0), k["label"].replace(" ", "_")))
PY
)"
if [[ -z "$CASES" ]]; then
  bad "the python side produced no cases at all"
else
  T0="$(awk '/^T0 /{print $2}' <<<"$CASES")"
  while read -r _ a b; do own_add "$a" "$b"; done < <(awk '/^W /' <<<"$CASES")
  echo "  timeline: T0=$T0, $(awk '/^W /' <<<"$CASES" | wc -l | tr -d ' ') arm windows"
  n=0
  while read -r _ t want label; do
    n=$((n+1))
    got="$(pedestal_at "$t")"
    if awk -v g="$got" -v w="$want" 'BEGIN{exit !((g-w<0.0005)&&(w-g<0.0005))}'; then
      ok "instant $label: awk $got == python $want"
    else
      bad "instant $label: awk $got != python $want"
    fi
  done < <(awk '/^C /' <<<"$CASES")
  [[ "$n" -ge 9 ]] || bad "expected at least 9 instants from core148, compared $n"
  # ...and the comparison must be able to disagree. A tolerance test that has only ever seen equal
  # inputs is not known to be a test.
  #
  # THE INSTANT HERE IS THE LAST, NOT THE FIRST, and that is the whole point of this sub-test. The
  # first instant is `before aref` on a cold host, where the pedestal is 0.0000 -- and 0.50 x 0 and
  # 0.98 x 0 are the same number, so a perturbation applied there is invisible whether or not
  # pedestal_at reads SELF_N at all. Driven against the first instant this assertion FAILED while
  # the code was correct: a witness chosen where the quantity cannot move certifies nothing.
  probe_t="$(awk '/^C /{t=$2} END{print t}' <<<"$CASES")"
  bad_self_n="$(SELF_N=0.50 pedestal_at "$probe_t")"
  right="$(pedestal_at "$probe_t")"
  if [[ "$bad_self_n" != "$right" ]]; then
    ok "a perturbed SELF_N moves the pedestal ($right -> $bad_self_n), so the comparison can fail"
  else
    bad "SELF_N=0.50 gave the same pedestal as 0.98; pedestal_at is ignoring it"
  fi
fi

echo
echo "== 2. the quietness gate, every branch =="
# want=quiet expects return 1 (no violation); want=refuse expects return 0 (violation).
sample() { printf 'uptime: up 42 days\nloadavg: %s 1/703 2556983\ngovernor: performance\n' "$1"; }
check() {
  local want="$1" desc="$2" why rc
  why="$(quiet_violation)"; rc=$?
  if [[ "$want" == "refuse" && "$rc" -eq 0 ]] || [[ "$want" == "quiet" && "$rc" -eq 1 ]]; then
    ok "$desc -> $want ($why)"
  else
    bad "$desc -> wanted $want, got rc=$rc ($why)"
  fi
}
# A saturated timeline (one long window ending now) and a cold one (no windows at all).
now="$(date +%s)"
OWN_WINDOWS=""; T0=$((now-4000)); own_add "$((now-3900))" "$now"
HS="$(sample '1.01 1.01 1.00')"; check quiet  "saturated host at 1.01"
HS="$(sample '1.01 1.28 1.00')"; check refuse "saturated host at 1.28 (0.30 of foreign work)"
HS="$(sample '0.99 2.17 1.83')"; check refuse "the ONE tracked co-tenant excursion, 2.17"
OWN_WINDOWS=""; T0="$now"
HS="$(sample '1.01 1.01 1.00')"; check refuse "COLD host at 1.01 -- test 3's flat 1.25 bound passed this"
HS="$(sample '0.08 0.17 0.11)')"; check quiet  "cold host at 0.17, the tail of an earlier launch"
HS="$(printf 'uptime: up 42 days\ngovernor: performance\n')"; check refuse "no loadavg line at all"
HS="$(printf 'loadavg: banana pear 1.00 1/7 9\n')"; check refuse "unparsable loadavg fields"
# The branch where the pedestal itself fails to compute. No healthy run reaches it, and a guard that
# silently treated a missing pedestal as zero would refuse everything; one that treated it as
# saturated would pass everything.
_saved="$(declare -f pedestal_at)"
pedestal_at() { printf ''; }
HS="$(sample '1.01 1.01 1.00')"; check refuse "pedestal_at returns nothing"
eval "$_saved"
HS="$(sample '1.01 1.01 1.00')"; check refuse "gate restored after the override (cold, still 1.01)"

echo
echo "== 3. the sampler cpu, on topologies this host does not have =="
scase() {  # $1 online, $2 forbidden, $3 expected ("" = must refuse)
  local got; got="$(pick_scpu "$1" "$2")"
  if [[ "$got" == "$3" ]]; then
    ok "online=[$1] forbidden=[$2] -> ${got:-<refuse>}"
  else
    bad "online=[$1] forbidden=[$2] -> '${got}', wanted '${3:-<refuse>}'"
  fi
}
scase "0-31"     "0 1 16 17" "2"    # janus
scase "0,1,4-6"  "0 1"       "4"    # mixed list-and-range form, which sysfs also writes
scase "0-3"      "0 1 2 3"   ""     # nowhere to put it: must refuse, not fall back to a measured cpu
scase "0-1"      "0 1"       ""
scase "2-3"      "0 1 16 17" "2"
[[ "$(expand_online '0,1,4-6' | tr '\n' ' ')" == "0 1 4 6 5 "* || "$(expand_online '0,1,4-6' | tr '\n' ' ')" == "0 1 4 5 6 " ]] \
  && ok "expand_online '0,1,4-6' = $(expand_online '0,1,4-6' | tr '\n' ' ')" \
  || bad "expand_online '0,1,4-6' = $(expand_online '0,1,4-6' | tr '\n' ' ')"

echo
echo "== 4. the sampler itself, on $HOST =="
# Not skippable. The whole run's frequency data comes from this script, and "it will probably work"
# is the position rule 26 exists to refuse. If the host is unreachable this test FAILS rather than
# defers, because a driver launched without this check is a driver whose instrument is unwitnessed.
SPAT='freq150-sampl[e]r'
RDIR="${KEEL_REMOTE_DIR:-/tmp/keel-remote}"
STMP="$(mktemp -d)/freq150-sampler.sh"
# The sampler's source comes out of the DRIVER's heredoc, never retyped.
awk '/^cat >"\$SAMPLER_LOCAL" <<.SAMPLER.$/{f=1;next} f&&/^SAMPLER$/{exit} f' "$DRIVER" >"$STMP"
echo "  extracted $(wc -l <"$STMP" | tr -d ' ') sampler lines from $DRIVER"
if ! bash -n "$STMP"; then bad "the extracted sampler is not valid bash"; fi
if ! ssh -o BatchMode=yes -o ConnectTimeout=10 "$HOST" true 2>/dev/null; then
  bad "$HOST unreachable: the sampler cannot be witnessed, so the driver must not be launched"
else
  # Installed BY scp, the way the driver does it. Not by ssh's stdin: KEEL_SSH_OPTS carries -n, and
  # the driver's first launch shipped an EMPTY sampler that way. So the hash of what landed is
  # asserted here too -- it is the check that caught that, and it belongs in the test that would
  # otherwise be the last thing to notice.
  ssh -o BatchMode=yes "$HOST" "mkdir -p '$RDIR'" >/dev/null 2>&1
  scp -q -o BatchMode=yes "$STMP" "$HOST:$RDIR/freq150-sampler.sh"
  ssh -o BatchMode=yes "$HOST" "chmod +x '$RDIR/freq150-sampler.sh'"
  want_hash="$({ shasum -a 256 "$STMP" 2>/dev/null || sha256sum "$STMP"; } | cut -c1-16)"
  got_hash="$(ssh -o BatchMode=yes "$HOST" "sha256sum '$RDIR/freq150-sampler.sh' | cut -c1-16" | tr -d ' \n')"
  [[ "$got_hash" == "$want_hash" ]] && ok "the sampler that landed on $HOST hashes $got_hash, as shipped" \
                                    || bad "shipped $want_hash but $HOST has $got_hash"
  before="$(ssh -o BatchMode=yes "$HOST" "pgrep -c -f '$SPAT' || true" | tr -d ' \n')"
  [[ "$before" == "0" ]] && ok "bracket pattern reads 0 with no sampler running" \
                         || bad "bracket pattern reads $before with no sampler running"
  naive="$(ssh -o BatchMode=yes "$HOST" 'pgrep -c -f freq150-sampler.sh || true' | tr -d ' \n')"
  echo "  for the record, the naive pattern reads $naive here with nothing running -- it matches the"
  echo "  shell ssh spawned to run it, which is why the driver's pattern carries a bracket."
  ssh -o BatchMode=yes "$HOST" "
    rm -f '$RDIR/freq150-trace-test.txt'
    nohup setsid taskset -c 2 '$RDIR/freq150-sampler.sh' 0 0.2 '$RDIR/freq150-trace-test.txt' \
      >/dev/null 2>&1 </dev/null & echo started" >/dev/null 2>&1
  sleep 2
  live="$(ssh -o BatchMode=yes "$HOST" "pgrep -c -f '$SPAT' || true" | tr -d ' \n')"
  [[ "$live" == "1" ]] && ok "bracket pattern reads 1 while the sampler runs" \
                       || bad "bracket pattern reads $live while the sampler runs"
  trace="$(ssh -o BatchMode=yes "$HOST" "cat '$RDIR/freq150-trace-test.txt'" 2>/dev/null)"
  lines="$(/usr/bin/grep -c . <<<"$trace")"
  # ~5 samples/s for 2s, minus startup. Fewer than 5 means the period is not being honoured or the
  # appends are being buffered -- either of which would make an arm's trace unusable.
  [[ "$lines" -ge 5 ]] && ok "trace grew to $lines lines in 2s (>=5 expected at 0.2s)" \
                       || bad "trace grew to only $lines lines in 2s"
  wellformed="$(/usr/bin/grep -cE '^[0-9]+\.[0-9]+ [0-9]+$' <<<"$trace")"
  [[ "$wellformed" == "$lines" && "$lines" -gt 0 ]] \
    && ok "all $lines samples are '<epoch.frac> <khz>'; first=$(head -1 <<<"$trace")" \
    || bad "$wellformed of $lines samples well-formed; first=$(head -1 <<<"$trace")"
  # The frequency must actually MOVE or at least be a plausible kHz on this ladder -- a stuck
  # instrument reads perfectly and says nothing (a readable constant certifies nothing).
  span="$(awk '{if($2+0>0){if(!n++||$2<lo)lo=$2; if($2>hi)hi=$2}} END{printf "%s %s", lo+0, hi+0}' <<<"$trace")"
  echo "  sampled kHz span over the test window: $span (host ladder is 1200000..4400000)"
  awk -v s="$span" 'BEGIN{split(s,a," "); exit !(a[1]>=1000000 && a[2]<=4400000)}' \
    && ok "sampled frequencies lie on the host's registered ladder" \
    || bad "sampled frequencies [$span] are off the registered ladder"
  ssh -o BatchMode=yes "$HOST" "pkill -f '$SPAT' || true" >/dev/null 2>&1
  sleep 1
  after="$(ssh -o BatchMode=yes "$HOST" "pgrep -c -f '$SPAT' || true" | tr -d ' \n')"
  [[ "$after" == "0" ]] && ok "the sampler is gone after the kill" \
                        || bad "$after sampler(s) survived the kill"
  ssh -o BatchMode=yes "$HOST" "rm -f '$RDIR/freq150-trace-test.txt'" >/dev/null 2>&1
fi

echo
echo "== 5. the arm table and the sequences, against the registration =="
REGDUMP="$(python3 - "$REG" <<'PY'
import importlib.util as u, sys
s = u.spec_from_file_location("p", sys.argv[1]); m = u.module_from_spec(s); s.loader.exec_module(m)
for name, a in m.ARMS.items():
    print("ARM %s %s %s %s" % (name, a["mask"], a["cores"], a["env"]))
print("HUNT_A %s" % " ".join(m.HUNT_A))
print("HUNT_B %s" % " ".join(m.HUNT_B))
print("CONTROL %s" % " ".join("%s:%s" % (k, "on" if v else "off") for k, v in m.CONTROL_SEQUENCE))
print("CONTROL_ARM %s" % m.CONTROL_ARM)
print("PERIOD %s" % m.SAMPLE_PERIOD_S)
print("SELF_N %s" % m.QUIET_SELF_N)
print("FOREIGN_MAX %s" % m.QUIET_FOREIGN_MAX)
print("SAMPLED_CPU %s" % m.SAMPLED_CPU)
PY
)"
while read -r _ name mask cores aenv; do
  [[ "$(mask_for "$name")"  == "$mask"  ]] && ok "$name mask $mask"    || bad "$name mask: driver $(mask_for "$name") vs registration $mask"
  [[ "$(cores_for "$name")" == "$cores" ]] && ok "$name cores $cores"  || bad "$name cores: driver $(cores_for "$name") vs registration $cores"
  [[ "$(env_for "$name")"   == "$aenv"  ]] && ok "$name env [$aenv]"   || bad "$name env: driver [$(env_for "$name")] vs registration [$aenv]"
done < <(awk '/^ARM /' <<<"$REGDUMP")
# The driver's DEFAULTS, read out of the driver rather than from this script's environment: a
# sequence that drifted from the registration would run arms nobody registered.
dflt() { sed -n "s/^$1=\"\${$1-\(.*\)}\"\$/\1/p;s/^$1=\"\${$1:-\(.*\)}\"\$/\1/p" "$DRIVER" | head -1; }
for k in HUNT_A HUNT_B CONTROL CONTROL_ARM; do
  wantv="$(sed -n "s/^$k //p" <<<"$REGDUMP")"
  gotv="$(dflt "$k")"
  [[ -n "$gotv" && "$gotv" == "$wantv" ]] && ok "$k = [$gotv]" || bad "$k: driver [$gotv] vs registration [$wantv]"
done
[[ "$(dflt SAMPLE_PERIOD)" == "$(sed -n 's/^PERIOD //p' <<<"$REGDUMP")" ]] \
  && ok "SAMPLE_PERIOD = $(dflt SAMPLE_PERIOD)" \
  || bad "SAMPLE_PERIOD: driver $(dflt SAMPLE_PERIOD) vs registration $(sed -n 's/^PERIOD //p' <<<"$REGDUMP")"
# The guard's two constants, which live in three places by necessity -- derived in
# derive-pedestal.py, registered here, applied in the driver -- so two of the three are compared.
[[ "$(dflt SELF_N)" == "$(sed -n 's/^SELF_N //p' <<<"$REGDUMP")" ]] \
  && ok "SELF_N = $(dflt SELF_N)" \
  || bad "SELF_N: driver $(dflt SELF_N) vs registration $(sed -n 's/^SELF_N //p' <<<"$REGDUMP")"
[[ "$(dflt QUIET_FOREIGN_MAX)" == "$(sed -n 's/^FOREIGN_MAX //p' <<<"$REGDUMP")" ]] \
  && ok "QUIET_FOREIGN_MAX = $(dflt QUIET_FOREIGN_MAX)" \
  || bad "QUIET_FOREIGN_MAX: driver $(dflt QUIET_FOREIGN_MAX) vs registration $(sed -n 's/^FOREIGN_MAX //p' <<<"$REGDUMP")"
[[ "$(dflt SAMPLED_CPU)" == "$(sed -n 's/^SAMPLED_CPU //p' <<<"$REGDUMP")" ]] \
  && ok "SAMPLED_CPU = $(dflt SAMPLED_CPU)" \
  || bad "SAMPLED_CPU: driver $(dflt SAMPLED_CPU) vs registration $(sed -n 's/^SAMPLED_CPU //p' <<<"$REGDUMP")"

echo
printf '=== test-driver-freq150: %d passed, %d failed ===\n' "$pass" "$fail"
[[ "$fail" -eq 0 ]] || exit 1
