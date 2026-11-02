#!/usr/bin/env bash
# PROTOTYPE (spike F): value conversion of every spike0 fixture to out/tf/*.tf.json + round trip,
# then a static check of the *.tf.json against the golden (baseline) attribute tree.
#   ./run_values.sh [pinned|main]      (run ./run_schema.sh first; it writes the overlaid OAS)
set -uo pipefail
cd "$(dirname "$0")"
OAS=${1:-pinned}
DT=/Users/nickpeihl/Development/Elastic/kibana/.claude/dashboards-terraform
node src/convert_and_round_trip.ts "out/kibana-$OAS.overlaid.yaml" "$DT/spike0/fixtures" out/tf
node src/convert_and_round_trip.ts "out/kibana-$OAS.overlaid.yaml" drift/escape_fixture out/tf-escape
echo "--- *.tf.json vs baseline schema (golden tree)"
node src/check_tf_json.ts "$DT/spike0/golden-attribute-tree.json" out/tf
