// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// stateDir copies the checked-in gate state into a temporary directory the test may change.
func stateDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{sourceFile, derivedTreeFile, knownMismatchesFile, baselineFile} {
		b, err := os.ReadFile(name)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), b, 0o600))
	}
	return dir
}

// editTree applies fn to the derived tree in dir, as a different mapping in Kibana would.
func editTree(t *testing.T, dir string, fn func([]attribute) []attribute) {
	t.Helper()
	p := filepath.Join(dir, derivedTreeFile)
	tree, err := readTree(p)
	require.NoError(t, err)
	tree.Attributes = fn(tree.Attributes)
	b, err := json.Marshal(tree)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(p, b, 0o600))
}

func editJSON(t *testing.T, path string, fn func(any) any) {
	t.Helper()
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	var v any
	require.NoError(t, json.Unmarshal(b, &v))
	b, err = json.Marshal(fn(v))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, b, 0o600))
}

func runGate(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	err := run(args, &stdout, &stderr)
	return stdout.String() + stderr.String(), err
}

// unmapMarkdown is the mapping mistake that first broke a real `terraform plan`: markdown's by-value/by-reference
// union with no `x-tf-union`, so the converter writes the whole config as one JSON string.
func unmapMarkdown(attrs []attribute) []attribute {
	var out []attribute
	for _, a := range attrs {
		for _, base := range []string{"panels[].markdown_config", "sections[].panels[].markdown_config"} {
			if a.Path == base {
				a = attribute{Path: base, Kind: "json-string", Mode: a.Mode}
			}
		}
		if strings.HasPrefix(a.Path, "panels[].markdown_config.") || strings.HasPrefix(a.Path, "sections[].panels[].markdown_config.") {
			continue
		}
		out = append(out, a)
	}
	return out
}

func TestCheckPassesOnTheCheckedInState(t *testing.T) {
	out, err := runGate(t, "check", "--dir", stateDir(t))
	require.NoError(t, err, out)
}

func TestCheckFailsOnANewKindMismatch(t *testing.T) {
	dir := stateDir(t)
	editTree(t, dir, unmapMarkdown)

	out, err := runGate(t, "check", "--dir", dir)

	require.Error(t, err)
	require.Contains(t, out, "new kind mismatch (breaks terraform plan): kind panels[].markdown_config: Kibana json-string, provider object")
	// Panels in sections repeat panels[]; the same mismatch is reported once, under its unique path.
	require.NotContains(t, out, "sections[].panels[].markdown_config")
	// Attributes under a mismatched one are not reported again as missing or extra.
	require.NotContains(t, out, "markdown_config.by_value")
}

// dropDescription is a provider attribute Kibana's mapping no longer produces: a new "extra" difference.
func dropDescription(attrs []attribute) []attribute {
	var out []attribute
	for _, a := range attrs {
		if a.Path != "description" {
			out = append(out, a)
		}
	}
	return out
}

const extraDescription = "extra description: provider string, not in Kibana's mapping"

func TestCheckFailsOnADifferenceNotInTheBaseline(t *testing.T) {
	dir := stateDir(t)
	editTree(t, dir, dropDescription)

	out, err := runGate(t, "check", "--dir", dir)

	require.Error(t, err)
	require.Contains(t, out, "new difference: "+extraDescription)
}

func TestCheckFailsWhenTheBaselineGrows(t *testing.T) {
	base := stateDir(t)
	dir := stateDir(t)
	editTree(t, dir, dropDescription)
	editJSON(t, filepath.Join(dir, baselineFile), func(v any) any { return append(v.([]any), extraDescription) })

	out, err := runGate(t, "check", "--dir", dir, "--base-dir", base)

	require.Error(t, err)
	require.Contains(t, out, "baseline.json grew: "+extraDescription)
	require.NotContains(t, out, "new difference")
}

const pieGroupBy = "kind panels[].vis_config.by_value.pie_chart_config.group_by: Kibana list<json-string>, provider list<object>"

