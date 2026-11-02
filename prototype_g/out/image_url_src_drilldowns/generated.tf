# __generated__ by Terraform
# Please review these resources and move them into your main configuration files.

# __generated__ by Terraform from "default/image_url_src_drilldowns"
resource "elasticstack_kibana_dashboard" "this" {
  access_control = null
  dashboard_id   = "image_url_src_drilldowns"
  description    = "Dashboard with typed image panel (URL src, mixed drilldowns)"
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
      image_config = {
        alt_text         = "Logo"
        background_color = "#111111"
        description      = "Image panel description"
        drilldowns = [
          {
            dashboard_drilldown = {
              dashboard_id    = "spike0-target-dashboard"
              label           = "Open dashboard"
              open_in_new_tab = null
              trigger         = "on_click_image"
              use_filters     = null
              use_time_range  = null
            }
            url_drilldown = null
          },
          {
            dashboard_drilldown = null
            url_drilldown = {
              encode_url      = null
              label           = "External link"
              open_in_new_tab = null
              trigger         = "on_click_image"
              url             = "https://example.com/details/{{context.panel.title}}"
            }
          },
        ]
        hide_border = true
        hide_title  = false
        object_fit  = "cover"
        src = {
          file = null
          url = {
            url = "https://example.com/logo.png"
          }
        }
        title = "Image panel title"
      }
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
      type                             = "image"
      vis_config                       = null
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
  title    = "Image url src"
}
