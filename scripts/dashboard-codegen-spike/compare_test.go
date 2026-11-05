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

// The spike's verification: the generated slo_burn_rate code against the hand-written handler
// (internal/kibana/dashboard/panel/sloburnrate). Each test writes a golden file under testdata/
// that REPORT.md quotes; `go test ./scripts/dashboard-codegen-spike/ -update` rewrites them.
//
// The read test runs on Kibana responses recorded in testdata/kibana/. To record them again
// against a running Kibana:
//
//	CODEGEN_SPIKE_RECORD=1 KIBANA_ENDPOINT=http://localhost:5601 go test ./scripts/dashboard-codegen-spike/ -run TestReadBack

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/elastic/terraform-provider-elasticstack/generated/kbapi"
	// Registers the panel handlers, which sets panelkit's sibling config-block names.
	_ "github.com/elastic/terraform-provider-elasticstack/internal/kibana/dashboard"
	"github.com/elastic/terraform-provider-elasticstack/internal/kibana/dashboard/models"
	"github.com/elastic/terraform-provider-elasticstack/internal/kibana/dashboard/panel/sloburnrate"
	"github.com/elastic/terraform-provider-elasticstack/scripts/dashboard-codegen-spike/sloburnratecomputed"
	"github.com/elastic/terraform-provider-elasticstack/scripts/dashboard-codegen-spike/sloburnratepreserve"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "rewrite golden files")

func golden(t *testing.T, name, got string) {
	t.Helper()
	p := filepath.Join("testdata", name)
	if *update {
		require.NoError(t, os.WriteFile(p, []byte(got), 0o600))
		return
	}
	want, err := os.ReadFile(p)
	require.NoError(t, err)
	require.Equal(t, string(want), got, "golden %s differs; rerun with -update and review", p)
}

// ---------------------------------------------------------------------------------------------
// Cases: the same practitioner configuration, written for each implementation.

type spikeCase struct {
	name string
	hw   models.SloBurnRateConfigModel // hand-written: plan = configuration
	cfg  sloburnratepreserve.Model     // generated: configuration (= plan for the preserve strategy)
}

func sv(s string) types.String  { return types.StringValue(s) }
func boolean(b bool) types.Bool { return types.BoolValue(b) }

type dd struct {
	url, label     string
	encode, newTab *bool
}

func (d dd) hw() models.URLDrilldownModel {
	m := models.URLDrilldownModel{URL: sv(d.url), Label: sv(d.label), EncodeURL: types.BoolNull(), OpenInNewTab: types.BoolNull()}
	if d.encode != nil {
		m.EncodeURL = boolean(*d.encode)
	}
	if d.newTab != nil {
		m.OpenInNewTab = boolean(*d.newTab)
	}
	return m
}

func (d dd) gen() sloburnratepreserve.DrilldownsModel {
	h := d.hw()
	return sloburnratepreserve.DrilldownsModel{URLDrilldown: &sloburnratepreserve.DrilldownsURLDrilldownModel{
		URL: h.URL, Label: h.Label, EncodeURL: h.EncodeURL, OpenInNewTab: h.OpenInNewTab, Trigger: sv("on_open_panel_menu"),
	}}
}

func newCase(name string, set func(hw *models.SloBurnRateConfigModel), dds ...dd) spikeCase {
	hw := models.SloBurnRateConfigModel{
		SloID: sv("slo-1"), Duration: sv("5m"),
		SloInstanceID: types.StringNull(), Title: types.StringNull(), Description: types.StringNull(),
		HideTitle: types.BoolNull(), HideBorder: types.BoolNull(),
	}
	if set != nil {
		set(&hw)
	}
	cfg := sloburnratepreserve.Model{
		SloID: hw.SloID, Duration: hw.Duration, SloInstanceID: hw.SloInstanceID, Title: hw.Title,
		Description: hw.Description, HideTitle: hw.HideTitle, HideBorder: hw.HideBorder,
	}
	for _, d := range dds {
		hw.Drilldowns = append(hw.Drilldowns, d.hw())
		cfg.Drilldowns = append(cfg.Drilldowns, d.gen())
	}
	return spikeCase{name: name, hw: hw, cfg: cfg}
}

