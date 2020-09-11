package logging

type Config struct {
	ParentFieldKey  string   `env:"log_parent_field"`
	TypeFieldKey    string   `env:"log_type_value"`
	RequestHeaders  []string `env:"log_request_headers"`
	ResponseHeaders []string `env:"log_response_headers"`
}

var DefaultConfig = &Config{
	RequestHeaders:  []string{"User-Agent", "Accept", "Referer"},
	ResponseHeaders: []string{"Cache-Control", "Content-Encoding", "Content-Type", "Location"},
}
