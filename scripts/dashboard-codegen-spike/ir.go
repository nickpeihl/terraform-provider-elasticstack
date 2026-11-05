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
	"maps"
	"sort"
	"strings"
)

// Attribute kinds the spike generates.
const (
	kindString  = "string"
	kindBool    = "bool"
	kindInt64   = "int64"
	kindFloat64 = "float64"
	kindObject  = "object"
	kindList    = "list" // a list of objects
)

// Modes, derived from the OpenAPI document alone:
//   - required: in the parent's `required` and without a `default`
//   - default: has a `default` (Kibana fills it when absent and returns it on read)
//   - optional: everything else
const (
	modeRequired = "required"
	modeDefault  = "default"
	modeOptional = "optional"
)

// attribute is one Terraform attribute, built from one OpenAPI property.
type attribute struct {
	Name        string // Terraform and API name (the spike keeps them equal)
	Kind        string
	Mode        string
	Default     any // set when Mode is modeDefault
	Description string

	MaxLength, MinLength *int
	MaxItems, MinItems   *int
	Enum                 []string

	// Object: the nested attributes; List: the element object's attributes.
	Attributes []*attribute
	// Union, when the object (or the list's element object) is an x-tf-union: one optional
	// nested attribute per variant, Attributes holds those, and the discriminator is not an attribute.
	Union *union
}

type union struct {
	Discriminator string
	Variants      []variant // sorted by Key
}

type variant struct {
	Value string     // discriminator value in the API
	Key   string     // Terraform attribute name, from key_template
	Attr  *attribute // the variant's object attribute (also in the parent's Attributes)
}

// unsupported collects the constructs the spike does not generate, with where they occur.
type unsupported []string

func (u *unsupported) add(path, what string) { *u = append(*u, path+": "+what) }

// builder turns OpenAPI schemas into attributes.
type builder struct {
	s    schemas
	errs unsupported
}

// supportedUnionFields lists the x-tf-union fields the spike implements.
var supportedUnionFields = map[string]bool{"discriminator": true, "key_template": true}

func (b *builder) objectAttributes(path string, node map[string]any) []*attribute {
	node, err := b.s.resolve(node)
	if err != nil {
		b.errs.add(path, err.Error())
		return nil
	}
	b.checkXTF(path, node, "x-tf-union")
	props, _ := node["properties"].(map[string]any)
	if props == nil {
		b.errs.add(path, "object without properties")
		return nil
	}
	required := map[string]bool{}
	if rs, ok := node["required"].([]any); ok {
		for _, r := range rs {
			if s, ok := r.(string); ok {
				required[s] = true
			}
		}
	}
	if ap, ok := node["additionalProperties"]; ok && ap != false {
		b.errs.add(path, "additionalProperties (a map)")
	}
	names := make([]string, 0, len(props))
	for k := range props {
		names = append(names, k)
	}
	sort.Strings(names)
	var out []*attribute
	for _, name := range names {
		p, _ := props[name].(map[string]any)
		if a := b.attribute(path+"."+name, name, p, required[name]); a != nil {
			out = append(out, a)
		}
	}
	return out
}

func (b *builder) attribute(path, name string, node map[string]any, required bool) *attribute {
	node, err := b.s.resolve(node)
	if err != nil {
		b.errs.add(path, err.Error())
		return nil
	}
	b.checkXTF(path, node, "x-tf-union")
	a := &attribute{Name: name, Mode: modeOptional}
	if d, ok := node["description"].(string); ok {
		a.Description = d
	}
	if def, ok := node["default"]; ok {
		a.Mode, a.Default = modeDefault, def
	} else if required {
		a.Mode = modeRequired
	}

	// anyOf whose members are all string enums is one string enum (Zod's z.literal union).
	if members, ok := node["anyOf"].([]any); ok {
		enum, ok := stringEnums(members)
		if !ok {
			b.errs.add(path, "anyOf without x-tf-union")
			return nil
		}
		a.Kind, a.Enum = kindString, enum
		return a
	}
	if _, ok := node["oneOf"]; ok {
		b.errs.add(path, "oneOf")
		return nil
	}
	if _, ok := node["allOf"]; ok {
		b.errs.add(path, "allOf")
		return nil
	}
	if node["nullable"] == true {
		b.errs.add(path, "nullable")
	}

	switch t, _ := node["type"].(string); t {
	case "string": //nolint:goconst // the OpenAPI type name, not the attribute kind
		a.Kind = kindString
		a.MaxLength, a.MinLength = intPtr(node["maxLength"]), intPtr(node["minLength"])
		if e, ok := node["enum"].([]any); ok {
			a.Enum, _ = stringEnums([]any{map[string]any{"type": "string", "enum": e}})
		}
	case "boolean":
		a.Kind = kindBool
	case "integer":
		a.Kind = kindInt64
	case "number":
		a.Kind = kindFloat64
	case "object":
		a.Kind = kindObject
		a.Attributes = b.objectAttributes(path, node)
	case "array":
		items, _ := node["items"].(map[string]any)
		items, err := b.s.resolve(items)
		if err != nil {
			b.errs.add(path, err.Error())
			return nil
		}
		a.Kind = kindList
		a.MaxItems, a.MinItems = intPtr(node["maxItems"]), intPtr(node["minItems"])
		if u, ok := items["x-tf-union"].(map[string]any); ok {
			a.Attributes, a.Union = b.union(path+"[]", items, u)
			return a
		}
		if it, _ := items["type"].(string); it != "object" {
			b.errs.add(path, "list of "+fmt.Sprint(items["type"]))
			return nil
		}
		a.Attributes = b.objectAttributes(path+"[]", items)
	default:
		if u, ok := node["x-tf-union"].(map[string]any); ok {
			a.Kind = kindObject
			a.Attributes, a.Union = b.union(path, node, u)
			return a
		}
		b.errs.add(path, fmt.Sprintf("type %v", node["type"]))
		return nil
	}
	return a
}

