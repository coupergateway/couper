package config

var DefaultSettings = Settings{
	DefaultPort:     8080,
	HealthPath:      "/healthz",
	LogFormat:       "common",
	RequestIDFormat: "common",
	XForwardedHost:  false,
}

type Settings struct {
	DefaultPort     int    `hcl:"default_port,optional"`
	HealthPath      string `hcl:"health_path,optional"`
	LogFormat       string `hcl:"log_format,optional"`
	RequestIDFormat string `hcl:"request_id_format,optional"`
	XForwardedHost  bool   `hcl:"xfh,optional"`
}
