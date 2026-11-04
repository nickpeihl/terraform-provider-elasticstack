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
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/elastic/terraform-provider-elasticstack/internal/kibana/dashboard"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

const dashboardResource = "elasticstack_kibana_dashboard"

// providerTree walks the provider's real dashboard schema into the same attribute-tree shape Kibana derives. Blocks
// (`kibana_connection`) are provider plumbing, not part of the Dashboards API, and are left out.
func providerTree(ctx context.Context) (attributeTree, error) {
	var resp resource.SchemaResponse
	dashboard.NewResource().Schema(ctx, resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		return attributeTree{}, fmt.Errorf("dashboard schema: %v", resp.Diagnostics)
	}
	var attrs []attribute
	walkAttributes("", resp.Schema.Attributes, &attrs)
	sort.Slice(attrs, func(i, j int) bool { return attrs[i].Path < attrs[j].Path })
	return attributeTree{Format: treeFormat, Resource: dashboardResource, Attributes: attrs}, nil
}

func walkAttributes(parent string, attrs map[string]schema.Attribute, out *[]attribute) {
	for name, a := range attrs {
		p := name
		if parent != "" {
			p = parent + "." + name
		}
		node := attribute{Path: p, Mode: modeOf(a)}
		var children map[string]schema.Attribute
		childPrefix := p
		switch at := a.(type) {
		case schema.StringAttribute:
			node.Kind = stringKind(at.CustomType)
		case schema.BoolAttribute:
			node.Kind = kindBool
		case schema.Int64Attribute, schema.Int32Attribute, schema.Float64Attribute, schema.Float32Attribute, schema.NumberAttribute:
			node.Kind = kindNumber
		case schema.ListAttribute:
			node.Kind, node.Element = kindList, elementKind(at.ElementType)
		case schema.SetAttribute:
			node.Kind, node.Element = kindSet, elementKind(at.ElementType)
		case schema.MapAttribute:
			node.Kind, node.Element = kindMap, elementKind(at.ElementType)
		case schema.ObjectAttribute:
			node.Kind = kindObject
		case schema.SingleNestedAttribute:
			node.Kind, children = kindObject, at.Attributes
		case schema.ListNestedAttribute:
			node.Kind, node.Element, children, childPrefix = kindList, kindObject, at.NestedObject.Attributes, p+"[]"
		case schema.SetNestedAttribute:
			node.Kind, node.Element, children, childPrefix = kindSet, kindObject, at.NestedObject.Attributes, p+"[]"
		case schema.MapNestedAttribute:
			node.Kind, node.Element, children, childPrefix = kindMap, kindObject, at.NestedObject.Attributes, p+"{}"
		default:
			node.Kind = fmt.Sprintf("unknown(%T)", a)
		}
		*out = append(*out, node)
		if children != nil {
			walkAttributes(childPrefix, children, out)
		}
	}
}

func modeOf(a schema.Attribute) string {
	switch {
	case a.IsRequired():
		return "required"
	case a.IsOptional() && a.IsComputed():
		return "optional_computed"
	case a.IsOptional():
		return "optional"
	default:
		return "computed"
	}
}

// stringKind tells a string holding JSON (jsontypes.Normalized, JSONWithDefaults, …) from a plain string.
func stringKind(custom basetypes.StringTypable) string {
	if custom != nil && strings.Contains(strings.ToLower(fmt.Sprintf("%T", custom)), "json") {
		return kindJSONString
	}
	return kindString
}

func elementKind(t attr.Type) string {
	if t == nil {
		return ""
	}
	if _, ok := t.(basetypes.ObjectTypable); ok {
		return kindObject
	}
	s := strings.ToLower(fmt.Sprintf("%T %s", t, t.String()))
	switch {
	case strings.Contains(s, "json"):
		return kindJSONString
	case strings.Contains(s, "string"):
		return kindString
	case strings.Contains(s, "bool"):
		return kindBool
	case strings.Contains(s, "int"), strings.Contains(s, "float"), strings.Contains(s, "number"):
		return kindNumber
	}
	return t.String()
}
