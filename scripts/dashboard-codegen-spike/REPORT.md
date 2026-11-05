# [tf-export 13] Spike: generating Go for one panel type

Time-boxed spike for nickpeihl/terraform-provider-elasticstack#5. It generates Plugin Framework Go for one panel type
(`slo_burn_rate`) from Kibana's annotated OpenAPI document and checks that Go against the hand-written handler
(`internal/kibana/dashboard/panel/sloburnrate`). The provider is unchanged: everything is under
`scripts/dashboard-codegen-spike/`, and nothing in the provider imports it.

## Answers

1. **How modes are derived.** The hypothesis holds with one addition: values the server generates.
   - A field is **required** when it is in the OpenAPI `required` list. That list already means "required in Zod
     with no default". None of the 299 dashboard properties with a `default` is also in `required`.
   - A field is **computed** (Optional + Computed) when it has a `default`. In every probe against Kibana 9.7, the
     server returned exactly the fields that carry a `default`, and none of the fields whose default is only stated
     in their description.
   - Across the whole resource, treating `default` as computed closes **101 of the 114** "Kibana optional, provider
     optional_computed" differences in the schema gate's baseline.
   - **The addition:** values the server generates or derives have no literal default, so `default` cannot mark
     them. Examples are panel and section `id`, Discover's `selected_tab_id`, and XY axis defaults. They need their
     own mark, such as `readOnly` or an `x-tf` key.
2. **Generic strategy instead of per-handler preserve logic.** Yes, for this panel, in either of two forms. Both
   are generated and both pass end to end through Terraform against Kibana 9.7:
   - **computed**: a defaulted field gets a schema `Default`. The read is a plain mapping that never looks at prior
     state, and import reads back exactly in every case tested. Its cost is a schema-mode change to released
     attributes.
   - **preserve**: the released schema stays as it is. One shared rule replaces the handler's preserve code: a
     defaulted field reads as null when it was unset and the API returned its default.

   The control row shows that, for this panel against Kibana 9.7, the only fields needing any preservation are the
   three marked with `default`. In that setting, the handler's null-preservation of `title`, `description`,
   `hide_title` and `hide_border` never changes the result. Older Kibana versions and other panel types were not
   tested. The `default` marking does not cover server **normalization** (semantic JSON equality, JSON-leaf defaults,
   plan alignment), which is most of the preserve code in the Lens panels.

Side finding: generating from the document exposed a bug in the hand-written code. The drilldown import path assumes
`open_in_new_tab` defaults to `false`. The document says `true`, and Kibana 9.7 returns `true`. Import therefore
inverts it: an unset value imports as `true`, and an explicit `false` imports as null.

## What was generated

The generator is `go run ./scripts/dashboard-codegen-spike generate`. It reads `components.schemas`, follows the
panel type's `config` reference, and builds an attribute tree from these OpenAPI constructs:

- `type`
- `required`
- `default`
- `description`
- `maxLength`/`minLength`
- `maxItems`/`minItems`
- `enum`, and an `anyOf` of string enums
- `x-tf-union` with `discriminator` and `key_template`

Any other construct is reported as unsupported; the generator does not guess. It emits one Go file:

| Part | What the generated code does |
| --- | --- |
| Model | One struct per object, `types.*` fields, `tfsdk` tags; a union element is a struct with a pointer per variant. |
| Schema | `Attributes()`, the config block's attributes with modes, defaults and validators. `SchemaAttribute()` wraps them in the existing `panelkit.PanelConfigBlock`: the sibling exclusion and panel-type guard stay hand-written. |
| Write | `ToAPI(Model)` builds the API's JSON object and decodes it into the `kbapi` type, which oapi-codegen names from the component name. That name is the only coupling to `kbapi`. |
| Read | `FromAPI(api)` for the computed strategy. `FromAPI(api, prior)` for the preserve strategy, which reads the prior only for defaulted fields. |

`codegenrt/` is the small hand-written runtime that the generated code calls (165 lines, with put/get helpers and the
default-aware rule). The generated file has 242 lines. The hand-written handler has 287 lines, and it also relies on
panelkit helpers for presentation, drilldowns and SLO instance IDs.

