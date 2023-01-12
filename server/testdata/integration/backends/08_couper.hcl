server {
  endpoint "/**" {
    proxy {
      backend = "be"
    }
  }
}

definitions {
  backend "be" {
    origin = "${env.COUPER_TEST_BACKEND_ADDR}"
    set_response_headers = {
      x-healthy-1 = backend.health.healthy ? "true" : "false"
      x-healthy-2 = backends.be.health.healthy ? "true" : "false"
      x-rp-1 = backend_request.path
      x-rp-2 = backend_requests.default.path
      x-rs-1 = backend_response.status
      x-rs-2 = backend_responses.default.status
    }
    custom_log_fields = {
      healthy_1 = backend.health.healthy
      healthy_2 = backends.be.health.healthy
      rp_1 = backend_request.path
      rp_2 = backend_requests.default.path
      rs_1 = backend_response.status
      rs_2 = backend_responses.default.status
    }
  }
}
