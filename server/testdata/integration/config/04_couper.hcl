server "backends" {
  endpoint "/" {
    proxy {
      backend = "ref"
      set_request_headers = {
        proxy = "a"
      }
    }

    request "sidekick" {
      backend = "ref"
      headers = {
        request = "b"
      }
    }

    response {
      headers = {
        proxy = backend_responses.default.json_body.Headers.Proxy
        request = backend_responses.sidekick.json_body.Headers.Request
      }
    }
  }
}

definitions {
  backend "ref" {
    origin = env.COUPER_TEST_BACKEND_ADDR
    path = "/anything"
  }
}

settings {
  no_proxy_from_env = true
}
