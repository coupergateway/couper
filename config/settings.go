package config

// DefaultSettings defines the <DefaultSettings> object.
var DefaultSettings = Settings{
	DefaultPort:     8080,
	HealthPath:      "/healthz",
	LogFormat:       "common",
	NoProxyFromEnv:  false,
	RequestIDFormat: "common",
	XForwardedHost:  false,
}

// Settings represents the <Settings> object.
type Settings struct {
	DefaultPort     int    `hcl:"default_port,optional"`
	HealthPath      string `hcl:"health_path,optional"`
	LogFormat       string `hcl:"log_format,optional"`
	LogPretty       bool   `hcl:"log_pretty,optional"`
	NoProxyFromEnv  bool   `hcl:"no_proxy_from_env,optional"`
	RequestIDFormat string `hcl:"request_id_format,optional"`
	SecureCookies   string `hcl:"secure_cookies,optional"`
	XForwardedHost  bool   `hcl:"xfh,optional"`
}
