# __generated__ by Terraform
# Please review these resources and move them into your main configuration files.

# __generated__ by Terraform from "default/vis_by_reference"
resource "elasticstack_kibana_dashboard" "this" {
  access_control = null
  dashboard_id   = "vis_by_reference"
  description    = "vis vis_config.by_reference full acceptance"
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
  panels = [
    {
      aiops_change_point_chart_config = null
      aiops_log_rate_analysis_config  = null
      aiops_pattern_analysis_config   = null
      apm_service_map_config          = null
      discover_session_config         = null
      esql_control_config             = null
      field_stats_table_config        = null
      grid = {
        h = 10
        w = 24
        x = 0
        y = 0
      }
      image_config                     = null
      links_config                     = null
      markdown_config                  = null
      ml_anomaly_charts_config         = null
      ml_anomaly_swimlane_config       = null
      ml_single_metric_viewer_config   = null
      options_list_control_config      = null
      range_slider_control_config      = null
      slo_alerts_config                = null
      slo_burn_rate_config             = null
      slo_error_budget_config          = null
      slo_overview_config              = null
      synthetics_monitors_config       = null
      synthetics_stats_overview_config = null
      time_slider_control_config       = null
      type                             = "vis"
      vis_config = {
        by_reference = {
          description = "By reference desc"
          drilldowns  = null
          hide_border = false
          hide_title  = true
          ref_id      = "lensRef"
          references_json = jsonencode([{
            id   = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
            name = "lensRef"
            type = "lens"
          }])
          time_range = {
            from = "now-7d"
            mode = "relative"
            to   = "now"
          }
          title = "Ref title"
        }
        by_value = null
      }
    },
  ]
  pinned_panels = null
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
  title    = "Vis by reference"
}
