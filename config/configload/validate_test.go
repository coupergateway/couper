package configload

import (
	"context"
	"testing"

	"github.com/hashicorp/hcl/v2"
	logrustest "github.com/sirupsen/logrus/hooks/test"

	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config/runtime"
)

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
			`server {
			   hosts = ["*:8888"]
			 }
			 server {
			   hosts = ["*:9999"]
			 }`,
			"",
		},
		{
			"labelled and unlabelled servers",
			`server {
			   hosts = ["*:8888"]
			 }
			 server "test" {
			   hosts = ["*:9999"]
			 }
			 `,
			"",
		},
		{
			"duplicate server labels",
			`server "test" {
			   hosts = ["*:8888"]
			 }
			 server "test" {
			   hosts = ["*:9999"]
			 }`,
			"",
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
			   api {
			     base_path = "/foo"
			   }
			   api "bar" {
			     base_path = "/bar"
			   }
			 }`,
			"",
		},
		{
			"duplicate api labels",
			`server "test" {
			   api "foo" {
			     base_path = "/foo"
			   }
			   api "foo" {
			     base_path = "/bar"
			   }
			 }`,
			``,
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
				tmpStoreCh := make(chan struct{})
				tmpMemStore := cache.New(log, tmpStoreCh)
				defer close(tmpStoreCh)
				_, err = runtime.NewServerConfiguration(conf, log, tmpMemStore)
			}

			var errMsg string
			if err != nil {
				errMsg = err.Error()
			}

			if tt.error != errMsg {
				subT.Errorf("%q: Unexpected configuration error:\n\tWant: %q\n\tGot:  %q", tt.name, tt.error, errMsg)
			}
		})
	}
}
