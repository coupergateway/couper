server "accepting-forwarded-url" {
  endpoint "/path" {
    response {
      json_body = request
    }
  }
}
settings {
  xfh = true
  accept_forwarded_url = [ "proto", "port" ]
}
