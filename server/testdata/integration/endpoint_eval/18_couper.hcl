server "backend_requests" {
  endpoint "/**" {
    request {
      method = "POST"
      url    = "{{.rsOrigin}}/resource?foo=bar"
    }
    response {
      json_body = backend_requests.default
    }
  }
}
