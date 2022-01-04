server {
  api {
    endpoint "/" {
      request "b" {
        url = "http://localhost/b"
        form_body = {
          a = backend_responses.a.status
        }
      }

      request "a" {
        url = "http://localhost/a"
        form_body = {
          aa = backend_responses.b.status
        }
      }

      response {
        json_body = {
          a = backend_responses.a.status
        }
      }
    }
  }
}