func cases() []spikeCase {
	return []spikeCase{
		newCase("minimal", nil),
		newCase("presentation", func(m *models.SloBurnRateConfigModel) {
			m.Title, m.Description, m.HideTitle, m.HideBorder = sv("Burn"), sv("About"), boolean(false), boolean(true)
		}),
		newCase("instance", func(m *models.SloBurnRateConfigModel) { m.SloInstanceID = sv("host-a") }),
		newCase("instance_star", func(m *models.SloBurnRateConfigModel) { m.SloInstanceID = sv("*") }),
		newCase("drilldowns", nil,
			dd{url: "https://a", label: "plain"},
			dd{url: "https://b", label: "off", encode: new(false), newTab: new(false)},
			dd{url: "https://c", label: "on", encode: new(true), newTab: new(true)},
		),
	}
}

// computedPlan is what Terraform plans for the computed strategy: the configuration with each
// schema Default filled in where the configuration left the attribute null.
func (c spikeCase) computedPlan() sloburnratecomputed.Model {
	orNull := func(v types.String, def string) types.String {
		if v.IsNull() {
			return sv(def)
		}
		return v
	}
	orNullB := func(v types.Bool, def bool) types.Bool {
		if v.IsNull() {
			return boolean(def)
		}
		return v
	}
	m := sloburnratecomputed.Model{
		SloID: c.cfg.SloID, Duration: c.cfg.Duration, SloInstanceID: orNull(c.cfg.SloInstanceID, "*"),
		Title: c.cfg.Title, Description: c.cfg.Description, HideTitle: c.cfg.HideTitle, HideBorder: c.cfg.HideBorder,
	}
	for _, d := range c.cfg.Drilldowns {
		u := d.URLDrilldown
		m.Drilldowns = append(m.Drilldowns, sloburnratecomputed.DrilldownsModel{URLDrilldown: &sloburnratecomputed.DrilldownsURLDrilldownModel{
			URL: u.URL, Label: u.Label, Trigger: u.Trigger,
			EncodeURL: orNullB(u.EncodeURL, true), OpenInNewTab: orNullB(u.OpenInNewTab, true),
		}})
	}
	return m
}

// ---------------------------------------------------------------------------------------------
// Schema

func TestSchemaVersusHandWritten(t *testing.T) {
	ctx := context.Background()
	hwBlock, ok := sloburnrate.SchemaAttribute().(schema.SingleNestedAttribute)
	require.True(t, ok)
	hw := map[string]schemaLine{}
	walkSchema(ctx, "", hwBlock.Attributes, hw)
	gen := map[string]schemaLine{}
	walkSchema(ctx, "", sloburnratecomputed.Attributes(), gen)

	paths := map[string]bool{}
	for p := range hw {
		paths[p] = true
	}
	for p := range gen {
		paths[p] = true
	}
	sorted := make([]string, 0, len(paths))
	for p := range paths {
		sorted = append(sorted, p)
	}
	sort.Strings(sorted)

	var structure, descriptions, same []string
	for _, p := range sorted {
		h, inH := hw[p]
		g, inG := gen[p]
		switch {
		case !inG:
			structure = append(structure, fmt.Sprintf("- only hand-written  %s: %s", p, h.shape))
		case !inH:
			structure = append(structure, fmt.Sprintf("+ only generated     %s: %s", p, g.shape))
		default:
			if h.shape != g.shape {
				structure = append(structure, fmt.Sprintf("~ differs            %s\n    hand-written: %s\n    generated:    %s", p, h.shape, g.shape))
			} else {
				same = append(same, fmt.Sprintf("= same               %s: %s", p, h.shape))
			}
			if h.description != g.description {
				descriptions = append(descriptions, fmt.Sprintf("%s\n    hand-written: %q\n    generated:    %q", p, h.description, g.description))
			}
		}
	}
	var b strings.Builder
	b.WriteString("# Schema: generated (computed strategy) versus hand-written slo_burn_rate_config\n")
	b.WriteString("# The preserve strategy differs from computed only in dropping Computed and Default.\n\n")
	b.WriteString("## Same\n")
	b.WriteString(strings.Join(same, "\n") + "\n\n## Structure\n")
	b.WriteString(strings.Join(structure, "\n") + "\n\n## Descriptions\n")
	b.WriteString(strings.Join(descriptions, "\n") + "\n")
	golden(t, "schema_diff.golden", b.String())
}

type schemaLine struct{ shape, description string }

