# Dashboard schema gate

Compares the `elasticstack_kibana_dashboard` schema that Kibana's variant mapping derives from its published
OpenAPI document with the provider's real schema. Kibana's Export → Terraform flyout writes configuration in the
derived shape, so where the two disagree on an attribute's kind (for example an object where the provider has a
string) that configuration fails `terraform plan`.

- **Derived schema:** Kibana's own `buildAttributeTree` (package `@kbn/as-code-dashboard-terraform`) run over
  `oas_docs/output/kibana.yaml`, which carries the `x-tf-*` mapping. The gate runs Kibana's TypeScript with Node
  (24 or later, for type stripping); it does not reimplement the derivation. The result is checked in as
  `derived_attribute_tree.json`, and `source.json` records the Kibana repository and revision it came from, like the
  pinned Kibana revision of `generated/kbapi`.
- **Provider schema:** the resource's `Schema`, walked into the same attribute-tree shape. Blocks
  (`kibana_connection`) are provider plumbing and are left out.

## What fails

| Difference | Where it goes | Rule |
| --- | --- | --- |
| Kind mismatch (`kind …`) | `known_kind_mismatches.json`, each with the ticket that fixes it | A mismatch not on the list fails. The list may shrink and never grow. |
| Missing, extra, mode (`missing …`, `extra …`, `mode …`) | `baseline.json` | A difference not in the baseline fails. The baseline may shrink and never grow. |

Both lists must match the comparison exactly: a fixed difference still on a list fails too, so the lists only
shrink by being updated. In CI, `--base-ref` reads the lists on the base branch, and any line added since fails.
There is one exception. A kind mismatch hides the differences at and under its attribute. When it is fixed, those
differences show and may enter the baseline. They are not new; the mismatch was hiding them.

How the comparison counts:

- Panels inside sections repeat `panels[]`. A difference under `sections[].panels[]` that `panels[]` also has is
  counted once, under `panels[]`.
- Under a missing, extra or kind-mismatched attribute nothing else is reported.
- A list and a set of the same element kind are the same kind: both are arrays in Terraform JSON syntax.
- Kibana cannot derive computed-ness from the OpenAPI document (it marks only required and optional), so most `mode`
  lines are `optional` versus `optional_computed`. They are tracked, not failed on.

## Commands

```sh
# What CI runs: compare, and fail if a list grew since the base branch.
make check-dashboard-schema DASHBOARD_SCHEMA_BASE_REF=origin/main

# Compare against a local Kibana checkout instead of the checked-in tree (for example a mapping change in review).
go run ./scripts/dashboard-schema-gate check --kibana-dir ../kibana

# Update the check: re-derive from a Kibana checkout, rewrite both lists, and set the provider version Kibana's
# flyout shows.
make update-dashboard-schema KIBANA_DIR=../kibana
```

`update` is the one step that bumps "the provider version this Kibana was verified against". It records the
provider's `VERSION` (Makefile) and revision in `source.json`, and writes the same version into `TERRAFORM_PROVIDER`
in Kibana's `src/platform/packages/shared/as-code/dashboard-terraform/src/provider.ts`. The Kibana change is then
committed in Kibana. `update` keeps the tickets of mismatches that remain and marks new ones `untracked: file a
ticket`. It reports every line it added or removed, and CI still rejects added lines.

Tests: `go test ./scripts/dashboard-schema-gate/`. The tests that derive from a deliberately wrong mapping in the
OpenAPI document need Node and `DASHBOARD_SCHEMA_GATE_KIBANA_DIR` set to a Kibana checkout at the recorded
revision; without them they skip. CI's gate job downloads the four Kibana files they read at that revision and runs
them.
