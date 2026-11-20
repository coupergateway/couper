server {
  endpoint "/configuration" {
    proxy {
      backend {
        # no origin: it evaluates to an empty string, raising a configuration error
      }
    }

    error_handler "configuration" {
      response {
        status = 403
        json_body = {
          handled_by = "configuration"
        }
      }
    }
  }

  endpoint "/wildcard" {
    proxy {
      backend {
        # no origin: same configuration error, caught by the super-type "*"
      }
    }

    error_handler "*" {
      response {
        status = 403
        json_body = {
          handled_by = "*"
        }
      }
    }
  }
}
