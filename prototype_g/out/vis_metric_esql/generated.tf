# __generated__ by Terraform
# Please review these resources and move them into your main configuration files.

# __generated__ by Terraform from "default/vis_metric_esql"
resource "elasticstack_kibana_dashboard" "this" {
  access_control = null
  dashboard_id   = "vis_metric_esql"
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
        h = 6
        w = 12
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
        by_reference = null
        by_value = {
          datatable_config     = null
          gauge_config         = null
          heatmap_config       = null
          legacy_metric_config = null
          metric_chart_config = {
            breakdown_by_json = null
            data_source_json = jsonencode({
              query = "FROM kibana_sample_data_logs | WHERE timestamp >= ?_tstart AND timestamp < ?_tend | STATS avg_bytes = AVG(bytes)"
              type  = "esql"
            })
            description = "ESQL Metric with trend background chart"
            drilldowns  = null
            filters     = null
            hide_border = null
            hide_title  = null
            metrics = [
              {
                config_json = jsonencode({
                  background_chart = {
                    type = "trend"
                  }
                  column = "avg_bytes"
                  type   = "primary"
                })
              },
            ]
            query           = null
            references_json = null
            time_range      = null
            title           = "Metric - ESQL with trendline"
          }
          mosaic_config     = null
          pie_chart_config  = null
          region_map_config = null
          tagcloud_config   = null
          treemap_config    = null
          waffle_config     = null
          xy_chart_config   = null
        }
      }
    },
    {
      aiops_change_point_chart_config = null
      aiops_log_rate_analysis_config  = null
      aiops_pattern_analysis_config   = null
      apm_service_map_config          = null
      discover_session_config         = null
      esql_control_config             = null
      field_stats_table_config        = null
      grid = {
        h = 6
        w = 12
        x = 12
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
        by_reference = null
        by_value = {
          datatable_config     = null
          gauge_config         = null
          heatmap_config       = null
          legacy_metric_config = null
          metric_chart_config = {
            breakdown_by_json = jsonencode({
              column = "extension.keyword"
            })
            data_source_json = jsonencode({
              query = "FROM logs | STATS ..."
              type  = "esql"
            })
            description = "ESQL Metric with background chart and breakdown"
            drilldowns  = null
            filters     = null
            hide_border = null
            hide_title  = null
            metrics = [
              {
                config_json = jsonencode({
                  background_chart = {
                    max_value = {
                      column = "bytes"
                    }
                    type = "bar"
                  }
                  color = {
                    range = "absolute"
                    steps = [{
                      color = "red"
                      lt    = 0
                    }]
                    type = "dynamic"
                  }
                  column = "count"
                  type   = "primary"
                })
              },
              {
                config_json = jsonencode({
                  column = "bytes"
                  compare = {
                    baseline = 100
                    palette  = "status"
                    to       = "baseline"
                    value    = false
                  }
                  type = "secondary"
                })
              },
            ]
            query           = null
            references_json = null
            time_range      = null
            title           = "Metric - ESQL Complex case"
          }
          mosaic_config     = null
          pie_chart_config  = null
          region_map_config = null
          tagcloud_config   = null
          treemap_config    = null
          waffle_config     = null
          xy_chart_config   = null
        }
      }
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
  title            = "Metric ES|QL"
}
