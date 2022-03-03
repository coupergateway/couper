server {
  endpoint "/" {
    request "a" {
      url = "{{ .origin }}/"
      backend = "be"
    }
    request {
      url = "{{ .origin }}/"
      backend = "be"
    }
    response {
      body = "a"
    }
  }

  endpoint "/named" {
    request "a" {
      url = "{{ .origin }}/"
      backend = "be"
    }
    request "named" {
      url = "{{ .origin }}/"
      backend = "be"
    }
    response {
      body = "a"
    }
  }

  endpoint "/default" {
    request "a" {
      url = "{{ .origin }}/"
      backend = "be"
    }
    request {
      url = "{{ .origin }}/"
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
