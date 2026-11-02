#!/usr/bin/env bash
# PROTOTYPE (spike G): fixture -> fake Kibana GET -> provider Read -> terraform -generate-config-out
# -> import -> drift plan -> replace (provider writes API JSON) -> compare with fixture.
# Usage: prototype_g/run_all.sh [outDir]   (needs: go, node, mise terraform 1.16.6)
set -uo pipefail
HERE="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(dirname "$HERE")"
FX="${FIXTURES:-/Users/nickpeihl/Development/Elastic/kibana/.claude/dashboards-terraform/spike0/fixtures}"
OUT="$(realpath -m "${1:-$HERE/out}")"
PORT=5699
rm -rf "$OUT"; mkdir -p "$OUT/bin" "$OUT/cap"
(cd "$ROOT" && go build -o "$OUT/bin/terraform-provider-elasticstack" .) || exit 1
cat > "$OUT/tfrc" <<RC
provider_installation {
  dev_overrides { "elastic/elasticstack" = "$OUT/bin" }
  direct {}
}
RC
export TF_CLI_CONFIG_FILE="$OUT/tfrc" NO_PROXY='*' no_proxy='*' TF_IN_AUTOMATION=1
TF() { mise exec terraform@1.16.6 -- terraform "$@" -no-color; }
node "$HERE/fake_kibana.mjs" "$FX" "$OUT/cap" $PORT > "$OUT/fake.log" 2>&1 &
FAKE=$!; trap 'kill $FAKE' EXIT; sleep 1
for f in "$FX"/*.json; do
  name="$(basename "$f" .json)"; [ "$name" = index ] && continue
  d="$OUT/$name"; mkdir -p "$d"
  sed "s/__ID__/$name/; s/__PORT__/$PORT/g" "$HERE/main.tf.tmpl" > "$d/main.tf"
  pushd "$d" >/dev/null
  TF plan -generate-config-out=generated.tf > 1_generate.txt 2>&1; echo "generate=$?" > status.txt
  TF apply -auto-approve > 2_import_apply.txt 2>&1; echo "import_apply=$?" >> status.txt
  TF plan -detailed-exitcode > 3_drift_plan.txt 2>&1; echo "drift_plan=$? (0=no-op,2=diff)" >> status.txt
  TF apply -auto-approve -replace=elasticstack_kibana_dashboard.this > 4_replace_apply.txt 2>&1; echo "replace_apply=$?" >> status.txt
  [ -f "$OUT/cap/PUT-$name.json" ] && cp "$OUT/cap/PUT-$name.json" written_api.json
  TF plan -detailed-exitcode > 5_post_write_plan.txt 2>&1; echo "post_write_plan=$? (0=no-op,2=diff)" >> status.txt
  popd >/dev/null
  node "$HERE/compare.mjs" "$f" "$d/written_api.json" > "$d/roundtrip.txt" 2>&1; echo "roundtrip=$?" >> "$d/status.txt"
  echo "== $name"; cat "$d/status.txt"
done
