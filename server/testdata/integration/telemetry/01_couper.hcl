server "metrics" {
  endpoint "/" {
    proxy {
      backend = "anything"
    }
  }
}


definitions {
  # backend origin within a definition block gets replaced with the integration test "anything" server.
  backend "anything" {
    path = "/anything"
    origin = env.COUPER_TEST_BACKEND_ADDR
  }
}

settings {
  metrics = true
  metrics_exporter = "prometheus"
}
