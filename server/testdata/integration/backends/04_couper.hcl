server {
  endpoint "/" {
    request "anything" {
      url = "{{ .origin }}/headers"
      backend = "be"
    }
    request {
      url = "{{ .origin }}/delay/1"
      backend = "be"
    }
    response {
      body = "a"
    }
  }

  endpoint "/named" {
    request "anything" {
      url = "{{ .origin }}/headers"
      backend = "be"
    }
    request "named" {
      url = "{{ .origin }}/delay/1"
      backend = "be"
    }
    response {
      body = "a"
    }
  }

  endpoint "/default" {
    request "anything" {
      url = "{{ .origin }}/headers"
      backend = "be"
    }
    request {
      url = "{{ .origin }}/delay/1"
      backend = "be"
    }
  }
}

definitions {
  backend "be" {
    origin = "{{ .origin }}"
    max_connections = 1
  }
}
