server "trailers" {
  endpoint "/**" {
    proxy {
      backend {
        origin                         = "{{.origin}}"
        http2                          = true
        disable_certificate_validation = true
      }
    }
  }
}
