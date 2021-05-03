server "log-format-common" {
  access_control = ["undefined"]
  endpoint "/" {
    response {
      status = 500
    }
  }
}
