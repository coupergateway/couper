server {
  endpoint "/ac1" {
    access_control = ["at"]
    required_permission = "rp"

    response {
    }

    error_handler "*" {
      response {
        status = 204
        headers = {
          from = "*"
        }
      }
    }
  }

  endpoint "/ac2" {
    access_control = ["at"]
    required_permission = "rp"

    response {
    }

    error_handler "*" {
      response {
        status = 204
        headers = {
          from = "*"
        }
      }
    }

    error_handler "access_control" {
      response {
        status = 204
        headers = {
          from = "access_control"
        }
      }
    }
  }

  endpoint "/ac3" {
    access_control = ["at"]
    required_permission = "rp"

    response {
    }

    error_handler "*" {
      response {
        status = 204
        headers = {
          from = "*"
        }
      }
    }

    error_handler "access_control" {
      response {
        status = 204
        headers = {
          from = "access_control"
        }
      }
    }

    error_handler "insufficient_permissions" {
      response {
        status = 204
        headers = {
          from = "insufficient_permissions"
        }
      }
    }
  }

  endpoint "/ep1" {
    request "r" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/not_there"
      expected_status = [200]
    }

    response {
    }

    error_handler "*" {
      response {
        status = 204
        headers = {
          from = "*"
        }
      }
    }
  }

  endpoint "/ep2" {
    request "r" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/not_there"
      expected_status = [200]
    }

    response {
    }

    error_handler "*" {
      response {
        status = 204
        headers = {
          from = "*"
        }
      }
    }

    error_handler "endpoint" {
      response {
        status = 204
        headers = {
          from = "endpoint"
        }
      }
    }
  }

  endpoint "/ep3" {
    request "r" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/not_there"
      expected_status = [200]
    }

    response {
    }

    error_handler "*" {
      response {
        status = 204
        headers = {
          from = "*"
        }
      }
    }

    error_handler "endpoint" {
      response {
        status = 204
        headers = {
          from = "endpoint"
        }
      }
    }

    error_handler "unexpected_status" {
      response {
        status = 204
        headers = {
          from = "unexpected_status"
        }
      }
    }
  }

  endpoint "/be1" {
    proxy {
      backend {
        origin = "${env.COUPER_TEST_BACKEND_ADDR}"
        timeout = "1ns"
      }
    }

    error_handler "*" {
      response {
        status = 204
        headers = {
          from = "*"
        }
      }
    }
  }

  endpoint "/be2" {
    proxy {
      backend {
        origin = "${env.COUPER_TEST_BACKEND_ADDR}"
        timeout = "1ns"
      }
    }

    error_handler "*" {
      response {
        status = 204
        headers = {
          from = "*"
        }
      }
    }

    error_handler "backend" {
      response {
        status = 204
        headers = {
          from = "backend"
        }
      }
    }
  }

  endpoint "/be3" {
    proxy {
      backend {
        origin = "${env.COUPER_TEST_BACKEND_ADDR}"
        timeout = "1ns"
      }
    }

    error_handler "*" {
      response {
        status = 204
        headers = {
          from = "*"
        }
      }
    }

    error_handler "backend" {
      response {
        status = 204
        headers = {
          from = "backend"
        }
      }
    }

    error_handler "backend_timeout" {
      response {
        status = 204
        headers = {
          from = "backend_timeout"
        }
      }
    }
  }

  endpoint "/be4" {
    proxy {
      backend {
        origin = "${env.COUPER_TEST_BACKEND_ADDR}"
        timeout = "1ns"
      }
    }

    error_handler "backend" {
      response {
        status = 204
        headers = {
          from = "backend"
        }
      }
    }

    error_handler "backend_timeout" {
      response {
        status = 204
        headers = {
          from = "backend_timeout"
        }
      }
    }
  }

  endpoint "/be5" {
    proxy {
      backend {
        origin = "${env.COUPER_TEST_BACKEND_ADDR}"
        timeout = "1ns"
      }
    }

    error_handler "backend" {
      response {
        status = 204
        headers = {
          from = "backend"
        }
      }
    }
  }

  endpoint "/be-dial" {
    proxy {
      backend {
        origin = "http://unkn.own"
      }
    }

    error_handler "*" {
      response {
        status = 204
        headers = {
          from = "*"
        }
      }
    }

    error_handler "backend" {
      response {
        status = 204
        headers = {
          from = "backend"
        }
      }
    }
  }
}

definitions {
  jwt "at" {
    signature_algorithm = "HS256"
    key = "asdf"
    permissions_claim = "p"
  }
}
