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

package esqlcontrol

import (
	"context"

	"github.com/elastic/terraform-provider-elasticstack/generated/kbapi"
	"github.com/elastic/terraform-provider-elasticstack/internal/kibana/dashboard/models"
	"github.com/elastic/terraform-provider-elasticstack/internal/kibana/dashboard/panel/iface"
	"github.com/elastic/terraform-provider-elasticstack/internal/kibana/dashboard/panelkit"
	"github.com/elastic/terraform-provider-elasticstack/internal/utils/typeutils"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const panelConfigAttrsKeyPrefix = panelType + "_config"

type Handler struct {
	panelkit.NoopHandlerBase
}

func (Handler) PanelType() string                 { return panelType }
func (Handler) SchemaAttribute() schema.Attribute { return SchemaAttribute() }

func (Handler) PinnedHandler() iface.PinnedHandler { return newPinnedHandler() }

func (Handler) AlignStateFromPlan(_ context.Context, plan, state *models.PanelModel) {
	AlignEsqlPanels(plan, state)
}

func (Handler) FromAPI(ctx context.Context, pm, prior *models.PanelModel, item kbapi.DashboardPanelItem) diag.Diagnostics {
	return panelkit.SimpleFromAPI(ctx, pm, prior,
		item.AsKibanaHTTPAPIsKbnDashboardPanelTypeEsqlControl,
		func(p kbapi.KibanaHTTPAPIsKbnDashboardPanelTypeEsqlControl) (kbapi.KibanaHTTPAPIsKbnDashboardPanelGrid, *string) {
			return p.Grid, p.Id
		},
		func(pm *models.PanelModel, prior *models.PanelModel, p kbapi.KibanaHTTPAPIsKbnDashboardPanelTypeEsqlControl) diag.Diagnostics {
			PopulateFromAPI(pm, prior, p.Config)
			return nil
		},
	)
}

func (Handler) ToAPI(pm models.PanelModel, dashboard *models.DashboardModel) (kbapi.DashboardPanelItem, diag.Diagnostics) {
	_ = dashboard
	grid := panelkit.GridToAPI(pm.Grid)
	id := typeutils.ValueStringPointer(pm.ID)
	panel := kbapi.KibanaHTTPAPIsKbnDashboardPanelTypeEsqlControl{
		Grid: grid,
		Id:   id,
		Type: kbapi.KibanaHTTPAPIsKbnDashboardPanelTypeEsqlControlTypeEsqlControl,
	}
	diags := BuildConfig(pm, &panel)
	var panelItem kbapi.DashboardPanelItem
	if diags.HasError() {
		return panelItem, diags
	}
	if err := panelItem.FromKibanaHTTPAPIsKbnDashboardPanelTypeEsqlControl(panel); err != nil {
		diags.AddError("Failed to create esql control panel", err.Error())
		return panelItem, diags
	}
	return panelItem, diags
}

// ValidatePanelConfig checks required shallow leaves for typed esql configs (covers contracttest flattened attrs).
func (Handler) ValidatePanelConfig(_ context.Context, attrs map[string]attr.Value, attrPath path.Path) diag.Diagnostics {
	var out diag.Diagnostics
	flat, nested, shaped := panelkit.ResolvePanelAttrsShape(attrs, panelConfigAttrsKeyPrefix, "variable_name", "esql_query")
	if !shaped {
		return out
	}

	attrRead := attrs
	cfgPath := attrPath
	if !flat {
		cfgPath = attrPath.AtName(panelConfigAttrsKeyPrefix)
		attrRead = nested.Attributes()
	}

	requiredString := func(name, label string) {
		v := attrRead[name]
		deferVal, missing := panelkit.StringAttrDeferOrMissing(v)
		if deferVal {
			return
		}
		if missing {
			out.AddAttributeError(cfgPath.AtName(name), "Invalid ES|QL control configuration", "`"+label+"` is required.")
		}
	}

	requiredString("variable_name", "variable_name")
	requiredString("variable_type", "variable_type")
	requiredString("control_type", "control_type")

	out.Append(validateQueryForControlType(attrRead, cfgPath)...)

	sel := attrRead["selected_options"]
	switch {
	case sel == nil:
		out.AddAttributeError(cfgPath.AtName("selected_options"), "Invalid ES|QL control configuration", "`selected_options` is required.")
	case sel.IsUnknown():
		// Defer until list elements are known (cross-resource graphs).
	default:
		if sel.IsNull() {
			out.AddAttributeError(cfgPath.AtName("selected_options"), "Invalid ES|QL control configuration", "`selected_options` is required.")
		}
	}
	return out
}

// validateQueryForControlType requires esql_query when control_type is VALUES_FROM_QUERY. Only that
// branch of Kibana's union has a query; a STATIC_VALUES control takes its options from available_options.
func validateQueryForControlType(attrs map[string]attr.Value, cfgPath path.Path) diag.Diagnostics {
	var out diag.Diagnostics
	ct, ok := attrs["control_type"].(types.String)
	if !ok || ct.IsUnknown() || ct.ValueString() != string(kbapi.VALUESFROMQUERY) {
		return out
	}
	if deferVal, missing := panelkit.StringAttrDeferOrMissing(attrs["esql_query"]); !deferVal && missing {
		out.AddAttributeError(cfgPath.AtName("esql_query"), "Invalid ES|QL control configuration",
			"`esql_query` is required when `control_type` is `VALUES_FROM_QUERY`.")
	}
	return out
}

// QueryForControlTypeValidator applies the esql_query rule of ValidatePanelConfig to an
// esql_control_config object outside panels[] (pinned_panels[]).
func QueryForControlTypeValidator() validator.Object {
	return queryForControlTypeValidator{}
}

type queryForControlTypeValidator struct{}

func (queryForControlTypeValidator) Description(_ context.Context) string {
	return "Requires `esql_query` when `control_type` is `VALUES_FROM_QUERY`."
}

func (v queryForControlTypeValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (queryForControlTypeValidator) ValidateObject(_ context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	resp.Diagnostics.Append(validateQueryForControlType(req.ConfigValue.Attributes(), req.Path)...)
}
