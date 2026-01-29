server {}

definitions {
  beta_job "withNegativeStartupDelay" {
    interval = "1m"
    startup_delay = "-100ms"

    request {}
  }
}
