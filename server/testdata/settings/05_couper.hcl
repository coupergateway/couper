server "accepting-forwarded-url" {
  endpoint "/path" {
    response {
      json_body = request
    }
  }
}
settings {
  accept_forwarded_url = [ "proto", "host", "port" ]
}
