server "dynamic-request" {
  endpoint "/**" {
    request {
      url    = "${env.COUPER_TEST_BACKEND_ADDR}/anything?q=${req.headers.query}"
      body   = req.headers.body
      method = req.query.method[0]

      headers = {
        Test = req.headers.test
      }

      backend {
        origin = env.COUPER_TEST_BACKEND_ADDR
      }
    }
  }
}
