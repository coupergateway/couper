server {
  api {
    endpoint "/" {
      request "r1" {
        url = "http://localhost/r1"
      }

      request "r2" {
        url = "http://localhost/r2"
        form_body = {
          r1 = backend_responses.r1.status
        }
      }

      request {
        url = "http://localhost/def"
        json_body = {
          r1 = backend_responses.r1.status
          r2 = backend_responses.r2.status
        }
      }
    }
  }
}
