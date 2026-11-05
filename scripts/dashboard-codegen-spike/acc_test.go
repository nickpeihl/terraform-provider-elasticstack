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

// End to end through Terraform: a throwaway provider whose only resources are one dashboard with
// one slo_burn_rate panel, its config attribute being the generated schema, written and read by the
// generated conversion. Terraform itself checks that each apply reads back as planned and that the
// plan after it is empty; the import steps check that import reads back the configuration.
//
//	CODEGEN_SPIKE_ACC=1 TF_ACC=1 KIBANA_ENDPOINT=http://localhost:5601 go test ./scripts/dashboard-codegen-spike/ -run TestAccGenerated -v
//
// It needs its own opt-in: `make testacc` runs every package with TF_ACC=1, and this test needs a Kibana
// with the Dashboards API.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/elastic/terraform-provider-elasticstack/generated/kbapi"
	"github.com/elastic/terraform-provider-elasticstack/scripts/dashboard-codegen-spike/sloburnratecomputed"
	"github.com/elastic/terraform-provider-elasticstack/scripts/dashboard-codegen-spike/sloburnratepreserve"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

type embeddable = kbapi.KibanaHTTPAPIsSloBurnRateEmbeddable

// panelResource is one dashboard holding one slo_burn_rate panel whose config is M.
type panelResource[M any] struct {
	name    string
	attrs   func() map[string]schema.Attribute
	toAPI   func(M) (embeddable, diag.Diagnostics)
	fromAPI func(api embeddable, prior *M) (M, diag.Diagnostics)
	kb      kibana
}

type panelState[M any] struct {
	ID     types.String `tfsdk:"id"`
	Config *M           `tfsdk:"config"`
}

func (r *panelResource[M]) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.name
}

func (r *panelResource[M]) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Required:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"config": schema.SingleNestedAttribute{Required: true, Attributes: r.attrs()},
	}}
}

func (r *panelResource[M]) write(ctx context.Context, plan panelState[M], diags *diag.Diagnostics) *panelState[M] {
	api, d := r.toAPI(*plan.Config)
	diags.Append(d...)
	if diags.HasError() {
		return nil
	}
	b, err := json.Marshal(api)
	if err != nil {
		diags.AddError("encode", err.Error())
		return nil
	}
	if err := r.kb.putPanel(plan.ID.ValueString(), b); err != nil {
		diags.AddError("put", err.Error())
		return nil
	}
	return r.read(ctx, plan.ID.ValueString(), plan.Config, diags)
}

func (r *panelResource[M]) read(_ context.Context, id string, prior *M, diags *diag.Diagnostics) *panelState[M] {
	raw, found, err := r.kb.getPanelConfig(id)
	if err != nil {
		diags.AddError("get", err.Error())
		return nil
	}
	if !found {
		return nil
	}
	var api embeddable
	if err := json.Unmarshal(raw, &api); err != nil {
		diags.AddError("decode", err.Error())
		return nil
	}
	m, d := r.fromAPI(api, prior)
	diags.Append(d...)
	return &panelState[M]{ID: types.StringValue(id), Config: &m}
}

func (r *panelResource[M]) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan panelState[M]
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if s := r.write(ctx, plan, &resp.Diagnostics); s != nil {
		resp.Diagnostics.Append(resp.State.Set(ctx, s)...)
	}
}

func (r *panelResource[M]) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan panelState[M]
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if s := r.write(ctx, plan, &resp.Diagnostics); s != nil {
		resp.Diagnostics.Append(resp.State.Set(ctx, s)...)
	}
}

func (r *panelResource[M]) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state panelState[M]
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	// On import, state holds only the ID: Config is nil, so the read has no prior.
	s := r.read(ctx, state.ID.ValueString(), state.Config, &resp.Diagnostics)
	if s == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, s)...)
}

func (r *panelResource[M]) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state panelState[M]
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if err := r.kb.delete(state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("delete", err.Error())
	}
}

func (r *panelResource[M]) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type spikeProvider struct{ kb kibana }

