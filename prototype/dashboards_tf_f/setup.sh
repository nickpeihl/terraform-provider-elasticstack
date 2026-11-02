#!/usr/bin/env bash
# PROTOTYPE (spike F): one-time setup. Node deps, the strict overlay applier, and the two OAS inputs
# (provider-pinned Kibana commit from generated/kbapi/Makefile, and the local Kibana main checkout).
set -euo pipefail
cd "$(dirname "$0")"
npm ci --no-audit --no-fund
(cd cmd/overlay_apply_strict && go build -o ../../bin/overlay-apply-strict .)
mkdir -p oas
REF=$(sed -n 's/^github_ref ?= //p' ../../generated/kbapi/Makefile)
[ -f "oas/kibana-pinned-${REF:0:8}.yaml" ] || curl -sSfo "oas/kibana-pinned-${REF:0:8}.yaml" "https://raw.githubusercontent.com/elastic/kibana/$REF/oas_docs/output/kibana.yaml"
cp /Users/nickpeihl/Development/Elastic/kibana/oas_docs/output/kibana.yaml oas/kibana-main.yaml
node -e "const y=require('js-yaml'),fs=require('fs');fs.writeFileSync('oas/kibana-pinned-${REF:0:8}.json',JSON.stringify(y.load(fs.readFileSync('oas/kibana-pinned-${REF:0:8}.yaml','utf8'))))"
