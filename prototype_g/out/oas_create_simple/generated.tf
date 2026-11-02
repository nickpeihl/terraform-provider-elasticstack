# __generated__ by Terraform
# Please review these resources and move them into your main configuration files.

# __generated__ by Terraform
resource "elasticstack_kibana_dashboard" "this" {
  access_control = null
  dashboard_id   = "oas_create_simple"
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
        h = 15
        w = 24
        x = 0
        y = 0
      }
      image_config = null
      links_config = null
      markdown_config = {
        by_reference = null
        by_value = {
          content     = "## Web logs overview\n\n&nbsp;\n\nCreated with the [Dashboards API](https://www.elastic.co/docs/api/doc/kibana) using the Kibana sample web logs dataset (`kibana_sample_data_logs`). Contains:\n- This markdown panel\n- 2 metrics, showing request count and average response size\n- A line chart based on an ES|QL query\n\n&nbsp;\n\n&nbsp;\n\n[Learn more about dashboards](https://www.elastic.co/docs/explore-analyze/dashboards)"
          description = null
          hide_border = null
          hide_title  = null
          settings = {
            open_links_in_new_tab = null
          }
          title = null
        }
      }
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
      type                             = "markdown"
      vis_config                       = null
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
        h = 5
        w = 12
        x = 24
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
              index_pattern = "kibana_sample_data_logs"
              time_field    = "timestamp"
              type          = "data_view_spec"
            })
            description = null
            drilldowns  = null
            filters     = null
            hide_border = null
            hide_title  = null
            metrics = [
              {
                config_json = jsonencode({
                  operation = "count"
                  type      = "primary"
                })
              },
            ]
            query = {
              expression = ""
              language   = "kql"
            }
            references_json = null
            time_range      = null
            title           = null
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
        h = 5
        w = 12
        x = 36
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
              index_pattern = "kibana_sample_data_logs"
              time_field    = "timestamp"
              type          = "data_view_spec"
            })
            description = null
            drilldowns  = null
            filters     = null
            hide_border = null
            hide_title  = null
            metrics = [
              {
                config_json = jsonencode({
                  field     = "bytes"
                  operation = "average"
                  type      = "primary"
                })
              },
            ]
            query = {
              expression = ""
              language   = "kql"
            }
            references_json = null
            time_range      = null
            title           = null
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
        h = 10
        w = 24
        x = 24
        y = 5
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
          metric_chart_config  = null
          mosaic_config        = null
          pie_chart_config     = null
          region_map_config    = null
          tagcloud_config      = null
          treemap_config       = null
          waffle_config        = null
          xy_chart_config = {
            axis = {
              x = {
                scale = null
                title = {
                  value   = null
                  visible = false
                }
              }
              y  = null
              y2 = null
            }
            decorations = {
              line_interpolation       = null
              minimum_bar_height       = null
              point_visibility         = null
              show_current_time_marker = null
              show_end_zones           = null
              show_value_labels        = null
            }
            description = null
            drilldowns  = null
            filters     = null
            fitting = {
              dotted    = null
              end_value = null
              type      = null
            }
            hide_border = null
            hide_title  = null
            layers = [
              {
                data_layer = {
                  breakdown_by_json = null
                  data_source_json = jsonencode({
                    query = "FROM kibana_sample_data_logs | STATS count = COUNT() BY BUCKET(@timestamp, 75, ?_tstart, ?_tend)"
                    type  = "esql"
                  })
                  x_json = jsonencode({
                    column = "BUCKET(@timestamp, 75, ?_tstart, ?_tend)"
                  })
                  y = [
                    {
                      config_json = jsonencode({
                        column = "count"
                      })
                    },
                  ]
                }
                reference_line_layer = null
                type                 = "line"
              },
            ]
            legend = null
            query = {
              expression = ""
              language   = "kql"
            }
            references_json = null
            time_range      = null
            title           = "Requests over time"
          }
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
  title            = "Web logs overview"
}
