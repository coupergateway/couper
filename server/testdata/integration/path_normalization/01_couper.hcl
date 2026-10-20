# Gateway under test on the default port. The origin is a plain test
# server that echoes the request target byte for byte.
server "gateway" {
  hosts = ["*:8080"]

  custom_log_fields = {
    path = request.path
  }

  endpoint "/admin/**" {
    access_control = ["creds"]
    proxy {
      backend = "origin"
    }
  }

  endpoint "/limited" {
    access_control = ["rl"]
    response {
      body = "limited"
    }
  }

  endpoint "/**" {
    proxy {
      backend = "origin"
    }
  }
}

definitions {
  basic_auth "creds" {
    user     = "alice"
    password = "secret"
  }

  beta_rate_limiter "rl" {
    period     = "60s"
    per_period = 1
    key        = request.path
  }

  backend "origin" {
    origin = env.COUPER_TEST_RAW_BACKEND_ADDR
  }
}
