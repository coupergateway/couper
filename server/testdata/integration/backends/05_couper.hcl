server {
  endpoint "/**" {
    proxy {
      path = "/**"
      url = "{{ .origin }}/"
    }
  }
}
