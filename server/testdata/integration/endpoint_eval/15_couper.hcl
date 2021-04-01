server "bodies" {
  endpoint "/req" {
    response {
      status = 200
      json_body = req
    }
  }
}
