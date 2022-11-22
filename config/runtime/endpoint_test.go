package runtime_test

import (
	"context"
	"testing"

	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config/configload"
	"github.com/avenga/couper/config/runtime"
	"github.com/avenga/couper/config/runtime/server"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler/producer"
	"github.com/avenga/couper/internal/test"
)

func TestEndpoint_Same_Requests(t *testing.T) {
	helper := test.New(t)
	hcl := `server {
  api {
    endpoint "/" {
      request "r1" {
        url = "/res1"
        backend = "ba"
      }

      request "r2" {
        url = "/res2"
        backend = "ba"
        json_body = backend_responses.r1.json_body
      }

      request {
        url = "/"
        backend = "bb"
        json_body = [
          backend_responses.r1.json_body
          ,
          backend_responses.r2.json_body
        ]
      }
    }
  }
}
definitions {
  backend "ba" {
    origin = "https://ba.example.com"
  }
  backend "bb" {
    origin = "https://bb.example.com"
  }
}
`
	conf, err := configload.LoadBytes([]byte(hcl), "couper.hcl")
	helper.Must(err)

	log, _ := test.NewLogger()
	logger := log.WithContext(context.TODO())

	tmpStoreCh := make(chan struct{})
	defer close(tmpStoreCh)

	evalCtx := conf.Context.(*eval.Context)
	confCtx := evalCtx.HCLContext()
	srvConf := conf.Servers[0]
	serverOptions, err := server.NewServerOptions(srvConf, logger)
	helper.Must(err)

	apiConf := srvConf.APIs[0]
	endpointConf := apiConf.Endpoints[0]

	endpointOptions, err := runtime.NewEndpointOptions(confCtx, endpointConf, apiConf, serverOptions, logger, conf, cache.New(logger, tmpStoreCh))
	helper.Must(err)
	sequences := endpointOptions.Sequences.(producer.Parallel)

	var r1, r1_, r2 *producer.Requests
	if s, ok := sequences[0].(*producer.Sequence); ok {
		vs := *s
		if p, ok := vs[0].(*producer.Parallel); ok {
			vp := *p
			if s, ok := vp[0].(*producer.Sequence); ok {
				vs := *s
				if r, ok := vs[0].(*producer.Requests); ok {
					r1_ = r
				}
			}
			if r, ok := vp[1].(*producer.Requests); ok {
				r1 = r
			}
		}
		if r, ok := vs[1].(*producer.Requests); ok {
			r2 = r
		}
	}
	if r1 == nil {
		t.Error("expected r1 not to be nil")
	}
	if r1_ == nil {
		t.Error("expected r1_ not to be nil")
	}
	if r2 == nil {
		t.Error("expected r2 not to be nil")
	}
	if r1 != r1_ {
		t.Error("expected same *producer.Requests")
	}
	if r1 == r2 {
		t.Error("did not expect same *producer.Requests")
	}
}

func TestEndpoint_Same_Proxies(t *testing.T) {
	helper := test.New(t)
	hcl := `server {
  api {
    endpoint "/" {
      proxy "p1" {
        backend "ba" {
          path = "/res1"
        }
      }

      request "r2" {
        url = "/res2"
        backend = "ba"
        json_body = backend_responses.p1.json_body
      }

      proxy "p3" {
        backend "ba" {
          path = "/res3"
        }
      }

      request {
        url = "/"
        backend = "bb"
        json_body = [
          backend_responses.p1.json_body
          ,
          backend_responses.r2.json_body
          ,
          backend_responses.p3.json_body
        ]
      }
    }
  }
}
definitions {
  backend "ba" {
    origin = "https://ba.example.com"
  }
  backend "bb" {
    origin = "https://bb.example.com"
  }
}
`
	conf, err := configload.LoadBytes([]byte(hcl), "couper.hcl")
	helper.Must(err)

	log, _ := test.NewLogger()
	logger := log.WithContext(context.TODO())

	tmpStoreCh := make(chan struct{})
	defer close(tmpStoreCh)

	evalCtx := conf.Context.(*eval.Context)
	confCtx := evalCtx.HCLContext()
	srvConf := conf.Servers[0]
	serverOptions, err := server.NewServerOptions(srvConf, logger)
	helper.Must(err)

	apiConf := srvConf.APIs[0]
	endpointConf := apiConf.Endpoints[0]

	endpointOptions, err := runtime.NewEndpointOptions(confCtx, endpointConf, apiConf, serverOptions, logger, conf, cache.New(logger, tmpStoreCh))
	helper.Must(err)
	sequences := endpointOptions.Sequences.(producer.Parallel)

	var p1, p1_, p3 *producer.Proxies
	if s, ok := sequences[0].(*producer.Sequence); ok {
		vs := *s
		if p, ok := vs[0].(*producer.Parallel); ok {
			vp := *p
			if p, ok := vp[0].(*producer.Proxies); ok {
				p3 = p
			}
			if s, ok := vp[1].(*producer.Sequence); ok {
				vs := *s
				if p, ok := vs[0].(*producer.Proxies); ok {
					p1_ = p
				}
			}
			if p, ok := vp[2].(*producer.Proxies); ok {
				p1 = p
			}
		}
	}
	if p1 == nil {
		t.Error("expected p1 not to be nil")
	}
	if p1_ == nil {
		t.Error("expected p1_ not to be nil")
	}
	if p3 == nil {
		t.Error("expected p3 not to be nil")
	}
	if p1 != p1_ {
		t.Error("expected same *producer.Proxies")
	}
	if p1 == p3 {
		t.Error("did not expect same *producer.Proxies")
	}
}

