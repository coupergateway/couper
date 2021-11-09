server "backend_probes" {
  endpoint "/health" {
    response {
      headers = {
        states = json_encode([
          backend_probes.UNKNOWN.state,
          backend_probes.healthy.state,
          backend_probes.healthy2.state,
          backend_probes.down.state,
          backend_probes.down2.state,
          backend_probes.degraded.state
        ])
      }
    }
  }

  endpoint "/dummy" {
    request "default" {
      backend = "healthy"
    }
    request "healthy2" {
      backend = "healthy2"
    }
    request "down" {
      backend = "down"
    }
    request "down2" {
      backend = "down2"
    }
    request "degraded" {
      backend = "degraded"
    }
  }
}

definitions {
  backend "healthy" {
    origin = "${env.COUPER_TEST_BACKEND_ADDR}/small"
    beta_health {}
  }
  backend "healthy2" {
    origin = "${env.COUPER_TEST_BACKEND_ADDR}/not-there"
    beta_health {
      expect_status = 404
    }
  }
  backend "down" {
    origin = "http://1.2.3.4"
    beta_health {}
  }
  backend "down2" {
    origin = "${env.COUPER_TEST_BACKEND_ADDR}"
    beta_health {}
  }
  backend "degraded" {
    origin = "http://1.2.3.4"
    beta_health {
      failure_threshold = 4
    }
  }
}
