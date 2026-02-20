server {
  endpoint "/" {
    proxy {
      backend {
        throttle {
          period = "1m"
          per_period = 60
        }
      }
    }
  }
}
