server {}

definitions {
  beta_job "withLabel" {
    interval = "1m"

    request "first" {
      url = "{{ .origin }}"
    }

    request {
      url = "{{ .origin }}"
      method = "POST"
      json_body = backend_responses.first.json_body
    }

    custom_log_fields = {
      status_a: backend_responses.first.status,
      status_b: backend_responses.default.status,
    }
  }
}
