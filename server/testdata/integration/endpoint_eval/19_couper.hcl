server "request" {
  endpoint "/**" {
    response {
      json_body = request
    }
  }
}
