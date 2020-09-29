package config

type Settings struct {
	DefaultPort    int    `hcl:"default_port,optional"`
	XForwardedHost bool   `hcl:"xfh,optional"`
	HealthPath     string `hcl:"health_path,optional"`
	LogFormat      string `hcl:"log_format,optional"`
}
