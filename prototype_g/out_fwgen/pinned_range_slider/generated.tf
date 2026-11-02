# __generated__ by Terraform
# Please review these resources and move them into your main configuration files.

# __generated__ by Terraform from "default/pinned_range_slider"
resource "elasticstack_kibana_dashboard" "this" {
  access_control = null
  dashboard_id   = "pinned_range_slider"
  description    = "Dashboard with pinned range slider controls"
  filters        = null
  options = {
    auto_apply_filters = null
    hide_panel_borders = null
    hide_panel_titles  = null
    sync_colors        = null
    sync_cursor        = null
    sync_tooltips      = null
    use_margins        = null
  }
  panels = null
  pinned_panels = [
    {
      esql_control_config         = null
      grow                        = true
      options_list_control_config = null
      range_slider_control_config = {
        by_esql = null
        by_field = {
          data_view_id       = "spike0-data-view"
          field_name         = "source.bytes"
          ignore_validations = true
          step               = 5
          title              = "Bytes Range Updated"
          use_global_filters = false
          value              = ["50", "400"]
        }
      }
      time_slider_control_config = null
      type                       = "range_slider_control"
      width                      = "medium"
    },
    {
      esql_control_config         = null
      grow                        = null
      options_list_control_config = null
      range_slider_control_config = {
        by_esql = {
          esql_query         = "FROM logs-* | STATS min = MIN(bytes), max = MAX(bytes)"
          ignore_validations = false
          step               = 10
          title              = "Bytes Range"
          use_global_filters = true
          value              = null
          values_source      = "esql_query"
        }
        by_field = null
      }
      time_slider_control_config = null
      type                       = "range_slider_control"
      width                      = null
    },
  ]
  query = {
    expression = ""
    language   = "kql"
  }
  refresh_interval = {
    pause = true
    value = 0
  }
  sections = null
  space_id = "default"
  tags     = null
  time_range = {
    from = "now-15m"
    mode = null
    to   = "now"
  }
  timeouts = null
  title    = "Pinned range slider"
}
