#!/usr/bin/env bash
# PROTOTYPE (spike F): drift lint. (1) strict overlay apply = unmatched-target check, (2) un-mapped-union
# lint over the slice and over the full resource, (3) merge-asymmetry notes, (4) variant-lock diff
# pinned vs main.
cd "$(dirname "$0")"
for o in pinned main; do ./run_schema.sh $o >/dev/null 2>"out/lint.$o.txt"; echo "[$o] exit=$?"; grep -v '^applied' "out/lint.$o.txt"; done
echo "--- full resource (no slice filter), pinned"
node src/emit_attribute_tree.ts out/kibana-pinned.overlaid.yaml out/full-tree.pinned.json --full 2>&1 | grep -v '^NOTE' | grep -v '^UNMAPPED sections' | sed -E 's/: union \((.{0,120}).*/ :: \1/'
echo "--- variant lock: pinned -> main"
diff <(node -e "console.log(JSON.stringify(require('./out/candidate-attribute-tree.pinned.variant-lock.json'),null,1))") \
     <(node -e "console.log(JSON.stringify(require('./out/candidate-attribute-tree.main.variant-lock.json'),null,1))")
