server {
  endpoint "/" {
    proxy {
      backend {
        origin = "{{ .origin }}"
        path = "/anonymous"
        max_connections = 1
      }
    }
  }

  endpoint "/be" {
    proxy {
      backend = "be"
    }
  }

  endpoint "/fake-sequence" {
    request {
      backend = "seq"
    }

    request "two" {
      url = "{{ .origin }}/two"
      backend = "seq"
    }

    request "three" {
      url = "{{ .origin }}/three"
      backend = "seq"
    }
  }
}

definitions {
  backend "be" {
    origin = "{{ .origin }}"
    path = "/reference"
    max_connections = 1
  }

  backend "seq" {
    origin = "{{ .origin }}"
    path = "/sequence"
    max_connections = 1
  }
}
