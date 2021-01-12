package logging

type Config struct {
	NoProxyFromEnv  bool
	ParentFieldKey  string   `env:"log_parent_field"`
	RequestHeaders  []string `env:"log_request_headers"`
	ResponseHeaders []string `env:"log_response_headers"`
	TypeFieldKey    string   `env:"log_type_value"`
}

var DefaultConfig = &Config{
	RequestHeaders:  []string{"User-Agent", "Accept", "Referer"},
	ResponseHeaders: []string{"Cache-Control", "Content-Encoding", "Content-Type", "Location"},
}
