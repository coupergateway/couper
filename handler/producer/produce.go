package producer

import (
	"net/http"
)

var (
	_ Roundtrip = &Proxy{}
	_ Roundtrip = &Request{}
)

type Roundtrip interface {
	Produce(req *http.Request) *Result
	SetDependsOn(ps string)
}