// fixPieGroupBy maps the pie chart's group_by items as the provider has them, objects holding `config_json`
// (tf-export 20). It leaves `config_json` optional where the provider requires it, so the fix also shows a difference
// the mismatch hid.
func fixPieGroupBy(attrs []attribute) []attribute {
	var out []attribute
	for _, a := range attrs {
		for _, p := range []string{panels, sectionPanels} {
			groupBy := p + ".vis_config.by_value.pie_chart_config.group_by"
			if a.Path == groupBy {
				a.Element = kindObject
				out = append(out, attribute{Path: groupBy + "[].config_json", Kind: kindJSONString, Mode: "optional"})
			}
		}
		out = append(out, a)
	}
	return out
}

func TestCheckPassesOnTheKnownKindMismatchesRecountedByUniquePath(t *testing.T) {
	dir := stateDir(t)
	out, err := runGate(t, "check", "--dir", dir)

	require.NoError(t, err, out)
	// The prototype's diff listed 15 lines, some repeated under sections: 14 by unique attribute path, 7 since
	// tf-export 11 unwrapped the XY axis grid and ticks and the field statistics ES|QL query, and 6 since
	// tf-export 12 mapped reference-line thresholds to objects.
	require.Contains(t, out, "6 known kind mismatch(es)")
	var known knownMismatches
	require.NoError(t, readJSON(filepath.Join(dir, knownMismatchesFile), &known))
	for line := range known {
		require.NotContains(t, line, sectionPanels, "panels in sections are counted under panels[]")
	}
}

func TestCheckFailsUntilAFixedKindMismatchIsRemoved(t *testing.T) {
	base := stateDir(t)
	dir := stateDir(t)
	editTree(t, dir, fixPieGroupBy)

	out, err := runGate(t, "check", "--dir", dir, "--base-dir", base)
	require.Error(t, err)
	require.Contains(t, out, "fixed kind mismatch: "+pieGroupBy)

	editJSON(t, filepath.Join(dir, knownMismatchesFile), func(v any) any {
		delete(v.(map[string]any), pieGroupBy)
		return v
	})
	// The fix shows a difference the mismatch hid. It was always there, so taking it into the baseline is not growth.
	unmasked := "mode panels[].vis_config.by_value.pie_chart_config.group_by[].config_json: Kibana optional, provider required"
	out, err = runGate(t, "check", "--dir", dir, "--base-dir", base)
	require.Error(t, err)
	require.Contains(t, out, "new difference: "+unmasked)

	editJSON(t, filepath.Join(dir, baselineFile), func(v any) any { return append(v.([]any), unmasked) })
	out, err = runGate(t, "check", "--dir", dir, "--base-dir", base)
	require.NoError(t, err, out)
	require.Contains(t, out, "5 known kind mismatch(es)")
}

func TestCheckFailsWhenTheKnownKindMismatchesGrow(t *testing.T) {
	base := stateDir(t)
	dir := stateDir(t)
	editTree(t, dir, unmapMarkdown)
	markdown := "kind panels[].markdown_config: Kibana json-string, provider object"
	editJSON(t, filepath.Join(dir, knownMismatchesFile), func(v any) any {
		v.(map[string]any)[markdown] = "a ticket"
		return v
	})

	out, err := runGate(t, "check", "--dir", dir, "--base-dir", base)

	require.Error(t, err)
	require.Contains(t, out, "known_kind_mismatches.json grew: "+markdown)
	require.NotContains(t, out, "new kind mismatch")
}

// kibanaCheckout is a Kibana checkout holding only the file `update` writes.
func kibanaCheckout(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	b, err := os.ReadFile("testdata/kibana_provider.ts")
	require.NoError(t, err)
	p := filepath.Join(dir, kibanaProviderTS)
	require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o700))
	require.NoError(t, os.WriteFile(p, b, 0o600))
	return dir
}

func TestUpdateSetsTheProviderVersionTheFlyoutShows(t *testing.T) {
	dir := stateDir(t)
	kibana := kibanaCheckout(t)

	out, err := runGate(t, "update", "--dir", dir, "--kibana-dir", kibana, "--kibana-ref", "0123abcd",
		"--tree", derivedTreeFile, "--provider-version", "0.17.0", "--provider-commit", "fedcba987654")
	require.NoError(t, err, out)

	providerTS, err := os.ReadFile(filepath.Join(kibana, kibanaProviderTS))
	require.NoError(t, err)
	require.Contains(t, string(providerTS),
		"export const TERRAFORM_PROVIDER = { source: 'elastic/elasticstack', version: '0.17.0' } as const;")
	require.Contains(t, string(providerTS),
		"// 0.17.0 was the latest release when the mapping was compared with provider main at fedcba987654;")
	require.NotContains(t, string(providerTS), "0.16.6")

	out, err = runGate(t, "check", "--dir", dir)
	require.NoError(t, err, out)
	require.Contains(t, out, "verified against provider 0.17.0")
}

