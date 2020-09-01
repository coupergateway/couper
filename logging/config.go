package logging

type Config struct {
	RequestHeaders, ResponseHeaders []string
}

var DefaultConfig = Config{
	RequestHeaders:  []string{"User-Agent", "Accept", "Referer"},
	ResponseHeaders: []string{"Cache-Control", "Content-Encoding", "Content-Type", "Location"},
}
