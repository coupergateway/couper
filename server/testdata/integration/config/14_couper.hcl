server {
  endpoint "/unknown" {
    response {
      json_body = backends.UNKNOWN.health
    }
  }
  endpoint "/healthy/default" {
    response {
      json_body = backends.healthy.health
    }
  }
  endpoint "/healthy/expect_status" {
    response {
      json_body = backends.healthy_expect_status.health
    }
  }
  endpoint "/healthy/expect_text" {
    response {
      json_body = backends.healthy_expect_text.health
    }
  }
  endpoint "/healthy/path" {
    response {
      json_body = backends.healthy_path.health
    }
  }
  endpoint "/unhealthy/timeout" {
    response {
      json_body = backends.unhealthy_timeout.health
    }
  }
  endpoint "/unhealthy/bad_status" {
    response {
      json_body = backends.unhealthy_bad_status.health
    }
  }
  endpoint "/unhealthy/bad_expect_status" {
    response {
      json_body = backends.unhealthy_bad_expect_status.health
    }
  }
  endpoint "/unhealthy/bad_expect_text" {
    response {
      json_body = backends.unhealthy_bad_expect_text.health
    }
  }
  endpoint "/unhealthy/bad_path" {
    response {
      json_body = backends.unhealthy_bad_path.health
    }
  }
  endpoint "/failing" {
    response {
      json_body = backends.failing.health
    }
  }

  endpoint "/dummy" {
    request "default" {
      backend = "healthy"
    }
    request "healthy_expect_status" {
      backend = "healthy_expect_status"
    }
    request "healthy_expect_text" {
      backend = "healthy_expect_text"
    }
    request "healthy_path" {
      backend = "healthy_path"
    }
    request "unhealthy_timeout" {
      backend = "unhealthy_timeout"
    }
    request "unhealthy_bad_status" {
      backend = "unhealthy_bad_status"
    }
    request "unhealthy_bad_expect_status" {
      backend = "unhealthy_bad_expect_status"
    }
    request "unhealthy_bad_expect_text" {
      backend = "unhealthy_bad_expect_text"
    }
    request "unhealthy_bad_path" {
      backend = "unhealthy_bad_path"
    }
    request "failing" {
      backend = "failing"
    }
  }
}

definitions {
  backend "healthy" {
    origin = "${env.COUPER_TEST_BACKEND_ADDR}/health"
    beta_health {}
  }
  backend "healthy_expect_status" {
    origin = "${env.COUPER_TEST_BACKEND_ADDR}/not-there"
    beta_health {
      expect_status = 404
    }
  }
  backend "healthy_expect_text" {
    origin = "${env.COUPER_TEST_BACKEND_ADDR}/health"
    beta_health {
      expect_text = "👍"
    }
  }
  backend "healthy_path" {
    origin = env.COUPER_TEST_BACKEND_ADDR
    beta_health {
      path = "/anything?foo=bar"
      expect_text = "\"RawQuery\":\"foo=bar\""
    }
  }
  backend "unhealthy_timeout" {
    origin = "http://1.2.3.4"
    beta_health {}
  }
  backend "unhealthy_bad_status" {
    origin = "${env.COUPER_TEST_BACKEND_ADDR}"
    beta_health {}
  }
  backend "unhealthy_bad_expect_status" {
    origin = "${env.COUPER_TEST_BACKEND_ADDR}/health"
    beta_health {
      expect_status = 500
    }
  }
  backend "unhealthy_bad_expect_text" {
    origin = "${env.COUPER_TEST_BACKEND_ADDR}/health"
    beta_health {
      expect_text = "down?"
    }
  }
  backend "unhealthy_bad_path" {
    origin = "${env.COUPER_TEST_BACKEND_ADDR}/health"
    beta_health {
      path = "/"
    }
  }
  backend "failing" {
    origin = "http://1.2.3.4"
    beta_health {
      failure_threshold = 4
    }
  }
}
