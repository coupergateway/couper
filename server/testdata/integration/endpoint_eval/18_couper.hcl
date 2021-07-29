server "backend_requests" {
  endpoint "/body" {
    request {
      method = "POST"
      url    = "{{.rsOrigin}}/resource?foo=bar"
      body   = request.body
    }
    response {
      json_body = backend_requests.default
    }
  }
  endpoint "/json_body" {
    request {
      method    = "POST"
      url       = "{{.rsOrigin}}/resource?foo=bar"
      json_body = request.json_body
    }
    response {
      json_body = backend_requests.default
    }
  }
  endpoint "/form_body" {
    request {
      method    = "POST"
      url       = "{{.rsOrigin}}/resource?foo=bar"
      form_body = request.form_body
    }
    response {
      json_body = backend_requests.default
    }
  }
}
