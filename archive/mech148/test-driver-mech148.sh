#!/usr/bin/env bash
# Copyright 2026 Scott Friedman
# SPDX-License-Identifier: Apache-2.0
#
# Exercises driver-mech148.sh's two hand-written decision procedures against synthetic input,
# on the dev host, with no ssh and no benchmark. Two things are checked and they are checked
# for different reasons:
#
#   1. THE QUIETNESS GATE, driven RED on purpose. A precondition that has only ever been seen
#      to pass is an unread witness: it would print `host quiet` on a stuck reading, on an
#      unparsable sample, and on a host with no /proc/loadavg at all, and nothing in a green
#      run could tell the difference. So every refusal branch is asserted here, including the
#      two failure-to-read branches that no healthy run can reach.
#
#   2. THE ARM TABLE, cross-checked against predictions-mech148.py's ARMS. The registration is
#      the authority for what test 3 measures and this driver is a second copy of it; two
#      copies of a table agree by luck until something compares them. A mask or an env that
#      drifted from the registration would run arms nobody registered while the analyzer scored
#      the ones nobody ran.
#
# Run: bash archive/mech148/test-driver-mech148.sh
set -uo pipefail
cd "$(dirname "$0")/../.." || exit 3

# Overridable so the cross-check can be driven RED against a deliberately perturbed copy.
# A comparison that has only ever agreed is not known to be a comparison.
DRIVER="${DRIVER:-archive/mech148/driver-mech148.sh}"
pass=0; fail=0
ok()  { pass=$((pass+1)); printf '  PASS %s\n' "$1"; }
bad() { fail=$((fail+1)); printf '  FAIL %s\n' "$1"; }

# The two functions under test, taken from the driver by extraction rather than retyped: a
# retyped copy tests the copy. sed pulls each function body between its opening line and the
# closing brace at column 0.
extract() {
  awk -v f="$1" '
    $0 ~ "^" f "\\(\\)[ ]*\\{" { if ($0 ~ /\}[ ]*$/) { print; exit } inb = 1 }
    inb { print }
    inb && /^\}$/ { exit }
  ' "$DRIVER"
}
QUIET_L5_MAX=1.25
eval "$(extract quiet_violation)"
eval "$(extract env_for)"
eval "$(extract mask_for)"
eval "$(extract cores_for)"
for f in quiet_violation env_for mask_for cores_for; do
  declare -F "$f" >/dev/null || { echo "FATAL: could not extract $f() from $DRIVER"; exit 2; }
done
echo "extracted quiet_violation, env_for, mask_for, cores_for from $DRIVER"

sample() { printf 'uptime: up 41 days\nloadavg: %s 1/703 2556983\ngovernor: performance\n' "$1"; }

echo
echo "== 1. the quietness gate =="
# want=quiet expects return 1 (no violation); want=refuse expects return 0 (violation).
check() {
  local want="$1" desc="$2" why rc
  why="$(quiet_violation)"; rc=$?
  if [[ "$want" == quiet && "$rc" -eq 1 ]]; then ok "$desc -> quiet ($why)"
  elif [[ "$want" == refuse && "$rc" -eq 0 ]]; then ok "$desc -> REFUSED ($why)"
  else bad "$desc -> wanted $want, got rc=$rc: $why"; fi
}

# The tracked clean samples: every one of the 15 must pass, or the bound is not the bound the
# derivation claims. These are the real triples from core148-97a21f4.log.
for t in "0.00 0.00 0.00" "1.05 0.99 0.68" "0.92 0.98 0.92" "1.00 1.00 1.00" \
         "0.92 0.98 0.99" "1.01 1.01 1.00" "1.04 1.02 1.00"; do
  HS="$(sample "$t")"; check quiet "tracked clean sample [$t]"
done
# The tracked contaminated sample. This is the ONE positive example the record contains and
# the whole field choice turns on it.
HS="$(sample "0.99 2.17 1.83")"; check refuse "tracked excursion [0.99 2.17 1.83], line 257"
# ... and the same sample proves the field choice is load-bearing rather than decorative: its
# 1-minute value alone would have passed.
HS="$(sample "0.99 0.99 0.99")"; check quiet "the excursion's 1-minute value alone [0.99]"

