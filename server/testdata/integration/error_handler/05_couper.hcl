server {
  api {
    endpoint "/api-backend" {
      proxy {
        backend {
          origin  = "${env.COUPER_TEST_BACKEND_ADDR}"
          timeout = "1ns"
        }
      }
    }

    error_handler "backend" {
      response {
        status = 405
        json_body = {
          "api": "backend"
        }
      }
    }
  }

  api {
    endpoint "/api-backend-timeout" {
      proxy {
        backend {
          origin  = "${env.COUPER_TEST_BACKEND_ADDR}"
          timeout = "1ns"
        }
      }
    }

    error_handler "backend_timeout" {
      response {
        status = 405
        json_body = {
          "api": "backend-timeout"
        }
      }
    }
  }

  api {
    endpoint "/api-backend-validation" {
      proxy {
        backend {
          origin = "${env.COUPER_TEST_BACKEND_ADDR}"

          openapi {
            file = "01_schema.yaml"
          }
        }
      }
    }

    error_handler "backend_openapi_validation" {
      response {
        status = 405
        json_body = {
          "api": "backend-validation"
        }
      }
    }
  }

  api {
    endpoint "/api-anything" {
      proxy {
        backend {
          origin = "${env.COUPER_TEST_BACKEND_ADDR}"

          openapi {
            file = "02_schema.yaml"
          }
        }
      }
    }

    error_handler "backend_openapi_validation" {
      response {
        status = 405
        json_body = {
          "api": "backend-backend-validation"
        }
      }
    }
  }

  endpoint "/backend" {
    proxy {
      backend {
        origin  = "${env.COUPER_TEST_BACKEND_ADDR}"
        timeout = "1ns"
      }
    }

    error_handler "backend" {
      response {
        status = 405
        json_body = {
          "endpoint": "backend"
        }
      }
    }
  }

  endpoint "/backend-timeout" {
    proxy {
      backend {
        origin  = "${env.COUPER_TEST_BACKEND_ADDR}"
        timeout = "1ns"
      }
    }

    error_handler "backend_timeout" {
      response {
        status = 405
        json_body = {
          "endpoint": "backend-timeout"
        }
      }
    }
  }

  endpoint "/backend-validation" {
    proxy {
      backend {
        origin = "${env.COUPER_TEST_BACKEND_ADDR}"

        openapi {
          file = "01_schema.yaml"
        }
      }
    }

    error_handler "backend_openapi_validation" {
      response {
        status = 405
        json_body = {
          "endpoint": "backend-validation"
        }
      }
    }
  }

  endpoint "/anything" {
    proxy {
      backend {
        origin = "${env.COUPER_TEST_BACKEND_ADDR}"

        openapi {
          file = "02_schema.yaml"
        }
      }
    }

    error_handler "backend_openapi_validation" {
      response {
        status = 405
        json_body = {
          "endpoint": "backend-backend-validation"
        }
      }
    }
  }
}
