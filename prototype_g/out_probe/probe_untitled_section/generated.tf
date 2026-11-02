# __generated__ by Terraform
# Please review these resources and move them into your main configuration files.

# __generated__ by Terraform from "default/probe_untitled_section"
resource "elasticstack_kibana_dashboard" "this" {
  access_control = null
  dashboard_id   = "probe_untitled_section"
  description    = null
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
        h = null
        w = null
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
      type                             = ""
      vis_config                       = null
    },
  ]
  pinned_panels    = null
  query            = null
  refresh_interval = null
  sections         = null
  space_id         = "default"
  tags             = null
  time_range       = null
  timeouts         = null
  title            = "probe empty-title section"
}
