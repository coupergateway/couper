server {
  endpoint "/" {
    allowed_methods = ["in valid"]
    response {
      body = "a"
    }
  }
}
