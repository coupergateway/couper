package logging

type Config struct {
	ParentFieldKey                  string
	TypeFieldKey                    string
	RequestHeaders, ResponseHeaders []string
	UseXFF                          bool
}

var DefaultConfig = &Config{
	RequestHeaders:  []string{"User-Agent", "Accept", "Referer"},
	ResponseHeaders: []string{"Cache-Control", "Content-Encoding", "Content-Type", "Location"},
}
