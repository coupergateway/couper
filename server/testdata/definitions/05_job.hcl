server {}

definitions {
  beta_job "withLabel" {
    interval = "1m"

    request "first" {
      url = "{{ .origin }}"
      expected_status = [418]
    }
  }
}
