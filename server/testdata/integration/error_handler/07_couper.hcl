server {
  api {
    endpoint "/" {
      request {
        url = "http://localhost:8080/i"
        backend = "be"
        form_body = {
          a = request.query.a[0]
          b = 2
        }
        expected_status = [200]
      }

      error_handler "unexpected_status" {
        response {
          status = 417
          json_body = {
            handled_by = "unexpected_status"
          }
        }
      }

      error_handler "backend_openapi_validation" {
        response {
          status = 418
          json_body = {
            handled_by = "backend_openapi_validation"
          }
        }
      }
    }

    endpoint "/i" {
      response {
        body = "OK"
      }
    }
  }
}

definitions {
  backend "be" {
    openapi {
      file = "03_schema.yaml"
    }
  }
}