func (p *spikeProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "spike"
}
func (p *spikeProvider) Schema(context.Context, provider.SchemaRequest, *provider.SchemaResponse) {}
func (p *spikeProvider) Configure(context.Context, provider.ConfigureRequest, *provider.ConfigureResponse) {
}
func (p *spikeProvider) DataSources(context.Context) []func() datasource.DataSource { return nil }

func (p *spikeProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		func() resource.Resource {
			return &panelResource[sloburnratecomputed.Model]{
				name: "computed", attrs: sloburnratecomputed.Attributes, toAPI: sloburnratecomputed.ToAPI, kb: p.kb,
				fromAPI: func(api embeddable, _ *sloburnratecomputed.Model) (sloburnratecomputed.Model, diag.Diagnostics) {
					return sloburnratecomputed.FromAPI(api)
				},
			}
		},
		func() resource.Resource {
			return &panelResource[sloburnratepreserve.Model]{
				name: "preserve", attrs: sloburnratepreserve.Attributes, toAPI: sloburnratepreserve.ToAPI, kb: p.kb,
				fromAPI: sloburnratepreserve.FromAPI,
			}
		},
	}
}

func TestAccGeneratedSloBurnRate(t *testing.T) {
	if os.Getenv("CODEGEN_SPIKE_ACC") != "1" {
		t.Skip("spike test; set CODEGEN_SPIKE_ACC=1 to run it")
	}
	kb := newKibana(t)
	factories := map[string]func() (tfprotov6.ProviderServer, error){
		"spike": providerserver.NewProtocol6WithError(&spikeProvider{kb: kb}),
	}
	minimal := `slo_id = "slo-1"
    duration = "5m"`
	full := minimal + `
    title = "Burn"
    description = "About"
    hide_title = false
    hide_border = true
    slo_instance_id = "host-a"
    drilldowns = [
      { url_drilldown = { url = "https://a", label = "plain", trigger = "on_open_panel_menu" } },
      { url_drilldown = { url = "https://b", label = "off", trigger = "on_open_panel_menu", encode_url = false, open_in_new_tab = false } },
    ]`
	star := minimal + `
    slo_instance_id = "*"`

	for _, strategy := range []string{"computed", "preserve"} {
		t.Run(strategy, func(t *testing.T) {
			addr := "spike_" + strategy + ".t"
			cfg := func(body string) string {
				return fmt.Sprintf("resource %q \"t\" {\n  id = \"codegen-spike-acc-%s\"\n  config = {\n    %s\n  }\n}\n", "spike_"+strategy, strategy, body)
			}
			importStep := tfresource.TestStep{
				ResourceName: addr, ImportState: true, ImportStateVerify: true, ImportStateId: "codegen-spike-acc-" + strategy,
			}
			instance := tfresource.TestCheckResourceAttr(addr, "config.slo_instance_id", "*")
			encode := tfresource.TestCheckResourceAttr(addr, "config.drilldowns.0.url_drilldown.encode_url", "true")
			if strategy == "preserve" {
				instance = tfresource.TestCheckNoResourceAttr(addr, "config.slo_instance_id")
				encode = tfresource.TestCheckNoResourceAttr(addr, "config.drilldowns.0.url_drilldown.encode_url")
			}
			// The repository's config-directory rule is for the elasticstack provider's tests; this is the
			// spike's throwaway provider, and its configurations are built per strategy.
			//nolint:acctestconfigdirlint
			tfresource.Test(t, tfresource.TestCase{
				ProtoV6ProviderFactories: factories,
				Steps: []tfresource.TestStep{
					{Config: cfg(minimal), Check: instance},
					importStep,
					{Config: cfg(full), Check: tfresource.ComposeTestCheckFunc(
						encode,
						tfresource.TestCheckResourceAttr(addr, "config.drilldowns.1.url_drilldown.encode_url", "false"),
					)},
					importStep,
					{Config: cfg(star), Check: tfresource.TestCheckResourceAttr(addr, "config.slo_instance_id", "*")},
					{Config: cfg(minimal), Check: instance},
				},
			})
		})
	}
}