Files:

- `sloburnratecomputed/slo_burn_rate_gen.go` and `sloburnratepreserve/slo_burn_rate_gen.go`: the generated code,
  one file per strategy.
- `testdata/slo_burn_rate.oas.json`: the part of the OpenAPI document the generator read, so the tests do not need
  a Kibana checkout. It comes from nickpeihl/kibana `5a948a54` (branch `poc/dashboards-terraform-export`), which
  carries the nickpeihl/kibana#34 annotations.
- `testdata/survey.txt`: the output of `survey` over the full document.

## How it was verified

| Check | Test | Evidence |
| --- | --- | --- |
| The generated code matches the generator | `TestGeneratedCodeIsCurrent` | Regenerates both files from the extract. |
| Schema versus hand-written | `TestSchemaVersusHandWritten` | `testdata/schema_diff.golden` |
| Write versus hand-written: the same configuration produces the same request | `TestWriteVersusHandWritten` | `testdata/write.golden` |
| Read back: state after apply and after import equals the plan | `TestReadBack` | `testdata/read.golden`, on responses recorded from Kibana 9.7.0 in `testdata/kibana/` |
| Which fields Kibana fills (question 1) | Manual probe | `testdata/probes/defaults_probe.json` (time slider, synthetics monitors, filters, options). The `slo_burn_rate` fields are covered by `testdata/kibana/`. |
| Whole-resource mode experiment (question 1) | `dashboard-schema-gate check` with the patch | `testdata/gate_default_as_computed.txt`: the lines only the patched run reports |
| End to end through Terraform | `TestAccGeneratedSloBurnRate`, opt-in with `CODEGEN_SPIKE_ACC=1` | A throwaway provider whose resource is one dashboard with the generated panel. It runs these steps for both strategies against Kibana 9.7: create minimal, import, update to every field plus drilldowns, import, explicit `"*"`, back to minimal. Terraform checks that each apply is consistent and that the plan after it is empty. The steps deliberately avoid importing an explicit default, which the preserve strategy cannot do (see below). |

The end-to-end harness catches mistakes. With the default rule removed from the preserve read, the first step fails
with `Provider produced inconsistent result after apply … .config.slo_instance_id: was null, but now
cty.StringVal("*")`.

Read-back results (`testdata/read.golden`, abridged):

| Case | Hand-written | Generated computed | Generated preserve | Control: plain read, no default handling |
| --- | --- | --- | --- | --- |
| minimal | ✓ / ✓ | ✓ / ✓ | ✓ / ✓ | `slo_instance_id` null → `"*"` |
| presentation (title, description, hide_*) | ✓ / ✓ | ✓ / ✓ | ✓ / ✓ | `slo_instance_id` only |
| `slo_instance_id = "host-a"` | ✓ / ✓ | ✓ / ✓ | ✓ / ✓ | ✓ |
| `slo_instance_id = "*"` | ✓ / import → null | ✓ / ✓ | ✓ / import → null | ✓ |
| drilldowns | ✓ / import wrong on 3 fields | ✓ / ✓ | ✓ / import → null on the explicit `true`s | `encode_url`, `open_in_new_tab`, `slo_instance_id` |

Each cell shows *after apply / after import*, and ✓ means equal to the plan.

- Under the preserve strategy, an explicit default imports as null. This is inherent: after import, nothing
  distinguishes "set to the default" from "unset". The hand-written handler behaves the same way for `"*"`.
- Only the computed strategy imports every case exactly.
- Kibana stores the same configuration whether a default is sent explicitly (computed) or left out (hand-written).
  The recorded responses for minimal, presentation and drilldowns are identical.

## How the generated code differs from the hand-written handler

From `testdata/schema_diff.golden` and `testdata/write.golden`:

