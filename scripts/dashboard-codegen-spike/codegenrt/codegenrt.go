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

// Package codegenrt is the small hand-written runtime that the spike's generated panel code calls.
// Generated code converts between the Terraform model and the API's JSON object
// (map[string]any); these helpers keep that code short. Spike code: not used by the provider.
package codegenrt

import (
	"bytes"
	"encoding/json"

	"github.com/elastic/terraform-provider-elasticstack/internal/utils/typeutils"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// PutString writes v under k when it is known.
func PutString(o map[string]any, k string, v types.String) {
	if typeutils.IsKnown(v) {
		o[k] = v.ValueString()
	}
}

// PutBool writes v under k when it is known.
func PutBool(o map[string]any, k string, v types.Bool) {
	if typeutils.IsKnown(v) {
		o[k] = v.ValueBool()
	}
}

// PutInt64 writes v under k when it is known.
func PutInt64(o map[string]any, k string, v types.Int64) {
	if typeutils.IsKnown(v) {
		o[k] = v.ValueInt64()
	}
}

// PutFloat64 writes v under k when it is known.
func PutFloat64(o map[string]any, k string, v types.Float64) {
	if typeutils.IsKnown(v) {
		o[k] = v.ValueFloat64()
	}
}

// String reads k as a string; absent or another type reads as null.
func String(o map[string]any, k string) types.String {
	s, ok := o[k].(string)
	if !ok {
		return types.StringNull()
	}
	return types.StringValue(s)
}

// Bool reads k as a bool; absent or another type reads as null.
func Bool(o map[string]any, k string) types.Bool {
	b, ok := o[k].(bool)
	if !ok {
		return types.BoolNull()
	}
	return types.BoolValue(b)
}

// Int64 reads k as an integer; absent or another type reads as null.
func Int64(o map[string]any, k string) types.Int64 {
	n, ok := o[k].(json.Number)
	if !ok {
		return types.Int64Null()
	}
	i, err := n.Int64()
	if err != nil {
		return types.Int64Null()
	}
	return types.Int64Value(i)
}

// Float64 reads k as a number; absent or another type reads as null.
func Float64(o map[string]any, k string) types.Float64 {
	n, ok := o[k].(json.Number)
	if !ok {
		return types.Float64Null()
	}
	f, err := n.Float64()
	if err != nil {
		return types.Float64Null()
	}
	return types.Float64Value(f)
}

// Object reads k as a JSON object.
func Object(o map[string]any, k string) (map[string]any, bool) {
	v, ok := o[k].(map[string]any)
	return v, ok
}

// List reads k as a JSON array.
func List(o map[string]any, k string) ([]any, bool) {
	v, ok := o[k].([]any)
	return v, ok
}

// The Defaulted* helpers are the "preserve" strategy's one generic rule. A field that the OpenAPI
// document marks with a `default` reads as null when the practitioner did not set it (prior is
// null, which includes import, where there is no prior) and the API returned exactly that default.
// Any other value is kept, so a changed value shows as drift.

// DefaultedString applies the default-aware read rule to a string.
func DefaultedString(v, prior types.String, def string) types.String {
	if !typeutils.IsKnown(prior) && typeutils.IsKnown(v) && v.ValueString() == def {
		return types.StringNull()
	}
	return v
}

// DefaultedBool applies the default-aware read rule to a bool.
func DefaultedBool(v, prior types.Bool, def bool) types.Bool {
	if !typeutils.IsKnown(prior) && typeutils.IsKnown(v) && v.ValueBool() == def {
		return types.BoolNull()
	}
	return v
}

// DefaultedInt64 applies the default-aware read rule to an integer.
func DefaultedInt64(v, prior types.Int64, def int64) types.Int64 {
	if !typeutils.IsKnown(prior) && typeutils.IsKnown(v) && v.ValueInt64() == def {
		return types.Int64Null()
	}
	return v
}

// DefaultedFloat64 applies the default-aware read rule to a number.
func DefaultedFloat64(v, prior types.Float64, def float64) types.Float64 {
	if !typeutils.IsKnown(prior) && typeutils.IsKnown(v) && v.ValueFloat64() == def {
		return types.Float64Null()
	}
	return v
}

// Remarshal converts in to out through JSON, decoding numbers as json.Number.
func Remarshal(in, out any) diag.Diagnostics {
	b, err := json.Marshal(in)
	if err != nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("Failed to encode panel config", err.Error())}
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	if err := dec.Decode(out); err != nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("Failed to decode panel config", err.Error())}
	}
	return nil
}
