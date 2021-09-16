server "backend_probes" {
  endpoint "/" {
    proxy {
      backend = "invalid"
      set_response_headers = {
        state-1 = backend_probes.valid.state
        state-2 = backend_probes.invalid.state
      }
    }
  }
  endpoint "/foo" {
    proxy {
      backend = "invalid"
      set_response_headers = {
        state-1 = backend_probes.valid.state
        state-2 = backend_probes.invalid.state
      }
    }
  }
}
definitions {
  backend "valid" {
    name = "valid"
    origin = "https://httpbin.org"
    path = "/anything"
  }
  backend "invalid" {
    name = "invalid"
    origin = "http://google"
  }
}