package handler_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/runtime/server"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler"
)

func TestSpa_ServeHTTP(t *testing.T) {
	wd, werr := os.Getwd()
	if werr != nil {
		t.Error(werr)
	}

	appHtmlContent, ferr := os.ReadFile(path.Join(wd, "testdata/spa/app.html"))
	if ferr != nil {
		t.Fatal(ferr)
	}

	tests := []struct {
		cfg             *config.Spa
		req             *http.Request
		expectedContent []byte
		expectedCode    int
		expectedErr     string
	}{
		{&config.Spa{Name: "serve bootstrap file", BootstrapFile: path.Join(wd, "testdata/spa/app.html")}, httptest.NewRequest(http.MethodGet, "/", nil), appHtmlContent, http.StatusOK, ""},
		{&config.Spa{Name: "serve no bootstrap file", BootstrapFile: path.Join(wd, "testdata/spa/not_exist.html")}, httptest.NewRequest(http.MethodGet, "/", nil), nil, http.StatusNotFound, fmt.Sprintf("open %s/testdata/spa/not_exist.html: no such file or directory", wd)},
		{&config.Spa{Name: "serve bootstrap dir", BootstrapFile: path.Join(wd, "testdata/spa")}, httptest.NewRequest(http.MethodGet, "/", nil), nil, http.StatusInternalServerError, ""},
		{&config.Spa{Name: "serve bootstrap file /w simple-data", BootstrapFile: path.Join(wd, "testdata/spa/app_bs_data.html"),
			BootstrapData: hcl.StaticExpr(cty.StringVal("no-object"), hcl.Range{})}, httptest.NewRequest(http.MethodGet, "/", nil),
			nil, http.StatusInternalServerError, ":0,0-0: configuration error; bootstrap_data must be an object type"},
		{&config.Spa{Name: "serve bootstrap file /w obj-data", BootstrapFile: path.Join(wd, "testdata/spa/app_bs_data.html"),
			BootstrapData: hcl.StaticExpr(cty.ObjectVal(map[string]cty.Value{"prop": cty.StringVal("value")}), hcl.Range{})}, httptest.NewRequest(http.MethodGet, "/", nil),
			[]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>App</title>
    <script>const conf = {"prop":"value"};</script>
</head>
<body>
App with __BOOTSTRAP_DATA__.
</body>
</html>
`), http.StatusOK, ""},
		{&config.Spa{Name: "serve bootstrap file /w html obj-data", BootstrapFile: path.Join(wd, "testdata/spa/app_bs_data.html"),
			BootstrapData: hcl.StaticExpr(cty.ObjectVal(map[string]cty.Value{"prop": cty.StringVal("</script>")}), hcl.Range{})}, httptest.NewRequest(http.MethodGet, "/", nil),
			[]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>App</title>
    <script>const conf = {"prop":"\u003c/script\u003e"};</script>
</head>
<body>
App with __BOOTSTRAP_DATA__.
</body>
</html>
`), http.StatusOK, ""},
	}
	for _, tt := range tests {
		t.Run(tt.cfg.Name, func(subT *testing.T) {
			opts, _ := server.NewServerOptions(&config.Server{}, nil)
			s, err := handler.NewSpa(eval.NewDefaultContext().HCLContext(), tt.cfg, opts, nil)
			if err != nil {
				if tt.expectedErr != "" && err.Error() == tt.expectedErr {
					return
				}
				subT.Fatal(err)
			}

			res := httptest.NewRecorder()
			s.ServeHTTP(res, tt.req)

			if !res.Flushed {
				res.Flush()
			}

			if res.Code != tt.expectedCode {
				subT.Errorf("Expected status code %d, got: %d", tt.expectedCode, res.Code)
			}

			if tt.expectedContent != nil {
				if diff := cmp.Diff(tt.expectedContent, []byte(res.Body.String())); diff != "" {
					t.Error(diff)
				}
			}
		})
	}
}
