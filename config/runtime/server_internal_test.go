package runtime

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/runtime/server"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/internal/test"
)

func TestServer_isUnique(t *testing.T) {
	endpoints := make(map[string]bool)

	type testCase struct {
		pattern string
		clean   string
		unique  bool
	}

	for i, tc := range []testCase{
		{"/abc", "/abc", true},
		{"/abc", "/abc", false},
		{"/abc/", "/abc/", true},
		{"/x/{xxx}", "/x/{}", true},
		{"/x/{yyy}", "/x/{}", false},
		{"/x/{xxx}/a/{yyy}", "/x/{}/a/{}", true},
		{"/x/{yyy}/a/{xxx}", "/x/{}/a/{}", false},
	} {
		unique, cleanPattern := isUnique(endpoints, tc.pattern)
		endpoints[cleanPattern] = true

		if unique != tc.unique {
			t.Errorf("%d: Unexpected unique status given: %t", i+1, unique)
		}
		if cleanPattern != tc.clean {
			t.Errorf("%d: Unexpected cleanPattern given: %s, want %s", i+1, cleanPattern, tc.clean)
		}
	}
}

func TestServer_getEndpointsList(t *testing.T) {
	srvConf := &config.Server{
		APIs: []*config.API{
			{
				Endpoints: []*config.Endpoint{
					{Pattern: "/api/1"},
					{Pattern: "/api/2"},
				},
			},
		},
		Endpoints: []*config.Endpoint{
			{Pattern: "/free/1"},
			{Pattern: "/free/2"},
		},
	}

	serverOptions, _ := server.NewServerOptions(nil, nil)
	endpoints, _ := newEndpointMap(srvConf, serverOptions)
	if l := len(endpoints); l != 4 {
		t.Fatalf("Expected 4 endpointes, given %d", l)
	}

	checks := map[string]HandlerKind{
		"/api/1":  api,
		"/api/2":  api,
		"/free/1": endpoint,
		"/free/2": endpoint,
	}

	for pattern, kind := range checks {
		var exist bool
		for endpoint, parent := range endpoints {
			if endpoint.Pattern == pattern {
				if kind == api && parent == nil {
					t.Errorf("Expected an api endpoint for path pattern: %q", pattern)
				}
				exist = true
				break
			}
		}
		if !exist {
			t.Errorf("Expected an endpoint for path pattern: %q", pattern)
		}
	}
}

