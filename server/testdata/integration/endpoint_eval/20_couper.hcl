server "cty.NilVal" {
  endpoint "/1stchild" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
    }
    response {
      headers = {
        X-Value = env.COUPER_TEST_BACKEND_ADDR
        Z-Value = "y"
      }
    }
  }

  endpoint "/2ndchild/no" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
    }
    response {
      headers = {
        X-Value = env.COUPER_TEST_BACKEND_ADDR.not_there
        Z-Value = "y"
      }
    }
  }

  endpoint "/child-chain/no" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
    }
    response {
      headers = {
        X-Value = env.COUPER_TEST_BACKEND_ADDR.one.two
        Z-Value = "y"
      }
    }
  }

  endpoint "/list-idx" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
    }
    response {
      headers = {
        X-Value = backend_responses.default.json_body.Json.list[1]
        Z-Value = "y"
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
        Z-Value = "y"
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
        Z-Value = "y"
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
        Z-Value = "y"
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
        Z-Value = "y"
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
        Z-Value = "y"
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
        Z-Value = "y"
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
        Z-Value = "y"
      }
    }
  }

  endpoint "/conditional/false" {
    response {
      headers = {
        X-Value = request.form_body.state != null ? request.form_body.state : "x"
        Z-Value = "y"
      }
    }
  }

  endpoint "/conditional/true" {
    response {
      headers = {
        X-Value = true ? request.form_body.state : "x"
        Z-Value = "y"
      }
    }
  }

  endpoint "/conditional/null" {
    response {
      headers = {
        X-Value = null ? request.form_body.state : "x"
        Z-Value = "y"
      }
    }
  }

  endpoint "/conditional/nested" {
    response {
      headers = {
        X-Value = request.form_body.state != (2 + 2 == 4 ? "value" : null) ? "x" : request.form_body.state
        Z-Value = "y"
      }
    }
  }

  endpoint "/conditional/nested/true" {
    response {
      headers = {
        X-Value = request.form_body.state == null ? 1 + 1 == 2 ? "x" : null : request.form_body.state
        Z-Value = "y"
      }
    }
  }

  endpoint "/conditional/nested/false" {
    response {
      headers = {
        X-Value = request.form_body.state != null ? null : 1+1 == 3 ? null : "x"
        Z-Value = "y"
      }
    }
  }
}

settings {
  no_proxy_from_env = true
}
