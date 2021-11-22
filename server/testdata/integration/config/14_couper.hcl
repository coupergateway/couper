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
  endpoint "/healthy/headers" {
    response {
      json_body = backends.healthy_headers.health
    }
  }
  endpoint "/healthy/no_follow_redirect" {
    response {
      json_body = backends.healthy_no_follow_redirect.health
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
  endpoint "/unhealthy/headers" {
    response {
      json_body = backends.unhealthy_headers.health
    }
  }
  endpoint "/unhealthy/no_follow_redirect" {
    response {
      json_body = backends.unhealthy_no_follow_redirect.health
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
    request "healthy_headers" {
      backend = "healthy_headers"
    }
    request "healthy_no_follow_redirect" {
      backend = "healthy_no_follow_redirect"
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
    request "unhealthy_headers" {
      backend = "unhealthy_headers"
    }
    request "unhealthy_no_follow_redirect" {
      backend = "unhealthy_no_follow_redirect"
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
  backend "healthy_headers" {
    origin = env.COUPER_TEST_BACKEND_ADDR
    beta_health {
      path = "/anything"
      headers = {User-Agent: "Couper-Health-Check"}
      expect_text = "\"UserAgent\":\"Couper-Health-Check\""
    }
  }
  backend "healthy_no_follow_redirect" {
    origin = env.COUPER_TEST_BACKEND_ADDR
    beta_health {
      path = "/redirect?url=/health?redirected"
      expect_status = 302
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
  backend "unhealthy_headers" {
    origin = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
    beta_health {
      headers = {User-Agent = "FAIL"}
      expect_text = "Go-http-client"
    }
  }
  backend "unhealthy_no_follow_redirect" {
    origin = env.COUPER_TEST_BACKEND_ADDR
    beta_health {
      path = "/redirect?url=/health?redirected"
      expect_text = "👍"
    }
  }
  backend "failing" {
    origin = "http://1.2.3.4"
    beta_health {
      failure_threshold = 4
    }
  }
}
