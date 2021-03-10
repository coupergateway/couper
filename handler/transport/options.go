package transport

// Options represents the transport <Options> object.
type Options struct {
	basicAuth  string
	pathPrefix string
}

// NewOptions creates a new transport <Options> object.
func NewOptions(basicAuth, pathPrefix string) *Options {
	return &Options{
		basicAuth:  basicAuth,
		pathPrefix: pathPrefix,
	}
}
