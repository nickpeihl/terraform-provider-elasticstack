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

package esqlcontrol_test

import (
	"regexp"
	"testing"

	"github.com/elastic/terraform-provider-elasticstack/internal/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// esqlControlUnitConfig is a dashboard with one ES|QL control in the shape Kibana's Export → Terraform
// flyout writes: esql_control_config is an object, and esql_query is present only when the control's
// options come from a query. pinned places the control in the control bar instead of panels.
func esqlControlUnitConfig(pinned bool, esqlControlConfig string) string {
	panel := `panels = [{
    type = "esql_control"
    grid = { x = 0, y = 0, w = 24, h = 6 }
    esql_control_config = ` + esqlControlConfig + `
  }]`
	if pinned {
		panel = `pinned_panels = [{
    type = "esql_control"
    esql_control_config = ` + esqlControlConfig + `
  }]`
	}
	return `
provider "elasticstack" {
  elasticsearch {}
  kibana {}
}

resource "elasticstack_kibana_dashboard" "test" {
  title = "esql-control-validate"
  time_range = {
    from = "now-15m"
    to   = "now"
  }
  ` + panel + `
}
`
}

const (
	staticValuesWithoutQuery = `{
      selected_options  = ["a"]
      variable_name     = "host"
      variable_type     = "values"
      control_type      = "STATIC_VALUES"
      available_options = ["a", "b"]
    }`
	valuesFromQueryWithoutQuery = `{
      selected_options = ["a"]
      variable_name    = "host"
      variable_type    = "values"
      control_type     = "VALUES_FROM_QUERY"
    }`
)

func TestUnitResourceDashboardESQLControlQueryByControlType(t *testing.T) {
	tests := []struct {
		name        string
		pinned      bool
		config      string
		expectError *regexp.Regexp
	}{
		{name: "panel STATIC_VALUES without esql_query is valid", config: staticValuesWithoutQuery},
		{name: "pinned STATIC_VALUES without esql_query is valid", pinned: true, config: staticValuesWithoutQuery},
		{
			name:        "panel VALUES_FROM_QUERY requires esql_query",
			config:      valuesFromQueryWithoutQuery,
			expectError: regexp.MustCompile("`esql_query` is required when `control_type` is `VALUES_FROM_QUERY`"),
		},
		{
			name:        "pinned VALUES_FROM_QUERY requires esql_query",
			pinned:      true,
			config:      valuesFromQueryWithoutQuery,
			expectError: regexp.MustCompile("`esql_query` is required when `control_type` is `VALUES_FROM_QUERY`"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := resource.TestStep{Config: esqlControlUnitConfig(tt.pinned, tt.config)}
			if tt.expectError != nil {
				step.ExpectError = tt.expectError
			} else {
				step.PlanOnly = true
				step.ExpectNonEmptyPlan = true
			}
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: acctest.Providers,
				Steps:                    []resource.TestStep{step},
			})
		})
	}
}
