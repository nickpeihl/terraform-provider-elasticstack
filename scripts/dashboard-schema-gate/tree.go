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
)

// Attribute kinds of an attribute-tree/v1 document (Kibana's `AttributeKind`, plus set and map, which only the
// provider has).
const (
	kindString     = "string"
	kindJSONString = "json-string"
	kindBool       = "bool"
	kindNumber     = "number"
	kindObject     = "object"
	kindList       = "list"
	kindSet        = "set"
	kindMap        = "map"
)

// treeFormat is the format Kibana's `buildAttributeTree` (package @kbn/as-code-dashboard-terraform) writes.
const treeFormat = "attribute-tree/v1"

// attribute is one Terraform attribute of the dashboard resource. Paths use `a[].b` for the elements of a list or
// set and `a{}.b` for the values of a map.
type attribute struct {
	Path    string `json:"path"`
	Kind    string `json:"kind"`
	Element string `json:"element,omitempty"`
	Mode    string `json:"mode"`
}

// attributeTree is the attribute list of an attribute-tree/v1 document. Kibana's variant-key groups are not
// compared: they do not change what `terraform plan` accepts.
type attributeTree struct {
	Format     string      `json:"format"`
	Resource   string      `json:"resource"`
	Attributes []attribute `json:"attributes"`
}

func (a attribute) kindString() string {
	if a.Element != "" {
		return a.Kind + "<" + a.Element + ">"
	}
	return a.Kind
}

func readTree(path string) (attributeTree, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return attributeTree{}, err
	}
	var t attributeTree
	if err := json.Unmarshal(b, &t); err != nil {
		return attributeTree{}, fmt.Errorf("%s: %w", path, err)
	}
	if t.Format != treeFormat {
		return attributeTree{}, fmt.Errorf("%s: format %q, want %q", path, t.Format, treeFormat)
	}
	if t.Resource != dashboardResource {
		return attributeTree{}, fmt.Errorf("%s: resource %q, want %q", path, t.Resource, dashboardResource)
	}
	return t, nil
}