func walkSchema(ctx context.Context, prefix string, attrs map[string]schema.Attribute, out map[string]schemaLine) {
	for name, a := range attrs {
		p := strings.TrimPrefix(prefix+"."+name, ".")
		mode := "optional"
		switch {
		case a.IsRequired():
			mode = "required"
		case a.IsOptional() && a.IsComputed():
			mode = "optional+computed"
		case a.IsComputed():
			mode = "computed"
		}
		var kind, def string
		var vals []string
		switch v := a.(type) {
		case schema.StringAttribute:
			kind = "string"
			if v.Default != nil {
				def = v.Default.Description(ctx)
			}
			for _, x := range v.Validators {
				vals = append(vals, x.Description(ctx))
			}
		case schema.BoolAttribute:
			kind = "bool"
			if v.Default != nil {
				def = v.Default.Description(ctx)
			}
		case schema.SingleNestedAttribute:
			kind = "object"
			walkSchema(ctx, p, v.Attributes, out)
		case schema.ListNestedAttribute:
			kind = "list(object)"
			for _, x := range v.Validators {
				vals = append(vals, x.Description(ctx))
			}
			walkSchema(ctx, p+"[]", v.NestedObject.Attributes, out)
		default:
			kind = fmt.Sprintf("%T", a)
		}
		shape := kind + ", " + mode
		if def != "" {
			shape += ", default: " + def
		}
		if len(vals) > 0 {
			sort.Strings(vals)
			shape += ", validators: " + strings.Join(vals, "; ")
		}
		out[p] = schemaLine{shape: shape, description: a.GetMarkdownDescription()}
	}
}

// ---------------------------------------------------------------------------------------------
// Write: the API config each implementation sends for the same configuration.

func handWrittenRequest(t *testing.T, c spikeCase) json.RawMessage {
	t.Helper()
	var panel kbapi.KibanaHTTPAPIsKbnDashboardPanelTypeSloBurnRate
	hw := c.hw
	diags := sloburnrate.BuildConfig(models.PanelModel{SloBurnRateConfig: &hw}, &panel)
	require.False(t, diags.HasError(), "%v", diags)
	return canonicalJSON(t, panel.Config)
}

func computedRequest(t *testing.T, c spikeCase) json.RawMessage {
	t.Helper()
	api, diags := sloburnratecomputed.ToAPI(c.computedPlan())
	require.False(t, diags.HasError(), "%v", diags)
	return canonicalJSON(t, api)
}

func preserveRequest(t *testing.T, c spikeCase) json.RawMessage {
	t.Helper()
	api, diags := sloburnratepreserve.ToAPI(c.cfg)
	require.False(t, diags.HasError(), "%v", diags)
	return canonicalJSON(t, api)
}

func canonicalJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	var m any
	require.NoError(t, json.Unmarshal(b, &m))
	out, err := json.Marshal(m) // maps marshal with sorted keys
	require.NoError(t, err)
	return out
}

func TestWriteVersusHandWritten(t *testing.T) {
	var b strings.Builder
	b.WriteString("# Write: the config object each implementation sends for the same configuration\n\n")
	for _, c := range cases() {
		hw, comp, pres := handWrittenRequest(t, c), computedRequest(t, c), preserveRequest(t, c)
		fmt.Fprintf(&b, "## %s\nhand-written: %s\n", c.name, hw)
		for _, x := range []struct {
			name string
			got  json.RawMessage
		}{{"computed", comp}, {"preserve", pres}} {
			if bytes.Equal(hw, x.got) {
				fmt.Fprintf(&b, "%-13s same\n", x.name+":")
			} else {
				fmt.Fprintf(&b, "%-13s %s\n", x.name+":", x.got)
			}
		}
		b.WriteString("\n")
	}
	golden(t, "write.golden", b.String())
}

// ---------------------------------------------------------------------------------------------
// Read: what each implementation puts in state from Kibana's response to its own request.

type recorded struct {
	KibanaVersion string          `json:"kibana_version"`
	Request       json.RawMessage `json:"request"`
	Response      json.RawMessage `json:"response"`
}

