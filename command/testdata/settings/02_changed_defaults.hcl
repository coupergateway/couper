server "basic" {}
settings {
  default_port = 9090
  health_path = "/status/health"
  no_proxy_from_env = true
  # log_format has own tests
  request_id_format = "uuid4"
  xfh = true
}
