server {
  endpoint "/**" {
    proxy {
      url = "{{ .origin }}/**"
    }
  }
}
