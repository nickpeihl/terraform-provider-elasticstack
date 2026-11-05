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
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const refPrefix = "#/components/schemas/"

// schemas is `components.schemas` of Kibana's OpenAPI document.
type schemas map[string]any

// loadSchemas reads `components.schemas` from Kibana's OpenAPI YAML, or from the JSON this tool's
// `extract` command writes (the same shape, trimmed to one panel type).
func loadSchemas(path string) (schemas, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Components struct {
			Schemas map[string]any `yaml:"schemas" json:"schemas"`
		} `yaml:"components" json:"components"`
	}
	if strings.HasSuffix(path, ".json") {
		err = json.Unmarshal(b, &doc)
	} else {
		err = yaml.Unmarshal(b, &doc)
	}
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if len(doc.Components.Schemas) == 0 {
		return nil, fmt.Errorf("%s has no components.schemas", path)
	}
	return doc.Components.Schemas, nil
}

// resolve follows $ref until it reaches a schema without one.
func (s schemas) resolve(node map[string]any) (map[string]any, error) {
	for range 32 {
		ref, ok := node["$ref"].(string)
		if !ok {
			return node, nil
		}
		name := strings.TrimPrefix(ref, refPrefix)
		next, ok := s[name].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("unresolved $ref %s", ref)
		}
		node = next
	}
	return nil, fmt.Errorf("$ref chain too long")
}

// panelTypeSchemaName is the component that describes one panel type in the dashboard's panels union.
func panelTypeSchemaName(panelType string) string {
	return "Kibana_HTTP_APIs_kbn-dashboard-panel-type-" + panelType
}

// configSchema returns the panel type's `config` schema and its component name.
func (s schemas) configSchema(panelType string) (map[string]any, string, error) {
	panel, ok := s[panelTypeSchemaName(panelType)].(map[string]any)
	if !ok {
		return nil, "", fmt.Errorf("no component %s", panelTypeSchemaName(panelType))
	}
	props, _ := panel["properties"].(map[string]any)
	cfg, ok := props["config"].(map[string]any)
	if !ok {
		return nil, "", fmt.Errorf("%s has no config property", panelTypeSchemaName(panelType))
	}
	ref, _ := cfg["$ref"].(string)
	resolved, err := s.resolve(cfg)
	if err != nil {
		return nil, "", err
	}
	return resolved, strings.TrimPrefix(ref, refPrefix), nil
}

var refPattern = regexp.MustCompile(`#/components/schemas/([^"]+)`)

// closure returns the named components and every component they reference, transitively.
func (s schemas) closure(roots ...string) schemas {
	out := schemas{}
	stack := append([]string(nil), roots...)
	for len(stack) > 0 {
		name := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if _, done := out[name]; done {
			continue
		}
		node, ok := s[name]
		if !ok {
			continue
		}
		out[name] = node
		b, _ := json.Marshal(node)
		for _, m := range refPattern.FindAllStringSubmatch(string(b), -1) {
			stack = append(stack, m[1])
		}
	}
	return out
}

// names returns the component names, sorted.
func (s schemas) names() []string {
	out := make([]string, 0, len(s))
	for k := range s {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