func TestReadBack(t *testing.T) {
	record := os.Getenv("CODEGEN_SPIKE_RECORD") == "1"
	var b strings.Builder
	b.WriteString("# Read: state after apply (prior = plan) and after import (no prior), compared with the plan\n")
	b.WriteString("# \"consistent\" means equal to the plan: no \"inconsistent result after apply\" and, for import,\n")
	b.WriteString("# no diff against the configuration. Responses: testdata/kibana/, recorded from a live Kibana.\n\n")
	b.WriteString("| case | implementation | after apply | after import |\n| --- | --- | --- | --- |\n")
	for _, c := range cases() {
		impls := []struct {
			name    string
			request json.RawMessage
			check   func(resp kbapi.KibanaHTTPAPIsSloBurnRateEmbeddable) (apply, imp []string)
		}{
			{"hand-written", handWrittenRequest(t, c), func(resp kbapi.KibanaHTTPAPIsSloBurnRateEmbeddable) ([]string, []string) {
				plan := c.hw
				var applied, imported models.PanelModel
				require.False(t, sloburnrate.PopulateFromAPI(&applied, &models.PanelModel{SloBurnRateConfig: &plan}, resp).HasError())
				require.False(t, sloburnrate.PopulateFromAPI(&imported, nil, resp).HasError())
				return modelDiff(plan, *applied.SloBurnRateConfig), modelDiff(plan, *imported.SloBurnRateConfig)
			}},
			{"generated computed", computedRequest(t, c), func(resp kbapi.KibanaHTTPAPIsSloBurnRateEmbeddable) ([]string, []string) {
				plan := c.computedPlan()
				got, diags := sloburnratecomputed.FromAPI(resp)
				require.False(t, diags.HasError())
				// Import reads the same way: the computed strategy never consults prior state.
				d := modelDiff(plan, got)
				return d, d
			}},
			{"generated preserve", preserveRequest(t, c), func(resp kbapi.KibanaHTTPAPIsSloBurnRateEmbeddable) ([]string, []string) {
				plan := c.cfg
				applied, diags := sloburnratepreserve.FromAPI(resp, &plan)
				require.False(t, diags.HasError())
				imported, diags := sloburnratepreserve.FromAPI(resp, nil)
				require.False(t, diags.HasError())
				return modelDiff(plan, applied), modelDiff(plan, imported)
			}},
			// Control: the released (Optional) schema with a plain read and no default handling at
			// all. What differs here is exactly what any preserve logic has to handle.
			{"control: plain read", preserveRequest(t, c), func(resp kbapi.KibanaHTTPAPIsSloBurnRateEmbeddable) ([]string, []string) {
				got, diags := sloburnratecomputed.FromAPI(resp)
				require.False(t, diags.HasError())
				d := modelDiff(c.cfg, got)
				return d, d
			}},
		}
		for _, impl := range impls {
			fixture := filepath.Join("testdata", "kibana", c.name+"."+slug(impl.name, "_")+".json")
			if record {
				recordResponse(t, fixture, c.name+"-"+slug(impl.name, "-"), impl.request)
			}
			var rec recorded
			raw, err := os.ReadFile(fixture)
			require.NoError(t, err, "no recorded response; record with CODEGEN_SPIKE_RECORD=1")
			require.NoError(t, json.Unmarshal(raw, &rec))
			require.JSONEq(t, string(rec.Request), string(impl.request), "%s: the request changed since it was recorded; record again", fixture)

			var resp kbapi.KibanaHTTPAPIsSloBurnRateEmbeddable
			require.NoError(t, json.Unmarshal(rec.Response, &resp))
			apply, imp := impl.check(resp)
			fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", c.name, impl.name, verdict(apply), verdict(imp))
		}
	}
	golden(t, "read.golden", b.String())
}

// slug makes an implementation name usable in a file name or dashboard ID.
func slug(s, sep string) string {
	return strings.Join(strings.FieldsFunc(s, func(r rune) bool { return r == ' ' || r == ':' }), sep)
}

func verdict(diffs []string) string {
	if len(diffs) == 0 {
		return "consistent"
	}
	return "differs: " + strings.Join(diffs, "; ")
}

// modelDiff lists the attributes whose values differ, as `path: want → got`.
func modelDiff(want, got any) []string {
	w, g := map[string]string{}, map[string]string{}
	flatten("", reflect.ValueOf(want), w)
	flatten("", reflect.ValueOf(got), g)
	keys := map[string]bool{}
	for k := range w {
		keys[k] = true
	}
	for k := range g {
		keys[k] = true
	}
	var out []string
	for k := range keys {
		wv, gv := w[k], g[k]
		if wv == "" {
			wv = "<absent>"
		}
		if gv == "" {
			gv = "<absent>"
		}
		if wv != gv {
			out = append(out, fmt.Sprintf("%s: %s → %s", k, wv, gv))
		}
	}
	sort.Strings(out)
	return out
}

var attrValue = reflect.TypeFor[attr.Value]()

