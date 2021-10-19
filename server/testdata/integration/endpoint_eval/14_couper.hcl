server "bodies" {
  endpoint "/request/body" {
    request {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
      body = "foo"

      backend {
        origin = env.COUPER_TEST_BACKEND_ADDR
      }
    }
  }

  endpoint "/request/body/ct" {
    request {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
      headers = {
        content-type = "application/foo"
      }
      body = "foo"

      backend {
        origin = env.COUPER_TEST_BACKEND_ADDR
      }
    }
  }

  endpoint "/request/json_body/null" {
    request {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
      json_body = null

      backend {
        origin = env.COUPER_TEST_BACKEND_ADDR
      }
    }
  }

  endpoint "/request/json_body/boolean" {
    request {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
      json_body = true

      backend {
        origin = env.COUPER_TEST_BACKEND_ADDR
      }
    }
  }

  endpoint "/request/json_body/boolean/ct" {
    request {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
      headers = {
        content-type = "application/foo+json"
      }
      json_body = true

      backend {
        origin = env.COUPER_TEST_BACKEND_ADDR
      }
    }
  }

  endpoint "/request/json_body/number" {
    request {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
      json_body = 1.2

      backend {
        origin = env.COUPER_TEST_BACKEND_ADDR
      }
    }
  }

  endpoint "/request/json_body/string" {
    request {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
      json_body = "foo"

      backend {
        origin = env.COUPER_TEST_BACKEND_ADDR
      }
    }
  }

  endpoint "/request/json_body/object" {
    request {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
      json_body = {"foo": "bar"}

      backend {
        origin = env.COUPER_TEST_BACKEND_ADDR
      }
    }
  }

  endpoint "/request/json_body/array" {
    request {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
      json_body = [0,1,2]

      backend {
        origin = env.COUPER_TEST_BACKEND_ADDR
      }
    }
  }

  endpoint "/request/json_body/dyn" {
    request {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
      json_body = request.json_body

      backend {
        origin = env.COUPER_TEST_BACKEND_ADDR
      }
    }
  }

  endpoint "/request/form_body" {
    request {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
      form_body = {
        foo = "ab c"
		bar = ",:/"
      }

      backend {
        origin = env.COUPER_TEST_BACKEND_ADDR
      }
    }
  }

  endpoint "/request/form_body/ct" {
    request {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
      headers = {
        content-type = "application/my-form-urlencoded"
      }
      form_body = {
        foo = "ab c"
		bar = ",:/"
      }

      backend {
        origin = env.COUPER_TEST_BACKEND_ADDR
      }
    }
  }

  endpoint "/request/form_body/dyn" {
    request {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
      form_body = request.form_body

      backend {
        origin = env.COUPER_TEST_BACKEND_ADDR
      }
    }
  }

  endpoint "/response/body" {
    response {
      body = "foo"
    }
  }

  endpoint "/response/body/ct" {
    response {
      headers = {
        content-type = "application/foo"
      }
      body = "foo"
    }
  }

  endpoint "/response/json_body/null" {
    response {
      json_body = null
    }
  }

  endpoint "/response/json_body/boolean" {
    response {
      json_body = true
    }
  }

  endpoint "/response/json_body/boolean/ct" {
    response {
      headers = {
        content-type = "application/foo+json"
      }
      json_body = true
    }
  }

  endpoint "/response/json_body/number" {
    response {
      json_body = 1.2
    }
  }

  endpoint "/response/json_body/string" {
    response {
      json_body = "foo"
    }
  }

  endpoint "/response/json_body/object" {
    response {
      json_body = {"foo": "bar"}
    }
  }

  endpoint "/response/json_body/array" {
    response {
      json_body = [0,1,2]
    }
  }
}
