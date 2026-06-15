server "ba-argon2-ratelimit" {
  api {
    endpoint "/private" {
      access_control = ["ip_rate", "ba_argon2"]
      response {
        json_body = {
          ok = true
        }
      }
    }
  }
}

definitions {
  # listed before basic_auth so it gates argon2 derivation; static key
  # so every request draws from one budget.
  beta_rate_limiter "ip_rate" {
    period        = "60s"
    per_period    = 2
    period_window = "fixed"
    key           = "static"
  }

  basic_auth "ba_argon2" {
    htpasswd_file = "../files/htpasswd_argon2"
  }
}
