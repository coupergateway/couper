definitions {
  jwt "JWT" {
    error_handler {
      proxy {
        backend {
          origin = "https:/example.com"
        }
        backend = "BE"
      }
    }
  }
}