| Area | Hand-written | Generated | Why, and what it would take |
| --- | --- | --- | --- |
| Drilldown shape | Flat `{url, label, encode_url, open_in_new_tab}`; trigger and type are implied | `{ url_drilldown = { url, label, trigger, … } }`, from `x-tf-union` | The generator follows Kibana's mapping, as the export flyout does. The schema gate already baselines this as 4 `extra` lines and 1 `missing` line. Generating it for the released resource would need a Kibana mapping change or a state upgrade. A single-variant union could also collapse by rule, but that rule would be new mapping vocabulary. |
| `slo_instance_id`, `encode_url`, `open_in_new_tab` | Optional, plus per-field preserve code | Optional+Computed with `Default` (computed strategy), or Optional with the shared rule (preserve strategy) | Question 2 above. |
| `duration` validation | Regex `^\d+[mhd]$` | `LengthAtMost(32)` | The document has no `pattern`, because Zod's refinement is not exported. Validation is lost unless Kibana publishes a `pattern` (or an `x-tf` validator) for it. |
| `slo_id`, `slo_instance_id` length | Not validated | `LengthAtMost(64)` and `LengthAtMost(512)` | Gained from the document. |
| `trigger` | Fixed in code | Required, `OneOf("on_open_panel_menu")` | Follows the mapping. |
| Descriptions | Provider wording, in Markdown | Kibana's wording, without Markdown (8 of 8 differ) | Documentation would change. Better descriptions belong in Kibana. |
| Write of defaults | Omitted when unset | Sent explicitly in the computed strategy (`"slo_instance_id":"*"`) | Kibana stores the same result either way. |
| Kept hand-written | — | Panel scaffolding: `PanelConfigBlock`, sibling exclusion, `type` guard, grid and ID, rejecting `config_json`, `ValidatePanelConfig` | Shared across panels. It is outside the panel's config schema, so it is not generated. |

## Question 1: required, optional and computed

**Hypothesis:** required means required in Zod with no default; computed means it has a default.

Evidence:

1. **`required` and `default` never overlap.** The `survey` command walks the 466 component schemas reachable from
   the dashboard schemas. It finds 299 properties with a `default`, and none of them in `required`. Kibana publishes
   Zod's input schema, in which a `.default()` field is optional, so the document's `required` already means
   "required in Zod with no default".
2. **A marked default is echoed back, and a prose-only default is not.** These probes PUT a dashboard to Kibana 9.7
   and read it back:

   | Field | `default` in the document | Returned when omitted |
   | --- | --- | --- |
   | `slo_burn_rate.slo_instance_id` | `"*"` | `"*"` |
   | drilldown `encode_url`, `open_in_new_tab` | `true`, `true` | `true`, `true` |
   | `time_slider_control.start_percentage_of_time_range`, `end_…`, `is_anchored` | `0`, `1`, `false` | `0`, `1`, `false` |
   | dashboard `options.*` | yes, each field | yes, each field |
   | `hide_title`, `hide_border` (all panels) | no; the description says "Defaults to false" | not returned |
   | filter `negate`, `disabled`, `is_multi_index` | no; stated in the description | not returned |
   | `synthetics_monitors.view` | no; stated in the description | not returned |

   The probed fields whose default is only in prose are not stored, so Optional is the right mode for them. That
   was probed for three groups, not all 91 such properties in `testdata/survey.txt`. At least one of the 91,
   Discover's `selected_tab_id`, is filled by the server (see item 3), so a prose-only default does not always mean
   "not stored". Fields like that need the explicit mark proposed below.
3. **The whole resource, through the schema gate.** With a three-line change to Kibana's derivation
   (`testdata/kibana_default_as_computed.patch`: a property with a `default` derives as `optional_computed`), the
   gate compares the derived schema with the provider's real schema:
   - 101 of the baseline's 114 `mode … Kibana optional, provider optional_computed` lines close. A further 4
     `Kibana optional, provider required` lines are only relabeled. Where the provider
     made an attribute computed by hand, the document marks a default.
   - The 13 that stay open are not literal defaults:
     - `panels[].id`, `sections[].id`: generated by the server.
     - `discover_session_config.by_reference.selected_tab_id`: chosen by the server; its default is only in
       prose.
     - 9 XY axis and pie `legend` attributes: objects the server fills from other fields.
     - `config_json`: provider plumbing.
   - 113 new lines appear: Kibana marks a default where the provider is plain Optional. These are the fields where
     the provider keeps the released Optional schema and handles the echoed default with preserve code, as it does
     for `slo_instance_id`. Of them, 73 are under `panels[]`, 30 under `pinned_panels[]`, 8 in `options`, and one
     each in `tags` and `sections[]`.

