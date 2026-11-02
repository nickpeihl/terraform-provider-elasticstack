#!/usr/bin/env bash
# PROTOTYPE (spike F): overlay brittleness probe. Runs strict apply + emit + diff + round trip on the real
# Kibana main OAS and on synthetic re-bundling mutations of the pinned OAS.
cd "$(dirname "$0")/.."
DT=/Users/nickpeihl/Development/Elastic/kibana/.claude/dashboards-terraform
mkdir -p out/drift; node drift/make_mutations.mjs oas/kibana-pinned-23436cdb.json out/drift/specs
for spec in oas/kibana-main.yaml out/drift/specs/m*.json; do
  name=$(basename "${spec%.*}")
  echo "=================== $name"
  ./bin/overlay-apply-strict -spec "$spec" -out "out/drift/$name.overlaid.yaml" \
    overlay/dashboards.variants.overlay.yaml overlay/dashboards.modes.overlay.yaml 2>&1 | grep -v '^applied' | sed 's/^/  /'
  node src/emit_attribute_tree.ts "out/drift/$name.overlaid.yaml" "out/drift/$name.tree.json" 2>&1 | sed 's/^/  /'
  node "$DT/spike0/diff-tree.mjs" "out/drift/$name.tree.json" | grep -v '^candidate\|^golden\|^$' | sed 's/^/  /'
  node src/convert_and_round_trip.ts "out/drift/$name.overlaid.yaml" "$DT/spike0/fixtures" "out/drift/$name.tf" | tail -1 | sed 's/^/  /'
done
