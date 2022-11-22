package producer

import (
	"net/http"
	"sync"
)

var (
	_ Roundtrip = Proxies{}
	_ Roundtrip = Requests{}
	_ Roundtrip = Parallel{}
	_ Roundtrip = Sequence{}
)

type Roundtrip interface {
	Produce(req *http.Request, additionalChs *sync.Map) chan *Result
	Len() int
	Names() []string
}
