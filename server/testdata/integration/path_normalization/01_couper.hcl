# Gateway under test on the default port. The origin is a second Couper
# server in this file. It echoes the URL it receives, so any dot segment
# or encoding that reaches it is visible to the test.
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

server "origin" {
  hosts = ["*:9898"]

  endpoint "/admin/panel.txt" {
    response {
      body = "TOP SECRET admin panel"
    }
  }

  endpoint "/**" {
    response {
      body = "origin saw ${request.url}"
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
    origin = "http://localhost:9898"
  }
}
