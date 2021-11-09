server "backend_probes" {
  endpoint "/health" {
    response {
      headers = {
        states = json_encode([
          backend_probes.UNKNOWN.state,
          backend_probes.healthy.state,
          backend_probes.down.state,
          backend_probes.degraded.state
        ])
      }
    }
  }

  endpoint "/dummy" {
    request "default" {
      backend = "healthy"
    }
    request "down" {
      backend = "down"
    }
    request "degraded" {
      backend = "degraded"
    }
  }
}

definitions {
  backend "healthy" {
    origin = "${env.COUPER_TEST_BACKEND_ADDR}"
    beta_health {}
  }
  backend "down" {
    origin = "http://1.2.3.4"
    beta_health {}
  }
  backend "degraded" {
    origin = "http://1.2.3.4"
    beta_health {
      failure_threshold = 4
    }
  }
}