// union builds the variant attributes of an x-tf-union over anyOf objects.
func (b *builder) union(path string, node, meta map[string]any) ([]*attribute, *union) {
	for k := range meta {
		if !supportedUnionFields[k] {
			b.errs.add(path, "x-tf-union."+k)
		}
	}
	disc, _ := meta["discriminator"].(string)
	tmpl, _ := meta["key_template"].(string)
	if tmpl == "" {
		tmpl = "{value}"
	}
	members, _ := node["anyOf"].([]any)
	if len(members) == 0 {
		members, _ = node["oneOf"].([]any)
	}
	u := &union{Discriminator: disc}
	var attrs []*attribute
	for _, m := range members {
		mn, _ := m.(map[string]any)
		mn, err := b.s.resolve(mn)
		if err != nil {
			b.errs.add(path, err.Error())
			continue
		}
		props, _ := mn["properties"].(map[string]any)
		dp, _ := props[disc].(map[string]any)
		values, ok := stringEnums([]any{dp})
		if !ok || len(values) != 1 {
			b.errs.add(path, "union member without a single discriminator value")
			continue
		}
		key := strings.ReplaceAll(tmpl, "{value}", values[0])
		// The variant object is the member without its discriminator property.
		trimmed := maps.Clone(mn)
		tp := map[string]any{}
		for k, v := range props {
			if k != disc {
				tp[k] = v
			}
		}
		trimmed["properties"] = tp
		va := &attribute{Name: key, Kind: kindObject, Mode: modeOptional, Description: str(mn["description"])}
		va.Attributes = b.objectAttributes(path+"."+key, trimmed)
		u.Variants = append(u.Variants, variant{Value: values[0], Key: key, Attr: va})
		attrs = append(attrs, va)
	}
	sort.Slice(u.Variants, func(i, j int) bool { return u.Variants[i].Key < u.Variants[j].Key })
	sort.Slice(attrs, func(i, j int) bool { return attrs[i].Name < attrs[j].Name })
	// A union with one variant: the variant is required, so the element cannot be empty.
	if len(attrs) == 1 {
		attrs[0].Mode = modeRequired
	}
	return attrs, u
}

// checkXTF records x-tf keys on node that the spike does not implement.
func (b *builder) checkXTF(path string, node map[string]any, allowed ...string) {
	for k := range node {
		if !strings.HasPrefix(k, "x-tf-") {
			continue
		}
		ok := false
		for _, a := range allowed {
			ok = ok || a == k
		}
		if !ok {
			b.errs.add(path, k)
		}
	}
}

func stringEnums(members []any) ([]string, bool) {
	var out []string
	for _, m := range members {
		mn, _ := m.(map[string]any)
		if mn == nil || mn["type"] != "string" {
			return nil, false
		}
		e, _ := mn["enum"].([]any)
		if len(e) == 0 {
			return nil, false
		}
		for _, v := range e {
			s, ok := v.(string)
			if !ok {
				return nil, false
			}
			out = append(out, s)
		}
	}
	return out, len(out) > 0
}

func intPtr(v any) *int {
	switch n := v.(type) {
	case int:
		return &n
	case float64:
		i := int(n)
		return &i
	}
	return nil
}

func str(v any) string {
	s, _ := v.(string)
	return s
}
