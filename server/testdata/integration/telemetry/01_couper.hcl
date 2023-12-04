server "metrics" {
  endpoint "/" {
    proxy {
      backend = "anything"
    }
  }

  endpoint "/down" {
    proxy {
      backend = "not_healthy"
    }
  }
}


definitions {
  # backend origin within a definition block gets replaced with the integration test "anything" server.
  backend "anything" {
    path = "/anything"
    origin = env.COUPER_TEST_BACKEND_ADDR
  }

  backend "not_healthy" {
    origin = "http://1.2.3.4"
    timeout = "2s"
    beta_health {
      timeout = "250ms"
    }
  }
}

settings {
  beta_metrics = true
  beta_service_name = "my-service"
  no_proxy_from_env = true
}
