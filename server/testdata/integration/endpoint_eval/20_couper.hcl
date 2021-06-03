server "cty.NilVal" {
  endpoint "/list-idx" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
    }
    response {
      headers = {
        X-Value = backend_responses.default.json_body.Json.list[1]
      }
    }
  }

  endpoint "/list-idx-splat" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
    }
    response {
      headers = {
        X-Value = backend_responses.default.json_body.Json.list[*]
      }
    }
  }

  endpoint "/list-idx/no" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
    }
    response {
      headers = {
        X-Value = backend_responses.default.json_body.Json.list[21]
      }
    }
  }

  endpoint "/list-idx-chain/no" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
    }
    response {
      headers = {
        X-Value = backend_responses.default.json_body.Json.list[21][12]
      }
    }
  }

  endpoint "/list-idx-key-chain/no" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
    }
    response {
      headers = {
        X-Value = backend_responses.default.json_body.Json.list[21].obj[1]
      }
    }
  }

  endpoint "/root/no" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
    }
    response {
      headers = {
        X-Value = no-root
      }
    }
  }

  endpoint "/tpl" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
    }
    response {
      headers = {
        X-Value = "${env.COUPER_TEST_BACKEND_ADDR}mytext"
      }
    }
  }

  endpoint "/for" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
    }
    response {
      headers = {
        X-Value = [for i, v in backend_responses.default.json_body.Json.list: v if i < 1]
      }
    }
  }
}

settings {
  no_proxy_from_env = true
}
