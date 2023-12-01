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

  endpoint "/default2" {
    request {
      url = "{{ .origin }}/"
      backend = "be"
    }
    request "a" {
      url = "{{ .origin }}/"
      backend = "be"
    }
  }

  endpoint "/ws" {
    proxy {
      url = "{{ .origin }}/"
      websockets = true
      backend = "be"
    }
  }

  endpoint "/proxy-seq" {
    request "p" {
      url = "{{ .origin }}/"
      backend = "be"
    }

    proxy {
      url = "{{ .origin }}/"
      backend = "be"
    }
  }

  endpoint "/proxy-seq-ref" {
    request "seq" {
      url = "{{ .origin }}/"
      backend = "be"
    }

    proxy {
      set_response_headers = {
        x-a: backend_responses.seq.body
      }
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
