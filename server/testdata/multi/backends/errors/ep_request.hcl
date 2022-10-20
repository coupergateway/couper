server {
  endpoint "/" {
    request {
      backend {
        origin = "https:/example.com"
      }
      backend = "BE"
    }
  }
}
