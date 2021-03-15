server "settings" {
  api {
    endpoint "/" {
      proxy {
        backend {
          origin = "http://example.com"
        }
      }
    }
  }
}

settings {
  secure_cookies = "enforce"
}