func flatten(prefix string, v reflect.Value, out map[string]string) {
	if v.Type().Implements(attrValue) {
		out[prefix] = v.Interface().(attr.Value).String()
		return
	}
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			out[prefix] = "<null>"
			return
		}
		flatten(prefix, v.Elem(), out)
	case reflect.Slice:
		if v.IsNil() {
			out[prefix] = "<null>"
			return
		}
		for i := range v.Len() {
			flatten(fmt.Sprintf("%s[%d]", prefix, i), v.Index(i), out)
		}
	case reflect.Struct:
		for i := range v.NumField() {
			tag := v.Type().Field(i).Tag.Get("tfsdk")
			flatten(strings.TrimPrefix(prefix+"."+tag, "."), v.Field(i), out)
		}
	}
}

// recordResponse puts a dashboard with one slo_burn_rate panel holding config, reads it back, and
// writes the request and Kibana's config to fixture.
func recordResponse(t *testing.T, fixture, id string, config json.RawMessage) {
	t.Helper()
	k := newKibana(t)
	dashboardID := "codegen-spike-" + strings.ReplaceAll(id, "_", "-")
	require.NoError(t, k.putPanel(dashboardID, config))
	resp, found, err := k.getPanelConfig(dashboardID)
	require.NoError(t, err)
	require.True(t, found)
	require.NoError(t, k.delete(dashboardID))
	b, err := json.MarshalIndent(recorded{KibanaVersion: k.version(t), Request: config, Response: resp}, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(fixture), 0o750))
	require.NoError(t, os.WriteFile(fixture, append(b, '\n'), 0o600))
}

// kibana is the spike's minimal Dashboards API client.
type kibana struct {
	endpoint, user, password string
}

func newKibana(t *testing.T) kibana {
	t.Helper()
	k := kibana{os.Getenv("KIBANA_ENDPOINT"), os.Getenv("KIBANA_USERNAME"), os.Getenv("KIBANA_PASSWORD")}
	if k.endpoint == "" {
		t.Skip("KIBANA_ENDPOINT is not set")
	}
	return k
}

func (k kibana) do(method, path string, body any) (int, []byte, error) {
	var rd *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		rd = bytes.NewReader(b)
	} else {
		rd = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, strings.TrimRight(k.endpoint, "/")+path, rd)
	if err != nil {
		return 0, nil, err
	}
	req.SetBasicAuth(k.user, k.password)
	req.Header.Set("kbn-xsrf", "true")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	var buf bytes.Buffer
	_, err = buf.ReadFrom(resp.Body)
	return resp.StatusCode, buf.Bytes(), err
}

func (k kibana) putPanel(id string, config json.RawMessage) error {
	body := map[string]any{
		"title": "codegen spike " + id,
		"panels": []any{map[string]any{
			"type":   "slo_burn_rate",
			"grid":   map[string]any{"x": 0, "y": 0, "w": 24, "h": 10},
			"config": config,
		}},
	}
	status, b, err := k.do(http.MethodPut, "/api/dashboards/"+id, body)
	if err != nil {
		return err
	}
	if status >= 300 {
		return fmt.Errorf("PUT dashboard %s: %d: %s", id, status, b)
	}
	return nil
}

func (k kibana) getPanelConfig(id string) (json.RawMessage, bool, error) {
	status, b, err := k.do(http.MethodGet, "/api/dashboards/"+id, nil)
	if err != nil {
		return nil, false, err
	}
	if status == http.StatusNotFound {
		return nil, false, nil
	}
	if status >= 300 {
		return nil, false, fmt.Errorf("GET dashboard %s: %d: %s", id, status, b)
	}
	var d struct {
		Data struct {
			Panels []struct {
				Config json.RawMessage `json:"config"`
			} `json:"panels"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, false, err
	}
	if len(d.Data.Panels) != 1 {
		return nil, false, fmt.Errorf("dashboard %s has %d panels", id, len(d.Data.Panels))
	}
	return d.Data.Panels[0].Config, true, nil
}

func (k kibana) delete(id string) error {
	status, b, err := k.do(http.MethodDelete, "/api/dashboards/"+id, nil)
	if err != nil {
		return err
	}
	if status >= 300 && status != http.StatusNotFound {
		return fmt.Errorf("DELETE dashboard %s: %d: %s", id, status, b)
	}
	return nil
}

func (k kibana) version(t *testing.T) string {
	t.Helper()
	_, b, err := k.do(http.MethodGet, "/api/status", nil)
	require.NoError(t, err)
	var s struct {
		Version struct {
			Number string `json:"number"`
		} `json:"version"`
	}
	require.NoError(t, json.Unmarshal(b, &s))
	return s.Version.Number
}
