server "dynamic-request" {
  endpoint "/**" {
    request {
      url    = "${env.COUPER_TEST_BACKEND_ADDR}/anything?q=${request.headers.query}"
      body   = request.headers.body
      method = request.query.method[0]

      headers = {
        Test = request.headers.test
      }

      backend {
        origin = env.COUPER_TEST_BACKEND_ADDR
      }
    }
  }
}
