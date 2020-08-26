package runtime

import "go.avenga.cloud/couper/gateway/errors"

// Mux represents a Mux object.
type Mux struct {
	API       Routes
	APIPath   string
	APIErrTpl *errors.Template
	FS        Routes
	FSPath    string
	FSErrTpl  *errors.Template
	SPA       Routes
	SPAPath   string
}
