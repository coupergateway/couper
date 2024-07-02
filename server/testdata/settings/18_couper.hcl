server {
  api {
    endpoint "/**" {
      proxy {
        backend {
          origin = "https://example.com"
        }
        expected_status = [200]
        unexpected_status = [401]
      }
    }
  }
}
