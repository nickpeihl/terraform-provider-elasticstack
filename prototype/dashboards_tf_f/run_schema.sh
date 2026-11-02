#!/usr/bin/env bash
# PROTOTYPE (spike F): overlay -> attribute tree -> diff against spike0 golden.
#   ./run_schema.sh [pinned|main]
set -euo pipefail
cd "$(dirname "$0")"
OAS=${1:-pinned}
case $OAS in pinned) SPEC=oas/kibana-pinned-23436cdb.yaml;; main) SPEC=oas/kibana-main.yaml;; esac
DT=/Users/nickpeihl/Development/Elastic/kibana/.claude/dashboards-terraform
mkdir -p out
./bin/overlay-apply-strict -spec "$SPEC" -out "out/kibana-$OAS.overlaid.yaml" overlay/dashboards.variants.overlay.yaml overlay/dashboards.modes.overlay.yaml
node src/emit_attribute_tree.ts "out/kibana-$OAS.overlaid.yaml" "out/candidate-attribute-tree.$OAS.json"
node "$DT/spike0/diff-tree.mjs" "out/candidate-attribute-tree.$OAS.json"
