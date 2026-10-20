# The url attribute builds the upstream URL from request data. A traversal
# sequence in that data must not reach the origin.
server "gateway" {
  endpoint "/u" {
    proxy {
      url = "${env.COUPER_TEST_RAW_BACKEND_ADDR}/files/${request.query.f[0]}"
    }
  }

  endpoint "/r" {
    request "r" {
      url = "${env.COUPER_TEST_RAW_BACKEND_ADDR}/files/${request.query.f[0]}"
    }

    response {
      body = backend_responses.r.body
    }
  }
}