**Rule to adopt:** in `required` → Required. Has `default` → Optional + Computed, with the default as the
schema `Default` when it is a literal. Generated or derived by the server → Computed, marked explicitly in Kibana
(proposed: `readOnly`, or an `x-tf-computed` key). Everything else → Optional.

The gate README's line "Kibana cannot derive computed-ness from the OpenAPI document" is too strong. Kibana can
derive it for defaulted fields, which are most of them.

**Caveat:** the probes cover a sample of fields, not every one of the 299. The rule depends on a convention in
Kibana: a value the server stores must use `.default()`, not a default applied in code. The gate would enforce that
convention once its derivation uses `default`. A field that violates it shows up as an inconsistent result after
apply in acceptance tests.

## Question 2: a generic strategy instead of per-handler preserve logic

The hand-written handler keeps state consistent with three pieces of per-panel code:

- `NullPreservePresentationFromPrior`
- `PreserveSloInstanceID`, with its `"*"` sentinel rule
- `ReadURLDrilldownsFromAPI`, with hard-coded import defaults

The control row in the read-back table shows what really needs handling against Kibana 9.7. A plain read differs
from the plan only on the three fields with a `default`. The presentation null-preservation never changes the
result.

Both generic strategies replace all three pieces and pass the same Terraform end-to-end steps:

| | Computed strategy | Preserve strategy |
| --- | --- | --- |
| Per-panel code | None. The read takes no prior state. | None. One shared rule (`codegenrt.Defaulted*`), applied where the document marks a `default`. |
| Import | Exact in every case | An explicit default imports as null (inherent; same as hand-written) |
| Plan readability | The plan shows the value Kibana will store (`slo_instance_id = "*"`) | The plan shows null |
| Released schema | Changes: Optional → Optional+Computed on each defaulted attribute. After upgrading, existing state with null in a defaulted attribute will probably show a one-time in-place update to the default. That follows from Plugin Framework behaviour and was not verified here. | Unchanged |
| Drift on unset fields | Shown | Shown, except a value equal to the default. The hand-written `PreserveSloInstanceID` hides *any* API value for an unset field. |

Defaults on attributes nested in a list work in Plugin Framework 1.19: `drilldowns[].url_drilldown.encode_url`
plans as `true`, and Terraform verified it.

**Where the generic strategy stops.** Of the provider's 23k lines of panel code, 10.6k are the Lens charts and
`visconfig`. Their preserve code is mostly about normalization, not defaults:

- `PreservePriorJSONWithDefaultsIfEquivalent`: 15 call sites
- `PreservePriorNormalizedWithDefaultsIfEquivalent`: 4 call sites
- each handler's `PopulateJSONDefaults`
- `models_plan_state_alignment.go`

These compare JSON leaves semantically, fill defaults inside JSON strings, and align plan and state. Marking
defaults does not remove that code. The hand-coded `PopulateJSONDefaults` maps could probably be generated from the
same `default` keys inside JSON-leaf schemas, but this spike did not test that.

**Recommendation:** use the computed strategy for new resources and for attributes not yet released. For released
attributes, use the preserve strategy, or computed plus a documented one-time diff, and make that choice once, at
the resource level. Either way, replace the hard-coded drilldown import defaults with the document's defaults.
Retire `NullPreservePresentationFromPrior` only after the read-back test passes for every panel type that uses it,
against the oldest supported Kibana version. Fixing the `open_in_new_tab` import bug does not depend on this spike.

## Effort estimate for the rest

`survey` over the full document shows **6 of 24 panel types generate today**: `slo_alerts`, `slo_burn_rate`,
`slo_error_budget`, `synthetics_monitors`, `synthetics_stats_overview` and `time_slider_control`. For 11 panel
types, `survey` stops at the top-level config union (by value or by reference), so their counts below are lower
bounds.

