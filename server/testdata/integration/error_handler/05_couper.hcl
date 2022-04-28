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

  api {
    endpoint "/c" {
      proxy {
        backend {
          origin  = "${env.COUPER_TEST_BACKEND_ADDR}"

          openapi {
            file = "01_schema.yaml"
          }
        }
      }

      error_handler "backend_openapi_validation" {
        // should win
        response {
          status = 405
          body = "endpoint:backend_openapi_validation"
        }
      }
    }

    error_handler "backend" {
      response {
        status = 400
        body = "api:backend"
      }
    }
  }

  api {
    endpoint "/d" {
      proxy {
        backend {
          origin  = "${env.COUPER_TEST_BACKEND_ADDR}"

          openapi {
            file = "01_schema.yaml"
          }
        }
      }

      error_handler "backend" {
        // should win
        response {
          status = 405
          body = "endpoint:backend"
        }
      }
    }

    error_handler "backend_openapi_validation" {
      response {
        status = 400
        body = "api:backend_openapi_validation"
      }
    }
  }

  api {
    endpoint "/e" {
      proxy {
        backend {
          origin  = "${env.COUPER_TEST_BACKEND_ADDR}"

          openapi {
            file = "01_schema.yaml"
          }
        }
      }

      error_handler "backend_openapi_validation" {
        // should win
        response {
          status = 405
          body = "endpoint:backend_openapi_validation"
        }
      }
    }

    error_handler "*" {
      response {
        status = 400
        body = "api:*"
      }
    }
  }

  api {
    endpoint "/f" {
      proxy {
        backend {
          origin  = "${env.COUPER_TEST_BACKEND_ADDR}"

          openapi {
            file = "01_schema.yaml"
          }
        }
      }

      error_handler "*" {
        // should win
        response {
          status = 405
          body = "endpoint:*"
        }
      }
    }

    error_handler "backend_openapi_validation" {
      response {
        status = 400
        body = "api:backend_openapi_validation"
      }
    }
  }

  api {
    endpoint "/g" {
      proxy {
        backend {
          origin  = "${env.COUPER_TEST_BACKEND_ADDR}"

          openapi {
            file = "01_schema.yaml"
          }
        }
      }

      error_handler "backend" {
        // should win
        response {
          status = 405
          body = "endpoint:backend"
        }
      }
    }

    error_handler "*" {
      response {
        status = 400
        body = "api:*"
      }
    }
  }

  api {
    endpoint "/h" {
      proxy {
        backend {
          origin  = "${env.COUPER_TEST_BACKEND_ADDR}"

          openapi {
            file = "01_schema.yaml"
          }
        }
      }

      error_handler "*" {
        // should win
        response {
          status = 405
          body = "endpoint:*"
        }
      }
    }

    error_handler "backend" {
      response {
        status = 400
        body = "api:backend"
      }
    }
  }
}
