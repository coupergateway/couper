server {}

definitions {
  beta_job "withStartupDelay" {
    interval = "1m"
    startup_delay = "200ms"

    request {
      url = "{{ .origin }}"
    }
  }
}