func TestServer_validatePortHosts(t *testing.T) {
	type args struct {
		conf           *config.Couper
		configuredPort int
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"Same host/port in one server",
			args{
				&config.Couper{
					Servers: []*config.Server{
						{Hosts: []string{"*", "*", "*:9090", "*:9090", "*:8080"}},
					},
				}, 8080,
			},
			false,
		},
		{
			"Same host/port in two servers with *",
			args{
				&config.Couper{
					Servers: []*config.Server{
						{Hosts: []string{"*"}},
						{Hosts: []string{"*"}},
					},
				}, 8080,
			},
			true,
		},
		{
			"Same host/port in two servers with *:<port>",
			args{
				&config.Couper{
					Servers: []*config.Server{
						{Hosts: []string{"*:8080"}},
						{Hosts: []string{"*:8080"}},
					},
				}, 8080,
			},
			true,
		},
		{
			"Same host/port in two servers with example.com",
			args{
				&config.Couper{
					Servers: []*config.Server{
						{Hosts: []string{"example.com", "couper.io"}},
						{Hosts: []string{"example.com", "couper.io"}},
					},
				}, 8080,
			},
			true,
		},
		{
			"Same host/port in two servers with example.com:<port>",
			args{
				&config.Couper{
					Servers: []*config.Server{
						{Hosts: []string{"example.com:9090"}},
						{Hosts: []string{"example.com:9090"}},
					},
				}, 8080,
			},
			true,
		},
		{
			"Same port w/ different host in two servers",
			args{
				&config.Couper{
					Servers: []*config.Server{
						{Hosts: []string{"*", "example.com:9090"}},
						{Hosts: []string{"couper.io:9090"}},
					},
				}, 8080,
			},
			false,
		},
		{
			"Host is mandatory for multiple servers",
			args{
				&config.Couper{
					Servers: []*config.Server{
						{Hosts: []string{"*"}},
						{},
					},
				}, 8080,
			},
			true,
		},
		{
			"Host is optional for single server",
			args{
				&config.Couper{
					Servers: []*config.Server{
						{},
					},
				}, 8080,
			},
			false,
		},
		{
			"Invalid host format",
			args{
				&config.Couper{
					Servers: []*config.Server{
						{Hosts: []string{"_"}},
					},
				}, 8080,
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			tt.args.conf.Context = eval.NewContext(nil, nil)
			tt.args.conf.Settings = &config.DefaultSettings

			logger, _ := test.NewLogger()
			log := logger.WithContext(context.Background())
			quitCh := make(chan struct{})
			defer close(quitCh)
			memStore := cache.New(log, quitCh)

			if _, err := NewServerConfiguration(tt.args.conf, log, memStore); (err != nil) != tt.wantErr {
				subT.Errorf("validatePortHosts() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServer_GetCORS(t *testing.T) {
	parentCORS := &config.CORS{MaxAge: "123"}
	parent := &config.Server{
		CORS: parentCORS,
	}
	currCORS := &config.CORS{MaxAge: "321"}
	curr := &config.Server{
		CORS: currCORS,
	}

	if got := whichCORS(parent, &config.Server{}); got != parentCORS {
		t.Errorf("Unexpected CORS given: %#v", got)
	}

	if got := whichCORS(parent, curr); got != currCORS {
		t.Errorf("Unexpected CORS given: %#v", got)
	}

	currCORS.Disable = true

	if got := whichCORS(parent, curr); got != nil {
		t.Errorf("Unexpected CORS given: %#v", got)
	}
}

func TestServer_ParseDuration(t *testing.T) {
	var target time.Duration

	if err := parseDuration("non-duration", &target); err == nil {
		t.Error("Unexpected NIL-error given")
	}
	if target != 0 {
		t.Errorf("Unexpected duration given: %#v", target)
	}

	if err := parseDuration("1ms", &target); err != nil {
		t.Errorf("Unexpected error given: %#v", err)
	}
	if target != 1000000 {
		t.Errorf("Unexpected duration given: %#v", target)
	}
}

func TestServer_ParseBodyLimit(t *testing.T) {
	i, err := parseBodyLimit("non-size")
	if err == nil {
		t.Error("Unexpected NIL-error given")
	}
	if i != -1 {
		t.Errorf("Unexpected size given: %#v", i)
	}

	i, err = parseBodyLimit("")
	if err != nil {
		t.Error("Unexpected error given")
	}
	if i != 64000000 {
		t.Errorf("Unexpected size given: %#v", i)
	}

	i, err = parseBodyLimit("1K")
	if err != nil {
		t.Error("Unexpected error given")
	}
	if i != 1000 {
		t.Errorf("Unexpected size given: %#v", i)
	}
}

func TestServer_NewAC(t *testing.T) {
	srvConf := &config.Server{
		AccessControl:        []string{"s1", "s2"},
		DisableAccessControl: []string{"s1"},
	}
	apiConf := &config.API{
		AccessControl:        []string{"a1", "a2"},
		DisableAccessControl: []string{"a1"},
	}

	got := newAC(srvConf, nil)
	exp := config.AccessControl{
		AccessControl:        []string{"s1", "s2"},
		DisableAccessControl: []string{"s1"},
	}
	if !reflect.DeepEqual(got, exp) {
		t.Errorf("want\n%#v\ngot\n%#v", exp, got)
	}

	got = newAC(srvConf, apiConf)
	exp = config.AccessControl{
		AccessControl:        []string{"s1", "s2", "a1", "a2"},
		DisableAccessControl: []string{"s1", "a1"},
	}
	if !reflect.DeepEqual(got, exp) {
		t.Errorf("want\n%#v\ngot\n%#v", exp, got)
	}
}
