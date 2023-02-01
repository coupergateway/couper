server "first" {
  hosts = ["*:9090"]

  endpoint "/" {
    proxy {
      backend = "b1"
    }
  }

  endpoint "/empty" {
    response {
      status = 204
    }
  }
}

settings {
  server_timing_header = true
}

definitions {
  backend "b1" {
    origin = "http://localhost:8080"
  }
}
