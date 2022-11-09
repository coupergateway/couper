package producer

import (
	"net/http"
)

var (
	_ Roundtrip = Proxies{}
	_ Roundtrip = Requests{}
	_ Roundtrip = SequenceParallel{}
	_ Roundtrip = Sequence{}
)

type Roundtrip interface {
	Produce(req *http.Request) chan *Result
	Len() int
}
