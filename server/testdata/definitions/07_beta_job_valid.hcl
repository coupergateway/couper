server {}

definitions {
  beta_job "withLabel" {
    interval = "1m"

    request "first" {
      url = "{{ .origin }}"
    }

    request "second" {
      url = "{{ .origin }}"
      method = "POST"
      json_body = merge(backend_responses.first.json_body, request)
    }

    custom_log_fields = {
      status_a: backend_responses.first.status,
      status_b: backend_responses.second.status,
      client: request
    }
  }
}
