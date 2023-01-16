server "first" {
  hosts = ["*:9090"]

  endpoint "/" {
    proxy {
      backend = "b1"
    }
  }
}

settings {
  server_timing_headers = true
}

definitions {
  backend "b1" {
    origin = "http://localhost:8080"
  }
}
