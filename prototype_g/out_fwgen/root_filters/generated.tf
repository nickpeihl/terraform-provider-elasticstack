# __generated__ by Terraform
# Please review these resources and move them into your main configuration files.

# __generated__ by Terraform from "default/root_filters"
resource "elasticstack_kibana_dashboard" "this" {
  access_control = null
  dashboard_id   = "root_filters"
  description    = null
  filters = [
    {
      filter_json = jsonencode({
        condition = {
          field    = "status"
          operator = "is"
          value    = "active"
        }
        type = "condition"
      })
    },
    {
      filter_json = jsonencode({
        condition = {
          field    = "host.name"
          operator = "is"
          value    = "web-01"
        }
        type = "condition"
      })
    },
    {
      filter_json = jsonencode({
        group = {
          conditions = [{
            field    = "status"
            operator = "is"
            value    = "active"
            }, {
            field    = "type"
            operator = "is"
            value    = "user"
          }]
          operator = "and"
        }
        type = "group"
      })
    },
    {
      filter_json = jsonencode({
        dsl = {
          query = {
            script = {
              source = "doc.field.value > 0"
            }
          }
        }
        type = "dsl"
      })
    },
    {
      filter_json = jsonencode({
        disabled = false
        dsl = {
          query = {
            bool = {
              should = [{
                bool = {
                  must = [{
                    exists = {
                      field = "DestLocation"
                    }
                    }, {
                    geo_shape = {
                      DestLocation = {
                        relation = "intersects"
                        shape = {
                          coordinates = [[[-116.98646, 49.95363], [-69.4932, 36.01698], [-89.978, 27.62399], [-114.50929, 36.07558], [-116.98646, 49.95363]]]
                          type        = "Polygon"
                        }
                      }
                      ignore_unmapped = true
                    }
                  }]
                }
                }, {
                bool = {
                  must = [{
                    exists = {
                      field = "geo.coordinates"
                    }
                    }, {
                    geo_shape = {
                      "geo.coordinates" = {
                        relation = "intersects"
                        shape = {
                          coordinates = [[[-116.98646, 49.95363], [-69.4932, 36.01698], [-89.978, 27.62399], [-114.50929, 36.07558], [-116.98646, 49.95363]]]
                          type        = "Polygon"
                        }
                      }
                      ignore_unmapped = true
                    }
                  }]
                }
              }]
            }
          }
        }
        is_multi_index = true
        label          = "intersects shape"
        type           = "spatial"
      })
    },
  ]
  options = {
    auto_apply_filters = null
    hide_panel_borders = null
    hide_panel_titles  = null
    sync_colors        = null
    sync_cursor        = null
    sync_tooltips      = null
    use_margins        = null
  }
  panels           = null
  pinned_panels    = null
  query            = null
  refresh_interval = null
  sections         = null
  space_id         = "default"
  tags             = null
  time_range       = null
  timeouts         = null
  title            = "Root filters"
}
