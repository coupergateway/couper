server {
  api {
    endpoint "/anything" {
      proxy {
        backend {
          origin = "${env.COUPER_TEST_BACKEND_ADDR}"

          openapi {
            file = "02_schema.yaml"
          }
        }
      }
    }

    error_handler "backend_openapi_validation" {
      response {
        status = 418
          json_body = {
            req_path = backend_requests.default.path
            resp_status = backend_responses.default.status
            resp_json_body_query = backend_responses.default.json_body.Query
            resp_ct = backend_responses.default.headers.content-type
          }
      }
    }
  }
}
