server {
  endpoint "/named" {
    proxy {
      backend {
        # The guard expression from #983: a request whose upstream host is not
        # allowed gets an empty origin, which is a backend_origin error.
        origin = contains(split(",", "allowed.example.com"), request.headers.x-upstream-host) ? "https://${request.headers.x-upstream-host}" : ""
      }
    }

    error_handler "backend_origin" {
      response {
        status = 403
        json_body = {
          handled_by = "backend_origin"
        }
      }
    }
  }

  endpoint "/backend-super" {
    proxy {
      backend {
        # Same conditional origin, caught by the super-type "backend".
        origin = contains(split(",", "allowed.example.com"), request.headers.x-upstream-host) ? "https://${request.headers.x-upstream-host}" : ""
      }
    }

    error_handler "backend" {
      response {
        status = 403
        json_body = {
          handled_by = "backend"
        }
      }
    }
  }

  endpoint "/wildcard" {
    proxy {
      backend {
        # Same conditional origin, caught by the catch-all "*".
        origin = contains(split(",", "allowed.example.com"), request.headers.x-upstream-host) ? "https://${request.headers.x-upstream-host}" : ""
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
