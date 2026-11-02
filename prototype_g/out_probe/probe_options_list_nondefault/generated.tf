# __generated__ by Terraform
# Please review these resources and move them into your main configuration files.

# __generated__ by Terraform from "default/probe_options_list_nondefault"
resource "elasticstack_kibana_dashboard" "this" {
  access_control = null
  dashboard_id   = "probe_options_list_nondefault"
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
  sections         = null
  space_id         = "default"
  tags             = null
  time_range       = null
  timeouts         = null
  title            = "probe non-default control values"
}
