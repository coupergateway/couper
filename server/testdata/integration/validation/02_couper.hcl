server {
  endpoint "/buffer" {
    proxy {
      backend {
        origin = request.headers.origin
        openapi {
          file = "02_schema.yaml"
        }
        timeout = "5s"
      }
    }
  }
}
