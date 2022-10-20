server {
  endpoint "/" {
    proxy {
      backend {
        origin = "https:/example.com"
      }
      backend = "BE"
    }
  }
}
