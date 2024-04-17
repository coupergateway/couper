server "bodies" {
  endpoint "/req" {
    response {
      status = 200
      headers = {
        x-json: request.json_body
      }
      json_body = request
    }
  }
  endpoint "/req2" {
    response {
      status = 200
      json_body = request
    }
  }
}