| Work | Estimate | Notes |
| --- | --- | --- |
| Make the generator production code for the 6 panel types that generate | 3–5 days | Decide the `kbapi` boundary (the JSON object conversion used here, or typed fields), description policy, regenerate check in CI, golden tests, and wire into the handler registry behind the existing `iface.Handler`. |
| Constructs for the other 18 panel types | ~2 weeks | Lists of strings: 9 occurrences, small. `x-tf-ref`: 3. Top-level config unions and the other `x-tf-union` fields (`content`, `merge`, `variant_keys`, `default_value`, `discriminator_placement`, `exclusivity`). `x-tf-flatten`, `x-tf-unwrap`, `x-tf-omit`, `x-tf-json-leaf`. `nullable`. Kibana's `buildAttributeTree` already implements this vocabulary in TypeScript; port it, or have Kibana emit an intermediate tree for Go to consume. |
| Lens `vis` panels (12 chart types, 10.6k hand-written lines) | 2–4 weeks, high uncertainty | Most of the work is normalization (JSON-leaf equality, plan alignment), not schema or conversion. Generating the schema and conversion is in scope; the normalization stays hand-written or needs new markings in Kibana. |
| Compatibility with the released schema | 1–2 weeks | Not generation work, and it cannot be generated. The gate tracks 320 baseline lines and 6 kind mismatches. Each one needs a Kibana mapping change or a provider state upgrade before generated code can replace a handler. Choosing the computed strategy adds 113 mode changes. |
| Kibana marks for server-generated values | 1–2 days in Kibana | `id` fields, `selected_tab_id`, XY axis and pie legend defaults (question 1). |

**Total:** about 6–10 weeks to replace every panel handler. A useful first step is the 6 panel types that generate
today, plus the shared default rule: about 1 week. It retires the per-panel preserve code for those panels.

## Limits of this spike

- One panel type. Its config has no nested union at the top, no JSON leaves and no references.
- The default probes cover the fields listed in question 1, not all 299 defaulted or 91 prose-default properties.
- The read-back cases set all four presentation fields together, so only `minimal` exercises them unset.
- The generator emits no description where the document has none (5 of 14 attributes here). Production code would
  have to treat a missing description as unsupported, because the repository requires descriptions.
- The generator is spike code. It supports exactly the constructs listed above and fails loudly on anything else.
- The released-schema upgrade behaviour of the computed strategy was reasoned about, not tested.
- The recorded responses come from a Kibana 9.7.0 development build.

## Reproduce

```sh
# Unit tests: golden schema, write and read-back files, and the generated code
go test ./scripts/dashboard-codegen-spike/...

# Regenerate the code and goldens after a change
go test ./scripts/dashboard-codegen-spike/ -update

# Record the Kibana responses again
CODEGEN_SPIKE_RECORD=1 KIBANA_ENDPOINT=http://localhost:5601 go test ./scripts/dashboard-codegen-spike/ -run TestReadBack -update

# End to end through Terraform (needs TF_ACC_TERRAFORM_PATH or terraform on PATH)
CODEGEN_SPIKE_ACC=1 TF_ACC=1 KIBANA_ENDPOINT=http://localhost:5601 go test ./scripts/dashboard-codegen-spike/ -run TestAccGenerated -v

# Survey and extract from a Kibana checkout
go run ./scripts/dashboard-codegen-spike survey --oas ../kibana/oas_docs/output/kibana.yaml
go run ./scripts/dashboard-codegen-spike extract --oas ../kibana/oas_docs/output/kibana.yaml \
  --panel-type slo_burn_rate --out scripts/dashboard-codegen-spike/testdata/slo_burn_rate.oas.json

# The whole-resource mode experiment
git -C ../kibana apply "$PWD/scripts/dashboard-codegen-spike/testdata/kibana_default_as_computed.patch"
go run ./scripts/dashboard-schema-gate check --kibana-dir ../kibana   # compare with the run without the patch
```
