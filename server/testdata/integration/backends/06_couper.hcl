server {
  endpoint "/anon" {
    proxy {
      url = "{{ .origin }}/"
    }
  }

  endpoint "/ref" {
    proxy {
      backend = "be"
    }
  }

  endpoint "/catch" {
    proxy {
      backend = "be"
    }

    error_handler "backend_unhealthy" {
      response {
        status = 418
      }
    }
  }
}

definitions {
  backend "be" {
    origin = "{{ .origin }}"
    beta_health {
      expected_status = [204]
      failure_threshold = 0
      interval = "250ms"
    }
  }
}
