server "request" {
  endpoint "/**" {
    response {
      headers = {
        x-json = request.json_body
      }
      json_body = request
    }
  }
}
