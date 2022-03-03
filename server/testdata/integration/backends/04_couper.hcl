server {
  endpoint "/" {
    request "anything" {
      url = "{{ .origin }}/headers"
      backend = "httpbin"
    }
    request {
      url = "{{ .origin }}/delay/1"
      backend = "httpbin"
    }
    response {
      body = "a"
    }
  }
}

definitions {
  backend "httpbin" {
    origin = "{{ .origin }}"
    max_connections = 1
  }
}
