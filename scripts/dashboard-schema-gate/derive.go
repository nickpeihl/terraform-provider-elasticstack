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
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Paths inside a Kibana checkout.
const (
	kibanaOASPath    = "oas_docs/output/kibana.yaml"
	kibanaPackageSrc = "src/platform/packages/shared/as-code/dashboard-terraform/src"
	kibanaProviderTS = kibanaPackageSrc + "/provider.ts"
)

// kibanaDeriveFiles are the Kibana sources `buildAttributeTree` needs; they import only types from elsewhere.
var kibanaDeriveFiles = []string{"attribute_tree.ts", "oas_model.ts"}

//go:embed derive/*.mjs
var deriveScripts embed.FS

// kibanaSource reads files of a Kibana checkout.
type kibanaSource struct {
	dir string
}

func (s kibanaSource) read(rel string) ([]byte, error) {
	return os.ReadFile(filepath.Join(s.dir, rel))
}

// deriveTree runs Kibana's own `buildAttributeTree` over Kibana's published OpenAPI document, so the gate compares
// exactly what Kibana's exporter produces. It needs `node` (24 or later, for TypeScript type stripping).
func deriveTree(src kibanaSource) ([]byte, error) {
	work, err := os.MkdirTemp("", "dashboard-schema-gate-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(work)

	for _, name := range []string{"resolve_ts.mjs", "attribute_tree.mjs"} {
		b, err := deriveScripts.ReadFile("derive/" + name)
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(work, name), b, 0o600); err != nil {
			return nil, err
		}
	}
	srcDir := filepath.Join(work, "src")
	if err := os.Mkdir(srcDir, 0o700); err != nil {
		return nil, err
	}
	for _, name := range kibanaDeriveFiles {
		b, err := src.read(kibanaPackageSrc + "/" + name)
		if err != nil {
			return nil, fmt.Errorf("read Kibana %s: %w", name, err)
		}
		if err := os.WriteFile(filepath.Join(srcDir, name), b, 0o600); err != nil {
			return nil, err
		}
	}

	oasYAML, err := src.read(kibanaOASPath)
	if err != nil {
		return nil, fmt.Errorf("read Kibana OpenAPI: %w", err)
	}
	schemas, err := componentSchemasJSON(oasYAML)
	if err != nil {
		return nil, err
	}
	oasPath := filepath.Join(work, "schemas.json")
	if err := os.WriteFile(oasPath, schemas, 0o600); err != nil {
		return nil, err
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.Command("node", "--import", "./resolve_ts.mjs", "attribute_tree.mjs", srcDir, oasPath)
	cmd.Dir = work
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("node attribute_tree.mjs: %w\n%s", err, stderr.String())
	}
	return stdout.Bytes(), nil
}

// componentSchemasJSON keeps only `components.schemas` of the OpenAPI YAML, which is all the derivation reads.
func componentSchemasJSON(oasYAML []byte) ([]byte, error) {
	var doc struct {
		Components struct {
			Schemas map[string]any `yaml:"schemas"`
		} `yaml:"components"`
	}
	if err := yaml.Unmarshal(oasYAML, &doc); err != nil {
		return nil, fmt.Errorf("parse Kibana OpenAPI: %w", err)
	}
	if len(doc.Components.Schemas) == 0 {
		return nil, fmt.Errorf("kibana OpenAPI has no components.schemas")
	}
	return json.Marshal(map[string]any{"components": map[string]any{"schemas": doc.Components.Schemas}})
}
