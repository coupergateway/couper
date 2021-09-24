server "backend_probes" {
  endpoint "/valid" {
    proxy {
      backend = "valid"
    }
    set_response_headers = {
      state = backend_probes.valid.state
    }
  }
  endpoint "/foo" {
    proxy {
      backend = "invalid"
    }
  }
  endpoint "/invalid" {
    response {
      headers = {
        state = backend_probes.invalid.state
      }
    }
  }
  endpoint "/vali" {
    proxy {
      backend = "valid"
    }
    set_response_headers = {
      state = backend_probes.vali.state
      state-2 = backend_probes.valid
    }
  }
}
definitions {
  backend "valid" {
    name = "valid"
    origin = env.COUPER_TEST_BACKEND_ADDR
    health_check {}
  }
  backend "invalid" {
    name = "invalid"
    origin = "http://1.2.3.4"
    health_check {}
  }
}