func TestUpdateShrinksTheListsAndKeepsTheirTickets(t *testing.T) {
	dir := stateDir(t)
	fixed := filepath.Join(t.TempDir(), "tree.json")
	editTree(t, dir, fixPieGroupBy)
	b, err := os.ReadFile(filepath.Join(dir, derivedTreeFile))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(fixed, b, 0o600))

	out, err := runGate(t, "update", "--dir", dir, "--kibana-dir", kibanaCheckout(t), "--kibana-ref", "0123abcd",
		"--tree", fixed, "--provider-version", "0.16.6", "--provider-commit", "fedcba987654")
	require.NoError(t, err, out)
	require.Contains(t, out, "removed kind mismatch: "+pieGroupBy)

	var known knownMismatches
	require.NoError(t, readJSON(filepath.Join(dir, knownMismatchesFile), &known))
	require.NotContains(t, known, pieGroupBy)
	require.Equal(t, "nickpeihl/terraform-provider-elasticstack#6 (tf-export 20)",
		known["kind panels[].vis_config.by_value.pie_chart_config.metrics: Kibana list<json-string>, provider list<object>"])
}

func TestUpdateRequiresTheKibanaCheckoutWhoseConstantItUpdates(t *testing.T) {
	_, err := runGate(t, "update", "--dir", stateDir(t), "--tree", derivedTreeFile)
	require.ErrorContains(t, err, "--kibana-dir is required")
}

// kibanaWithMapping copies what the derivation reads from the Kibana checkout in DASHBOARD_SCHEMA_GATE_KIBANA_DIR
// and lets the test change the published OpenAPI document. It needs node, so it skips without that checkout.
func kibanaWithMapping(t *testing.T, edit func(oas string) string) string {
	t.Helper()
	src := os.Getenv("DASHBOARD_SCHEMA_GATE_KIBANA_DIR")
	if src == "" {
		t.Skip("DASHBOARD_SCHEMA_GATE_KIBANA_DIR is not set to a Kibana checkout")
	}
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is not installed")
	}
	dir := t.TempDir()
	files := []string{kibanaOASPath, kibanaProviderTS}
	for _, name := range kibanaDeriveFiles {
		files = append(files, kibanaPackageSrc+"/"+name)
	}
	for _, rel := range files {
		b, err := os.ReadFile(filepath.Join(src, rel))
		require.NoError(t, err)
		if rel == kibanaOASPath {
			b = []byte(edit(string(b)))
		}
		require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(dir, rel)), 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(dir, rel), b, 0o600))
	}
	return dir
}

func TestCheckFailsOnADeliberatelyWrongMappingInKibana(t *testing.T) {
	mapped := "description: Markdown panel config\n          x-tf-union:\n            structural: true\n"
	kibana := kibanaWithMapping(t, func(oas string) string {
		require.Equal(t, 1, strings.Count(oas, mapped), "markdown's union mapping moved in the OpenAPI document")
		return strings.Replace(oas, mapped, "description: Markdown panel config\n", 1)
	})

	out, err := runGate(t, "check", "--dir", stateDir(t), "--kibana-dir", kibana)

	require.Error(t, err)
	require.Contains(t, out, "new kind mismatch (breaks terraform plan): kind panels[].markdown_config: Kibana json-string, provider object")
}

func TestCheckPassesOnKibanasMappingAtTheRecordedRef(t *testing.T) {
	kibana := kibanaWithMapping(t, func(oas string) string { return oas })

	out, err := runGate(t, "check", "--dir", stateDir(t), "--kibana-dir", kibana)

	require.NoError(t, err, out)
}

func TestCheckFailsOnABaseRefThatDoesNotResolve(t *testing.T) {
	// A mistyped ref, or a clone without the base branch, must not quietly drop the never-grow rule.
	_, err := runGate(t, "check", "--dir", stateDir(t), "--base-ref", "no-such-ref-for-the-gate")
	require.ErrorContains(t, err, `base ref "no-such-ref-for-the-gate"`)
}
