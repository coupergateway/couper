package frontend

import "net/http"

type Endpoint struct {
	Backend http.Handler
}
