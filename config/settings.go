package config

type Settings struct {
	DefaultPort     int    `hcl:"default_port,optional"`
	HealthPath      string `hcl:"health_path,optional"`
	LogFormat       string `hcl:"log_format,optional"`
	XForwardedHost  bool   `hcl:"xfh,optional"`
  RequestIDFormat string `hcl:"request_id_format,optional"`
}
