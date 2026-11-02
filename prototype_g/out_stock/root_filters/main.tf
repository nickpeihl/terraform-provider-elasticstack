terraform {
  required_providers {
    elasticstack = { source = "elastic/elasticstack" }
  }
}
provider "elasticstack" {
  kibana {
    endpoints = ["http://127.0.0.1:5699"]
    username  = "elastic"
    password  = "changeme"
  }
}
import {
  to = elasticstack_kibana_dashboard.this
  id = "default/root_filters"
}
