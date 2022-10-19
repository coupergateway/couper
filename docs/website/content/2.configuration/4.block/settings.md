# Settings

The `settings` block lets you configure the more basic and global behavior of your
gateway instance.

::attributes
---
values: [
  {
    "default": "[]",
    "description": "Which `X-Forwarded-*` request headers should be accepted to change the [request variables](../variables#request) `url`, `origin`, `protocol`, `host`, `port`. Valid values: `\"proto\"`, `\"host\"` and `\"port\"`. The port in a `X-Forwarded-Port` header takes precedence over a port in `X-Forwarded-Host`. Affects relative URL values for [`sp_acs_url`](saml) attribute and `redirect_uri` attribute within [`beta_oauth2`](oauth2) and [`oidc`](oidc).",
    "name": "accept_forwarded_url",
    "type": "tuple (string)"
  },
  {
    "default": "false",
    "description": "enables the Prometheus [metrics](/observation/metrics) exporter",
    "name": "beta_metrics",
    "type": "bool"
  },
  {
    "default": "9090",
    "description": "Prometheus exporter listen port",
    "name": "beta_metrics_port",
    "type": "number"
  },
  {
    "default": "\"couper\"",
    "description": "service name which applies to the `service_name` metric labels",
    "name": "beta_service_name",
    "type": "string"
  },
  {
    "default": "",
    "description": "adds the given PEM encoded CA certificate to the existing system certificate pool for all outgoing connections",
    "name": "ca_file",
    "type": "string"
  },
  {
    "default": "8080",
    "description": "Port which will be used if not explicitly specified per host within the [`hosts`](server) attribute.",
    "name": "default_port",
    "type": "number"
  },
  {
    "default": "",
    "description": "[environment](../command-line#global-options) Couper is to run in",
    "name": "environment",
    "type": "string"
  },
  {
    "default": "\"/healthz\"",
    "description": "Health path for all configured servers and ports",
    "name": "health_path",
    "type": "string"
  },
  {
    "default": "",
    "description": "TLS port mappings to define the TLS listen port and the target one. Self-signed certificates will be generated on the fly based on the given hostname. Certificates will be held in memory.",
    "name": "https_dev_proxy",
    "type": "object"
  },
  {
    "default": "\"common\"",
    "description": "tab/field based colored logs or JSON logs: `\"common\"` or `\"json\"`",
    "name": "log_format",
    "type": "string"
  },
  {
    "default": "\"info\"",
    "description": "sets the log level: `\"panic\"`, `\"fatal\"`, `\"error\"`, `\"warn\"`, `\"info\"`, `\"debug\"`, `\"trace\"`",
    "name": "log_level",
    "type": "string"
  },
  {
    "default": "false",
    "description": "global option for `json` log format which pretty prints with basic key coloring",
    "name": "log_pretty",
    "type": "bool"
  },
  {
    "default": "false",
    "description": "disables the connect hop to configured [proxy via environment](https://godoc.org/golang.org/x/net/http/httpproxy)",
    "name": "no_proxy_from_env",
    "type": "bool"
  },
  {
    "default": "false",
    "description": "enables [profiling](https://github.com/google/pprof/blob/main/doc/README.md#pprof)",
    "name": "pprof",
    "type": "bool"
  },
  {
    "default": "6060",
    "description": "Port for profiling interface",
    "name": "pprof_port",
    "type": "number"
  },
  {
    "default": "",
    "description": "client request HTTP header field that transports the `request.id` which Couper takes for logging and transport to the backend (if configured)",
    "name": "request_id_accept_from_header",
    "type": "string"
  },
  {
    "default": "\"Couper-Request-ID\"",
    "description": "HTTP header field which Couper uses to transport the `request.id` to the backend",
    "name": "request_id_backend_header",
    "type": "string"
  },
  {
    "default": "\"Couper-Request-ID\"",
    "description": "HTTP header field which Couper uses to transport the `request.id` to the client",
    "name": "request_id_client_header",
    "type": "string"
  },
  {
    "default": "\"common\"",
    "description": "`\"common\"` or `\"uuid4\"`. If set to `\"uuid4\"` an RFC 4122 UUID is used for `request.id` and related log fields. ",
    "name": "request_id_format",
    "type": "string"
  },
  {
    "default": "\"‌\"",
    "description": "`\"\"` or `\"strip\"`. If set to `\"strip\"`, the `Secure` flag is removed from all `Set-Cookie` HTTP header fields.",
    "name": "secure_cookies",
    "type": "string"
  },
  {
    "default": "false",
    "description": "whether to use the `X-Forwarded-Host` header as the request host",
    "name": "xfh",
    "type": "bool"
  }
]

---
::
