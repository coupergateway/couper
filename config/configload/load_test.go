package configload

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/runtime"
	logrustest "github.com/sirupsen/logrus/hooks/test"
)

func Test_refineEndpoints_noPattern(t *testing.T) {
	err := refineEndpoints(nil, config.Endpoints{{Pattern: ""}}, true)
	if err == nil || !strings.HasSuffix(err.Error(), "endpoint: missing path pattern; ") {
		t.Errorf("refineEndpoints() error = %v, wantErr: endpoint: missing path pattern ", err)
	}
}

func Test_VerifyBodyAttributes(t *testing.T) {
	type testCase struct {
		name    string
		content *hcl.BodyContent
		expErr  bool
	}

	for _, tc := range []testCase{
		{"without any body", &hcl.BodyContent{}, false},
		{"body", &hcl.BodyContent{Attributes: map[string]*hcl.Attribute{"body": {Name: "body"}}}, false},
		{"json_body", &hcl.BodyContent{Attributes: map[string]*hcl.Attribute{"json_body": {Name: "json_body"}}}, false},
		{"form_body", &hcl.BodyContent{Attributes: map[string]*hcl.Attribute{"form_body": {Name: "form_body"}}}, false},
		{"body/form_body", &hcl.BodyContent{Attributes: map[string]*hcl.Attribute{
			"body":      {Name: "body"},
			"form_body": {Name: "form_body"},
		}}, true},
		{"body/json_body", &hcl.BodyContent{Attributes: map[string]*hcl.Attribute{
			"body":      {Name: "body"},
			"json_body": {Name: "json_body"},
		}}, true},
		{"form_body/json_body", &hcl.BodyContent{Attributes: map[string]*hcl.Attribute{
			"form_body": {Name: "form_body"},
			"json_body": {Name: "json_body"},
		}}, true},
		{"body/json_body/form_body", &hcl.BodyContent{Attributes: map[string]*hcl.Attribute{
			"body":      {Name: "body"},
			"json_body": {Name: "json_body"},
			"form_body": {Name: "form_body"},
		}}, true},
	} {
		if err := verifyBodyAttributes(tc.content); !tc.expErr && err != nil {
			t.Errorf("Want no error, got: %v", err)
		}
	}
}

func TestLabels(t *testing.T) {
	tests := []struct {
		name  string
		hcl   string
		error string
	}{
		{
			"missing server",
			`definitions {}`,
			"configuration error: missing 'server' block",
		},
		{
			"server w/o label",
			`server {}`,
			"",
		},
		{
			"multiple servers w/o label",
			`server {}
			 server {}`,
			"configuration error: only one anonymous server allowed",
		},
		{
			"labelled and unlabelled servers",
			`server {}
			 server "test" {}
			 `,
			"couper.hcl:1,8-9: Missing name for server; All server blocks must have 1 labels (name).",
		},
		{
			"duplicate server labels",
			`server "test" {}
			 server "test" {}`,
			`configuration error: duplicate server name "test"`,
		},
		{
			"unique server label",
			`server "test" {
			   hosts = ["*:8888"]
			 }
			 server "foo" {
				hosts = ["*:9999"]
			 }
			 definitions {
			   basic_auth "test" {}
			 }`,
			"",
		},
		{
			"anonymous api block",
			`server "test" {
			   api {}
			 }`,
			"",
		},
		{
			"multiple anonymous api blocks",
			`server "test" {
			   api {
			     base_path = "/foo"
			   }
			   api {
			     base_path = "/bar"
			   }
			 }`,
			"",
		},
		{
			"mixed labelled api blocks",
			`server "test" {
			   api {}
			   api "foo" {}
			 }`,
			"couper.hcl:2,11-12: Missing name for api; All api blocks must have 1 labels (name).",
		},
		{
			"duplicate api labels",
			`server "test" {
			   api "foo" {}
			   api "foo" {}
			 }`,
			`configuration error: duplicate api name "test"`,
		},
		{
			"uniquely labelled api blocks per server",
			`server "foo" {
			   hosts = ["*:8888"]
			   api "foo" {
			     base_path = "/foo"
		       }
			   api "bar" {
			     base_path = "/bar"
			   }
			 }
			 server "bar" {
			   hosts = ["*:9999"]
			   api "foo" {
			     base_path = "/foo"
		       }
			   api "bar" {
			     base_path = "/bar"
			   }
			 }`,
			"",
		},
	}

	logger, _ := logrustest.NewNullLogger()
	log := logger.WithContext(context.TODO())

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			conf, err := LoadBytes([]byte(tt.hcl), "couper.hcl")
			if conf != nil {
				_, err = runtime.NewServerConfiguration(conf, log, nil)
			}

			var error = ""
			if err != nil {
				error = err.Error()
			}

			if tt.error != error {
				subT.Errorf("%q: Unexpected configuration error:\n\tWant: %q\n\tGot:  %q", tt.name, tt.error, error)
			}
		})
	}
}
