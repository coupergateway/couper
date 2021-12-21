package producer

import (
	"net/http"
)

var (
	_ Roundtrips = Proxies{}
	_ Roundtrips = Requests{}
	_ Roundtrips = Sequences{}
	_ Roundtrips = Sequence{}
)

type Roundtrips interface {
	Produce(req *http.Request, results chan<- *Result)
	Len() int
}
