package server_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/runtime/server"
	"github.com/avenga/couper/errors"
)

func TestServer_NewServerOptions_NoConfig(t *testing.T) {
	options, err := server.NewServerOptions(nil, nil)
	if err != nil {
		t.Errorf("Unexpected error given: %#v", err)
	}

	exp := &server.Options{
		APIErrTpls:    map[*config.API]*errors.Template(nil),
		FilesErrTpl:   errors.DefaultHTML,
		ServerErrTpl:  errors.DefaultHTML,
		APIBasePaths:  map[*config.API]string(nil),
		FilesBasePath: "",
		SPABasePath:   "",
		SrvBasePath:   "",
		ServerName:    "",
	}
	if !reflect.DeepEqual(options, exp) {
		t.Errorf("want\n%#v\ngot\n%#v", exp, options)
	}
}

func TestServer_NewServerOptions_EmptyConfig(t *testing.T) {
	conf := &config.Server{}

	options, err := server.NewServerOptions(conf, nil)
	if err != nil {
		t.Errorf("Unexpected error given: %#v", err)
	}

	exp := &server.Options{
		APIErrTpls:    map[*config.API]*errors.Template(nil),
		FilesErrTpl:   errors.DefaultHTML,
		ServerErrTpl:  errors.DefaultHTML,
		APIBasePaths:  map[*config.API]string(nil),
		FilesBasePath: "",
		SPABasePath:   "",
		SrvBasePath:   "/",
		ServerName:    "",
	}
	if !reflect.DeepEqual(options, exp) {
		t.Errorf("want\n%#v\ngot\n%#v", exp, options)
	}
}

func TestServer_NewServerOptions_ConfigWithPaths(t *testing.T) {
	api1 := &config.API{
		BasePath: "/api/v1",
	}
	api2 := &config.API{
		BasePath: "/api/v2",
	}

	abps := make(map[*config.API]string)
	abps[api1] = "/server/api/v1"
	abps[api2] = "/server/api/v2"

	aets := make(map[*config.API]*errors.Template)
	aets[api1] = errors.DefaultJSON
	aets[api2] = errors.DefaultJSON

	conf := &config.Server{
		BasePath: "/server",
		Name:     "ServerName",

		Files: &config.Files{
			BasePath: "/files",
		},
		Spa: &config.Spa{
			BasePath: "/spa",
		},
		APIs: config.APIs{
			api1, api2,
		},
	}

	options, err := server.NewServerOptions(conf, nil)
	if err != nil {
		t.Errorf("Unexpected error given: %#v", err)
	}

	exp := &server.Options{
		APIErrTpls:    aets,
		FilesErrTpl:   errors.DefaultHTML,
		ServerErrTpl:  errors.DefaultHTML,
		APIBasePaths:  abps,
		FilesBasePath: "/server/files",
		SPABasePath:   "/server/spa",
		SrvBasePath:   "/server",
		ServerName:    "ServerName",
	}
	if !reflect.DeepEqual(options, exp) {
		t.Errorf("want\n%#v\ngot\n%#v", exp, options)
	}
}

func TestServer_NewServerOptions_MissingErrTplFile(t *testing.T) {
	for _, testcase := range []*config.Server{{
		ErrorFile: "not-there",
	}, {
		Files: &config.Files{
			ErrorFile: "not-there",
		},
	}, {
		APIs: config.APIs{
			{ErrorFile: "not-there"},
		},
	},
	} {
		_, err := server.NewServerOptions(testcase, nil)
		if err == nil || !strings.Contains(err.Error(), "no such file or directory") {
			t.Errorf("Unexpected error given: %#v", err)
		}
	}
}

func TestServer_NewServerOptions_ConfigWithErrTpl_Valid(t *testing.T) {
	conf := &config.Server{
		ErrorFile: "testdata/error.file",
		Files: &config.Files{
			ErrorFile: "testdata/error.file",
		},
		APIs: config.APIs{
			{ErrorFile: "testdata/error.file"},
		},
	}

	_, err := server.NewServerOptions(conf, nil)
	if err != nil {
		t.Error("Unexpected error given")
	}
}