# The bound's own edges. `exceeds` is strict, so the bound value itself is quiet.
HS="$(sample "1.00 1.25 1.00")"; check quiet  "exactly at the bound [1.25]"
HS="$(sample "1.00 1.26 1.00")"; check refuse "one hundredth over the bound [1.26]"
# Today's observed co-tenant on janus, which is what prompted the ruling.
HS="$(sample "1.15 1.40 0.90")"; check refuse "the co-tenant observed 2026-09-02 [l5 1.40]"

echo
echo "  the branches no healthy run can reach (a guard must fail closed):"
HS=""                                              ; check refuse "empty host sample"
HS="uptime: up 41 days"                            ; check refuse "sample with no loadavg line"
HS="$(printf 'loadavg: \n')"                       ; check refuse "loadavg line with no fields"
HS="$(printf 'loadavg: 1.00\n')"                   ; check refuse "loadavg truncated after 1 field"
HS="$(printf 'loadavg: ssh: connect refused\n')"   ; check refuse "an error message where the numbers go"

echo
echo "== 2. the arm table against predictions-mech148.py =="
# The registration is read by python and the driver by bash, so neither can quietly define the
# other's answer.
regs="$(python3 -c '
import importlib.util, pathlib
p = pathlib.Path("archive/mech148/predictions-mech148.py")
s = importlib.util.spec_from_file_location("pred", p); m = importlib.util.module_from_spec(s)
s.loader.exec_module(m)
for name, a in m.ARMS.items():
    print(name, a["mask"], a["cores"], a["env"], sep="|")
')"
[[ -n "$regs" ]] || { echo "  FATAL: could not read ARMS from predictions-mech148.py"; exit 2; }
n=0
while IFS='|' read -r name mask cores aenv; do
  n=$((n+1))
  [[ "$(mask_for  "$name")" == "$mask"  ]] && ok "$name mask=$mask"   || bad "$name mask: driver says '$(mask_for "$name")', registration says '$mask'"
  [[ "$(cores_for "$name")" == "$cores" ]] && ok "$name cores=$cores" || bad "$name cores: driver says '$(cores_for "$name")', registration says '$cores'"
  [[ "$(env_for   "$name")" == "$aenv"  ]] && ok "$name env=[$aenv]"  || bad "$name env: driver says '$(env_for "$name")', registration says '$aenv'"
  # GOMAXPROCS must be the FIRST assignment in every env string: it is the canary for the
  # unquoted word-split in scripts/remote.sh:1687, and a canary in second place is not one.
  [[ "$aenv" == GOMAXPROCS=1* ]] && ok "$name env leads with the GOMAXPROCS canary" || bad "$name env does not lead with GOMAXPROCS=1: [$aenv]"
done <<<"$regs"
[[ "$n" -eq 5 ]] && ok "all 5 registered arms compared" || bad "compared $n arms, expected 5"

# The driver's two passes must cover exactly the registered arms, each once, and pass b must
# be pass a reversed -- that reversal is what makes time drift measurable.
A="$(sed -n 's/^ARMS_A="${ARMS_A-\(.*\)}"$/\1/p' "$DRIVER")"
B="$(sed -n 's/^ARMS_B="${ARMS_B-\(.*\)}"$/\1/p' "$DRIVER")"
echo "  ARMS_A=[$A] ARMS_B=[$B]"
want="$(tr ' ' '\n' <<<"$regs" >/dev/null; cut -d'|' -f1 <<<"$regs" | sort | tr '\n' ' ')"
[[ "$(tr ' ' '\n' <<<"$A" | sort | tr '\n' ' ')" == "$want" ]] && ok "pass a covers exactly the registered arms" || bad "pass a is [$A], registered set is [$want]"
[[ "$(tr ' ' '\n' <<<"$B" | sort | tr '\n' ' ')" == "$want" ]] && ok "pass b covers exactly the registered arms" || bad "pass b is [$B], registered set is [$want]"
rev="$(tr ' ' '\n' <<<"$A" | tail -r 2>/dev/null || tr ' ' '\n' <<<"$A" | tac)"
[[ "$(tr '\n' ' ' <<<"$rev")" == "$B " ]] && ok "pass b is pass a reversed" || bad "pass b is not pass a reversed: [$B] vs [$(tr '\n' ' ' <<<"$rev")]"

echo
printf 'test-driver-mech148: %d pass / %d fail\n' "$pass" "$fail"
[[ "$fail" -eq 0 ]] || exit 1
