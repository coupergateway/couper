server "couper" {
  endpoint "/abcdef" {
    proxy = "test"
    response {
      status = 204
    }
  }
  endpoint "/reuse" {
    proxy = "test"
    response {
      status = 204
    }
  }

  endpoint "/default" {
    proxy = "defaultName"
  }

  api {
    endpoint "/api-abcdef" {
      proxy = "test"
      response {
        status = 204
      }
    }
    endpoint "/api-reuse" {
      proxy = "test"
      response {
        status = 204
      }
    }

    endpoint "/api-default" {
      proxy = "defaultName"
    }
  }
}

definitions {
  proxy "defaultName" {
    url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
  }

  proxy "test" {
    name = "abcdef"
    url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
  }
}
