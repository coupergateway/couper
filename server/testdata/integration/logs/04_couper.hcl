server {
  api {
    endpoint "/**" {
      custom_log_fields = foo() # error!
      response {
        status = 204
      }
    }
  }
}

settings {
  log_level = "debug"
}
