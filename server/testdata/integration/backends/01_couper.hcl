server {
  api {
    base_path = "/proxy"

    endpoint "/path" {
      proxy {
        path = "/my-path"
        backend {
          # no origin
        }
      }
    }

    endpoint "/backend-path" {
      proxy {
        backend {
          path = "/my-path"
          # no origin
        }
      }
    }

    endpoint "/url" {
      proxy {
        url = "/my-path"
        backend {
          # no origin
        }
      }
    }
  }

  api {
    base_path = "/request"

    endpoint "/backend-path" {
      request {
        backend {
          path = "/my-path"
          # no origin
        }
      }
    }

    endpoint "/url" {
      proxy {
        url = "/my-path"
        backend {
          # no origin
        }
      }
    }
  }
}
