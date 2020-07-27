package handler

import (
	"net/http"
)

type Selectable interface {
	HasResponse(req *http.Request) bool
}
