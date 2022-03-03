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

  endpoint "/named" {
    request "anything" {
      url = "{{ .origin }}/headers"
      backend = "httpbin"
    }
    request "named" {
      url = "{{ .origin }}/delay/1"
      backend = "httpbin"
    }
    response {
      body = "a"
    }
  }

  endpoint "/default" {
    request "anything" {
      url = "{{ .origin }}/headers"
      backend = "httpbin"
    }
    request {
      url = "{{ .origin }}/delay/1"
      backend = "httpbin"
    }
  }
}

definitions {
  backend "httpbin" {
    origin = "{{ .origin }}"
    max_connections = 1
  }
}
