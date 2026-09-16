server "no-trailers" {
  endpoint "/h2/**" {
    proxy {
      backend {
        origin                         = "{{.origin}}"
        http2                          = true
        disable_certificate_validation = true
      }
    }
  }

  endpoint "/h2eval/**" {
    proxy {
      backend {
        origin                         = "{{.origin}}"
        http2                          = true
        disable_certificate_validation = true
      }
    }

    set_response_headers = {
      x-from-json-body = backend_responses.default.json_body.name
    }
  }

  endpoint "/h1/**" {
    proxy {
      backend {
        origin                         = "{{.origin}}"
        disable_certificate_validation = true
      }
    }
  }
}
