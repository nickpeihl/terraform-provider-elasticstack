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
	"fmt"
	"sort"
	"strings"
)

// differences between the derived and the provider's attribute trees, each written as one line that the gate's lists
// hold verbatim. Kind mismatches break `terraform plan`; the others (missing, extra, mode) do not.
type differences struct {
	kind  []string
	other []string
}

// sectionPanels is where the provider repeats the panel attributes for panels inside sections.
const (
	sectionPanels = "sections[].panels[]"
	panels        = "panels[]"
)

func compareTrees(derived, provider attributeTree) differences {
	d := index(derived)
	p := index(provider)

	var diffs differences
	// below reports whether path sits under a path already reported, so one difference is reported once.
	reported := map[string]bool{}
	below := func(path string) bool {
		for parent := parentOf(path); parent != ""; parent = parentOf(parent) {
			if reported[parent] {
				return true
			}
		}
		return false
	}

	for _, path := range unionPaths(d, p) {
		if below(path) {
			reported[path] = true
			continue
		}
		da, inDerived := d[path]
		pa, inProvider := p[path]
		switch {
		case !inProvider:
			diffs.other = append(diffs.other, fmt.Sprintf("missing %s: Kibana %s, not in the provider", path, da.kindString()))
			reported[path] = true
		case !inDerived:
			diffs.other = append(diffs.other, fmt.Sprintf("extra %s: provider %s, not in Kibana's mapping", path, pa.kindString()))
			reported[path] = true
		case !compatible(da, pa):
			diffs.kind = append(diffs.kind, fmt.Sprintf("kind %s: Kibana %s, provider %s", path, da.kindString(), pa.kindString()))
			reported[path] = true
		case da.Mode != pa.Mode:
			diffs.other = append(diffs.other, fmt.Sprintf("mode %s: Kibana %s, provider %s", path, da.Mode, pa.Mode))
		}
	}
	diffs.kind = foldSections(diffs.kind)
	diffs.other = foldSections(diffs.other)
	return diffs
}

// compatible reports whether a value the derived kind describes is accepted where the provider has its kind. A list
// and a set are both arrays in Terraform JSON syntax, so they only differ in element kind.
func compatible(derived, provider attribute) bool {
	collection := func(k string) string {
		if k == kindSet {
			return kindList
		}
		return k
	}
	return collection(derived.Kind) == collection(provider.Kind) && derived.Element == provider.Element
}

// foldSections drops a difference inside sections when panels[] has the same one: panels inside sections repeat
// panels[], so the difference is counted once, under its unique path.
func foldSections(lines []string) []string {
	have := map[string]bool{}
	for _, l := range lines {
		have[l] = true
	}
	var out []string
	for _, l := range lines {
		if strings.Contains(l, " "+sectionPanels) && have[strings.Replace(l, " "+sectionPanels, " "+panels, 1)] {
			continue
		}
		out = append(out, l)
	}
	sort.Strings(out)
	return out
}

func index(t attributeTree) map[string]attribute {
	m := make(map[string]attribute, len(t.Attributes))
	for _, a := range t.Attributes {
		m[a.Path] = a
	}
	return m
}

// unionPaths lists every path of either tree, parents before their children.
func unionPaths(a, b map[string]attribute) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range []map[string]attribute{a, b} {
		for path := range m {
			if !seen[path] {
				seen[path] = true
				out = append(out, path)
			}
		}
	}
	sort.Strings(out)
	return out
}

// parentOf returns the attribute path that holds path ("a[].b" -> "a", "a.b" -> "a", "a" -> "").
func parentOf(path string) string {
	before, _, ok := strings.CutLast(path, ".")
	if !ok {
		return ""
	}
	return strings.TrimSuffix(strings.TrimSuffix(before, "[]"), "{}")
}
