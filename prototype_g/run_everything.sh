#!/usr/bin/env bash
# PROTOTYPE (spike G): reproduce every run in the G write-up. ~3 min.
set -u
HERE="$(cd "$(dirname "$0")" && pwd)"; cd "$HERE"
FX=/Users/nickpeihl/Development/Elastic/kibana/.claude/dashboards-terraform/spike0/fixtures
summ() { echo "##### $1"; for d in "$2"/*/; do [ -f "$d/status.txt" ] && echo "$(basename "$d"): $(paste -sd' ' "$d/status.txt")"; done; }
PROTO_G_GENCONFIG=off ./run_all.sh out_stock >/dev/null 2>&1; summ "stock provider (core legacy generator)" out_stock
PROTO_G_STRIP=off     ./run_all.sh out_fwgen >/dev/null 2>&1; summ "framework GenerateResourceConfig, no fix" out_fwgen
./run_all.sh out >/dev/null 2>&1; summ "framework GenerateResourceConfig + config_json strip" out
rm -rf fixtures_xy_patched; node patch_xy_fixtures.mjs "$FX" fixtures_xy_patched
FIXTURES=$HERE/fixtures_xy_patched ./run_all.sh out_xy_patched >/dev/null 2>&1; summ "xy fixtures + legend/fitting (simulated GET defaults)" out_xy_patched
FIXTURES=$HERE/fixtures_probe ./run_all.sh out_probe >/dev/null 2>&1; summ "probes" out_probe
for d in out/*/ out_xy_patched/*/ out_probe/*/; do echo "--- roundtrip $(basename "$d")"; cat "$d/roundtrip.txt"; done