func TestEndpoint_Same_Parallel(t *testing.T) {
	helper := test.New(t)
	hcl := `server {
  api {
    endpoint "/" {
      request "r1" {
        url = "/res1"
        backend = "ba"
      }

      request "r2" {
        url = "/res2"
        backend = "ba"
      }

      request "r3" {
        url = "/res3"
        backend = "ba"
        json_body = [
          backend_responses.r1.json_body
          ,
          backend_responses.r2.json_body
        ]
      }

      request "r4" {
        url = "/res4"
        backend = "ba"
        json_body = [
          backend_responses.r1.json_body
          ,
          backend_responses.r2.json_body
        ]
      }

      request {
        url = "/"
        backend = "bb"
        json_body = [
          backend_responses.r3.json_body
          ,
          backend_responses.r4.json_body
        ]
      }
    }
  }
}
definitions {
  backend "ba" {
    origin = "https://ba.example.com"
  }
  backend "bb" {
    origin = "https://bb.example.com"
  }
}
`
	conf, err := configload.LoadBytes([]byte(hcl), "couper.hcl")
	helper.Must(err)

	log, _ := test.NewLogger()
	logger := log.WithContext(context.TODO())

	tmpStoreCh := make(chan struct{})
	defer close(tmpStoreCh)

	evalCtx := conf.Context.(*eval.Context)
	confCtx := evalCtx.HCLContext()
	srvConf := conf.Servers[0]
	serverOptions, err := server.NewServerOptions(srvConf, logger)
	helper.Must(err)

	apiConf := srvConf.APIs[0]
	endpointConf := apiConf.Endpoints[0]

	endpointOptions, err := runtime.NewEndpointOptions(confCtx, endpointConf, apiConf, serverOptions, logger, conf, cache.New(logger, tmpStoreCh))
	helper.Must(err)
	sequences := endpointOptions.Sequences.(producer.Parallel)

	var p1, p1_, p2 *producer.Parallel
	if s, ok := sequences[0].(*producer.Sequence); ok {
		vs := *s
		if p, ok := vs[0].(*producer.Parallel); ok {
			p2 = p
			vp := *p
			if s, ok := vp[0].(*producer.Sequence); ok {
				vs := *s
				if p, ok := vs[0].(*producer.Parallel); ok {
					p1 = p
				}
			}
			if s, ok := vp[1].(*producer.Sequence); ok {
				vs := *s
				if p, ok := vs[0].(*producer.Parallel); ok {
					p1_ = p
				}
			}
		}
	}
	if p1 == nil {
		t.Error("expected p1 not to be nil")
	}
	if p1_ == nil {
		t.Error("expected p1_ not to be nil")
	}
	if p2 == nil {
		t.Error("expected p2 not to be nil")
	}
	if p1 != p1_ {
		t.Error("expected same *producer.Parallel")
	}
	if p1 == p2 {
		t.Error("did not expect same *producer.Parallel")
	}
}

func TestEndpoint_Same_Sequence(t *testing.T) {
	helper := test.New(t)
	hcl := `server {
  api {
    endpoint "/" {
      request "r1" {
        url = "/res1"
        backend = "ba"
      }

      request "r2" {
        url = "/res2"
        backend = "ba"
        json_body = backend_responses.r1.json_body
      }

      request "r3" {
        url = "/res3"
        backend = "ba"
        json_body = backend_responses.r2.json_body
      }

      request "r4" {
        url = "/res4"
        backend = "ba"
        json_body = backend_responses.r2.json_body
      }

      request {
        url = "/"
        backend = "bb"
        json_body = [
          backend_responses.r3.json_body
          ,
          backend_responses.r4.json_body
        ]
      }
    }
  }
}
definitions {
  backend "ba" {
    origin = "https://ba.example.com"
  }
  backend "bb" {
    origin = "https://bb.example.com"
  }
}
`
	conf, err := configload.LoadBytes([]byte(hcl), "couper.hcl")
	helper.Must(err)

	log, _ := test.NewLogger()
	logger := log.WithContext(context.TODO())

	tmpStoreCh := make(chan struct{})
	defer close(tmpStoreCh)

	evalCtx := conf.Context.(*eval.Context)
	confCtx := evalCtx.HCLContext()
	srvConf := conf.Servers[0]
	serverOptions, err := server.NewServerOptions(srvConf, logger)
	helper.Must(err)

	apiConf := srvConf.APIs[0]
	endpointConf := apiConf.Endpoints[0]

	endpointOptions, err := runtime.NewEndpointOptions(confCtx, endpointConf, apiConf, serverOptions, logger, conf, cache.New(logger, tmpStoreCh))
	helper.Must(err)
	sequences := endpointOptions.Sequences.(producer.Parallel)

	var s1, s1_, s2 *producer.Sequence
	if s, ok := sequences[0].(*producer.Sequence); ok {
		s2 = s
		vs := *s
		if p, ok := vs[0].(*producer.Parallel); ok {
			vp := *p
			if s, ok := vp[0].(*producer.Sequence); ok {
				vs := *s
				if s, ok := vs[0].(*producer.Sequence); ok {
					s1 = s
				}
			}
			if s, ok := vp[1].(*producer.Sequence); ok {
				vs := *s
				if s, ok := vs[0].(*producer.Sequence); ok {
					s1_ = s
				}
			}
		}
	}
	if s1 == nil {
		t.Error("expected s1 not to be nil")
	}
	if s1_ == nil {
		t.Error("expected s1_ not to be nil")
	}
	if s2 == nil {
		t.Error("expected s2 not to be nil")
	}
	if s1 != s1_ {
		t.Error("expected same *producer.Sequence")
	}
	if s1 == s2 {
		t.Error("did not expect same *producer.Sequence")
	}
}
