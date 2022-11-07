package producer

import (
	"net/http"
)

var (
	_ Roundtrip = Proxies{}
	_ Roundtrip = Requests{}
	_ Roundtrip = Sequences{}
	_ Roundtrip = Sequence{}
)

type Roundtrip interface {
	Produce(req *http.Request) chan *Result
	Len() int
}
