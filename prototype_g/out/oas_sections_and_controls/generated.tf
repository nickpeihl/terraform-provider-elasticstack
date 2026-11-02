# __generated__ by Terraform
# Please review these resources and move them into your main configuration files.

# __generated__ by Terraform
resource "elasticstack_kibana_dashboard" "this" {
  access_control = null
  dashboard_id   = "oas_sections_and_controls"
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
  panels = null
  pinned_panels = [
    {
      esql_control_config = null
      grow                = true
      options_list_control_config = {
        by_esql = null
        by_field = {
          data_view_id       = "90943e30-9a47-11e8-b64d-95841ca0b247"
          display_settings   = null
          exclude            = null
          exists_selected    = null
          field_name         = "response.keyword"
          ignore_validations = null
          run_past_timeout   = null
          search_technique   = "prefix"
          selected_options   = []
          single_select      = false
          sort               = null
          title              = "Response code"
          use_global_filters = null
        }
      }
      range_slider_control_config = null
      time_slider_control_config  = null
      type                        = "options_list_control"
      width                       = "medium"
    },
  ]
  query            = null
  refresh_interval = null
  sections = [
    {
      collapsed = false
      grid = {
        y = 0
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
            h = 5
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
      ]
      title = "Key metrics"
    },
    {
      collapsed = false
      grid = {
        y = 8
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
                        index_pattern = "kibana_sample_data_logs"
                        time_field    = "timestamp"
                        type          = "data_view_spec"
                      })
                      x_json = jsonencode({
                        field     = "timestamp"
                        operation = "date_histogram"
                      })
                      y = [
                        {
                          config_json = jsonencode({
                            operation = "count"
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
                title           = "Requests over time (ES|QL)"
              }
            }
          }
        },
      ]
      title = "Traffic trends"
    },
  ]
  space_id = "default"
  tags     = null
  time_range = {
    from = "now-7d"
    mode = null
    to   = "now"
  }
  timeouts = null
  title    = "Operations overview"
}
