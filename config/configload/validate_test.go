package configload

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config/runtime"
	"github.com/avenga/couper/errors"
	logrustest "github.com/sirupsen/logrus/hooks/test"
)

func Test_VerifyBodyAttributes(t *testing.T) {
	type testCase struct {
		name   string
		body   *hclsyntax.Body
		expErr bool
	}

	for _, tc := range []testCase{
		{"without any body attributes", &hclsyntax.Body{}, false},
		{"body", &hclsyntax.Body{Attributes: map[string]*hclsyntax.Attribute{"body": {Name: "body"}}}, false},
		{"json_body", &hclsyntax.Body{Attributes: map[string]*hclsyntax.Attribute{"json_body": {Name: "json_body"}}}, false},
		{"form_body", &hclsyntax.Body{Attributes: map[string]*hclsyntax.Attribute{"form_body": {Name: "form_body"}}}, false},
		{"body/form_body", &hclsyntax.Body{Attributes: map[string]*hclsyntax.Attribute{
			"body":      {Name: "body"},
			"form_body": {Name: "form_body"},
		}}, true},
		{"body/json_body", &hclsyntax.Body{Attributes: map[string]*hclsyntax.Attribute{
			"body":      {Name: "body"},
			"json_body": {Name: "json_body"},
		}}, true},
		{"form_body/json_body", &hclsyntax.Body{Attributes: map[string]*hclsyntax.Attribute{
			"form_body": {Name: "form_body"},
			"json_body": {Name: "json_body"},
		}}, true},
		{"body/json_body/form_body", &hclsyntax.Body{Attributes: map[string]*hclsyntax.Attribute{
			"body":      {Name: "body"},
			"json_body": {Name: "json_body"},
			"form_body": {Name: "form_body"},
		}}, true},
	} {
		if err := verifyBodyAttributes(request, tc.body); !tc.expErr && err != nil {
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
			"multiple anonymous api blocks, different base_path",
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
			"multiple anonymous api blocks (sharing base_path)",
			`server "test" {
			   api {}
			   api {}
			 }`,
			"",
		},
		{
			"api blocks sharing base_path",
			`server "test" {
			   api {
			     base_path = "/foo"
			   }
			   api {
			     base_path = "/foo"
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
				defer close(tmpStoreCh)

				ctx, cancel := context.WithCancel(conf.Context)
				conf.Context = ctx
				defer cancel()

				_, err = runtime.NewServerConfiguration(conf, log, cache.New(log, tmpStoreCh))
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

func Test_validateBody(t *testing.T) {
	tests := []struct {
		name  string
		hcl   string
		error string
	}{
		{
			"missing backend label",
			`server {}
			 definitions {
			   backend {
			   }
			 }`,
			"couper.hcl:3,15-16: missing label; ",
		},
		{
			"empty backend label",
			`server {}
			 definitions {
			   backend "" {
			   }
			 }`,
			"couper.hcl:3,15-17: label is empty; ",
		},
		{
			"whitespace backend label",
			`server {}
			 definitions {
			   backend " 	" {
			   }
			 }`,
			"couper.hcl:3,15-19: label is empty; ",
		},
		{
			"invalid backend label",
			`server {}
			 definitions {
			   backend "foo bar" {
			   }
			 }`,
			"couper.hcl:3,15-24: label contains invalid character(s), allowed are 'a-z', 'A-Z', '0-9' and '_'; ",
		},
		{
			"anonymous_* backend label",
			`server {}
			 definitions {
			   backend "anonymous_foo" {
			   }
			 }`,
			"couper.hcl:3,15-30: backend label must not start with 'anonymous_'; ",
		},
		{
			"duplicate backend labels",
			`server {}
			 definitions {
			   backend "foo" {
			   }
			   backend "foo" {
			   }
			 }`,
			"couper.hcl:5,15-20: backend labels must be unique; ",
		},
		{
			"backend disable cert validation with tls block and server_ca_certificate",
			`server {}
			 definitions {
			   backend "foo" {
					disable_certificate_validation = false # value does not matter
					tls {
						server_ca_certificate = "asdf"
					}
			   }
			 }`,
			"couper.hcl:5,6-7,7: configured 'disable_certificate_validation' along with 'server_ca_certificate' attribute; ",
		},
		{
			"backend disable cert validation with tls block and server_ca_certificate_file",
			`server {}
			 definitions {
			   backend "foo" {
					disable_certificate_validation = true # value does not matter
					tls {
						server_ca_certificate_file = "asdf.crt"
					}
			   }
			 }`,
			"couper.hcl:5,6-7,7: configured 'disable_certificate_validation' along with 'server_ca_certificate_file' attribute; ",
		},
		{
			"backend disable cert validation with tls block and w/o server_ca_certificate*",
			`server {}
			 definitions {
			   backend "foo" {
					disable_certificate_validation = true # value does not matter
					tls {}
			   }
			 }`,
			"",
		},
		{
			"duplicate proxy labels",
			`server {}
			 definitions {
			   proxy "foo" {
			   }
			   proxy "foo" {
			   }
			 }`,
			"couper.hcl:5,13-18: proxy labels must be unique; ",
		},
		{
			"missing basic_auth label",
			`server {}
			 definitions {
			   basic_auth {
			   }
			 }`,
			"couper.hcl:3,18-19: missing label; ",
		},
		{
			"missing beta_oauth2 label",
			`server {}
			 definitions {
			   beta_oauth2 {
			   }
			 }`,
			"couper.hcl:3,19-20: missing label; ",
		},
		{
			"missing jwt label",
			`server {}
			 definitions {
			   jwt {
			   }
			 }`,
			"couper.hcl:3,11-12: missing label; ",
		},
		{
			"missing oidc label",
			`server {}
			 definitions {
			   oidc {
			   }
			 }`,
			"couper.hcl:3,12-13: missing label; ",
		},
		{
			"missing saml label",
			`server {}
			 definitions {
			   saml {
			   }
			 }`,
			"couper.hcl:3,12-13: missing label; ",
		},
		{
			"basic_auth with empty label",
			`server {}
			 definitions {
			   basic_auth "" {
			   }
			 }`,
			"couper.hcl:3,18-20: accessControl requires a label; ",
		},
		{
			"beta_oauth2 with empty label",
			`server {}
			 definitions {
			   beta_oauth2 "" {
			   }
			 }`,
			"couper.hcl:3,19-21: accessControl requires a label; ",
		},
		{
			"jwt with empty label",
			`server {}
			 definitions {
			   jwt "" {
			   }
			 }`,
			"couper.hcl:3,11-13: accessControl requires a label; ",
		},
		{
			"oidc with empty label",
			`server {}
			 definitions {
			   oidc "" {
			   }
			 }`,
			"couper.hcl:3,12-14: accessControl requires a label; ",
		},
		{
			"saml with empty label",
			`server {}
			 definitions {
			   saml "" {
			   }
			 }`,
			"couper.hcl:3,12-14: accessControl requires a label; ",
		},
		{
			"basic_auth with whitespace label",
			`server {}
			 definitions {
			   basic_auth " 	" {
			   }
			 }`,
			"couper.hcl:3,18-22: accessControl requires a label; ",
		},
		{
			"beta_oauth2 with whitespace label",
			`server {}
			 definitions {
			   beta_oauth2 " 	" {
			   }
			 }`,
			"couper.hcl:3,19-23: accessControl requires a label; ",
		},
		{
			"jwt with whitespace label",
			`server {}
			 definitions {
			   jwt " 	" {
			   }
			 }`,
			"couper.hcl:3,11-15: accessControl requires a label; ",
		},
		{
			"oidc with whitespace label",
			`server {}
			 definitions {
			   oidc " 	" {
			   }
			 }`,
			"couper.hcl:3,12-16: accessControl requires a label; ",
		},
		{
			"saml with whitespace label",
			`server {}
			 definitions {
			   saml " 	" {
			   }
			 }`,
			"couper.hcl:3,12-16: accessControl requires a label; ",
		},
		{
			"basic_auth reserved label granted_permissions",
			`server {}
			 definitions {
			   basic_auth "granted_permissions" {
			   }
			 }`,
			"couper.hcl:3,18-39: accessControl uses reserved name as label; ",
		},
		{
			"basic_auth reserved label required_permission",
			`server {}
			 definitions {
			   basic_auth "required_permission" {
			   }
			 }`,
			"couper.hcl:3,18-39: accessControl uses reserved name as label; ",
		},
		{
			"beta_oauth2 reserved label granted_permissions",
			`server {}
			 definitions {
			   beta_oauth2 "granted_permissions" {
			   }
			 }`,
			"couper.hcl:3,19-40: accessControl uses reserved name as label; ",
		},
		{
			"beta_oauth2 reserved label required_permission",
			`server {}
			 definitions {
			   beta_oauth2 "required_permission" {
			   }
			 }`,
			"couper.hcl:3,19-40: accessControl uses reserved name as label; ",
		},
		{
			"jwt reserved label granted_permissions",
			`server {}
			 definitions {
			   jwt "granted_permissions" {
			   }
			 }`,
			"couper.hcl:3,11-32: accessControl uses reserved name as label; ",
		},
		{
			"jwt reserved label required_permission",
			`server {}
			 definitions {
			   jwt "required_permission" {
			   }
			 }`,
			"couper.hcl:3,11-32: accessControl uses reserved name as label; ",
		},
		{
			"oidc reserved label granted_permissions",
			`server {}
			 definitions {
			   oidc "granted_permissions" {
			   }
			 }`,
			"couper.hcl:3,12-33: accessControl uses reserved name as label; ",
		},
		{
			"oidc reserved label required_permission",
			`server {}
			 definitions {
			   oidc "required_permission" {
			   }
			 }`,
			"couper.hcl:3,12-33: accessControl uses reserved name as label; ",
		},
		{
			"saml reserved label granted_permissions",
			`server {}
			 definitions {
			   saml "granted_permissions" {
			   }
			 }`,
			"couper.hcl:3,12-33: accessControl uses reserved name as label; ",
		},
		{
			"saml reserved label required_permission",
			`server {}
			 definitions {
			   saml "required_permission" {
			   }
			 }`,
			"couper.hcl:3,12-33: accessControl uses reserved name as label; ",
		},
		{
			"duplicate AC labels 1",
			`server {}
			 definitions {
			   basic_auth "foo" {
			   }
			   beta_oauth2 "foo" {
			   }
			 }`,
			"couper.hcl:5,19-24: AC labels must be unique; ",
		},
		{
			"duplicate AC labels 2",
			`server {}
			 definitions {
			   beta_oauth2 "foo" {
			   }
			   jwt "foo" {
			   }
			 }`,
			"couper.hcl:5,11-16: AC labels must be unique; ",
		},
		{
			"duplicate AC labels 3",
			`server {}
			 definitions {
			   jwt "foo" {
			   }
			   oidc "foo" {
			   }
			 }`,
			"couper.hcl:5,12-17: AC labels must be unique; ",
		},
		{
			"duplicate AC labels 4",
			`server {}
			 definitions {
			   oidc "foo" {
			   }
			   saml "foo" {
			   }
			 }`,
			"couper.hcl:5,12-17: AC labels must be unique; ",
		},
		{
			"duplicate AC labels 5",
			`server {}
			 definitions {
			   saml "foo" {
			   }
			   basic_auth "foo" {
			   }
			 }`,
			"couper.hcl:5,18-23: AC labels must be unique; ",
		},
		{
			"duplicate signing profile labels 1",
			`server {}
			 definitions {
			   jwt "foo" {
			     signing_ttl = "1m"
			   }
			   jwt_signing_profile "foo" {
			   }
			 }`,
			"couper.hcl:6,27-32: JWT signing profile labels must be unique; ",
		},
		{
			"jwt not used as signing profile",
			`server {}
			 definitions {
			   jwt_signing_profile "foo" {
			     signature_algorithm = "HS256"
			     key = "asdf"
			     ttl = "1m"
			   }
			   jwt "foo" {
			     signature_algorithm = "HS256"
			     key = "sdfg"
			   }
			 }`,
			"",
		},
		{
			"duplicate signing profile labels 2",
			`server {}
			 definitions {
			   jwt_signing_profile "foo" {
			   }
			   jwt "foo" {
			     signing_ttl = "1m"
			   }
			 }`,
			"couper.hcl:5,11-16: JWT signing profile labels must be unique; ",
		},
		{
			"same label for backend and AC",
			`server {}
			 definitions {
			   backend "foo" {
			   }
			   basic_auth "foo" {
			   }
			 }`,
			"",
		},
		{
			"invalid server base_path pattern single dot 1",
			`server {
			   base_path = "./s"
			 }`,
			`couper.hcl:2,20-23: base_path must not contain "." or ".." segments; `,
		},
		{
			"invalid server base_path pattern single dot 2",
			`server {
			   base_path = "/./s"
			 }`,
			`couper.hcl:2,20-24: base_path must not contain "." or ".." segments; `,
		},
		{
			"invalid server base_path pattern single dot 3",
			`server {
			   base_path = "/s/./s"
			 }`,
			`couper.hcl:2,20-26: base_path must not contain "." or ".." segments; `,
		},
		{
			"invalid server base_path pattern single dot 4",
			`server {
			   base_path = "/s/."
			 }`,
			`couper.hcl:2,20-24: base_path must not contain "." or ".." segments; `,
		},
		{
			"invalid server base_path pattern double dot 1",
			`server {
			   base_path = "../s"
			 }`,
			`couper.hcl:2,20-24: base_path must not contain "." or ".." segments; `,
		},
		{
			"invalid server base_path pattern double dot 2",
			`server {
			   base_path = "/../s"
			 }`,
			`couper.hcl:2,20-25: base_path must not contain "." or ".." segments; `,
		},
		{
			"invalid server base_path pattern double dot 3",
			`server {
			   base_path = "/s/../s"
			 }`,
			`couper.hcl:2,20-27: base_path must not contain "." or ".." segments; `,
		},
		{
			"invalid server base_path pattern double dot 4",
			`server {
			   base_path = "/s/.."
			 }`,
			`couper.hcl:2,20-25: base_path must not contain "." or ".." segments; `,
		},
		{
			"invalid api base_path pattern single dot 1",
			`server {
			   api {
			     base_path = "./s"
			   }
			 }`,
			`couper.hcl:3,22-25: base_path must not contain "." or ".." segments; `,
		},
		{
			"invalid api base_path pattern single dot 2",
			`server {
			   api {
			     base_path = "/./s"
			   }
			 }`,
			`couper.hcl:3,22-26: base_path must not contain "." or ".." segments; `,
		},
		{
			"invalid api base_path pattern single dot 3",
			`server {
			   api {
			     base_path = "/s/./s"
			   }
			 }`,
			`couper.hcl:3,22-28: base_path must not contain "." or ".." segments; `,
		},
		{
			"invalid api base_path pattern single dot 4",
			`server {
			   api {
			     base_path = "/s/."
			   }
			 }`,
			`couper.hcl:3,22-26: base_path must not contain "." or ".." segments; `,
		},
		{
			"invalid api base_path pattern double dot 1",
			`server {
			   api {
			     base_path = "../s"
			   }
			 }`,
			`couper.hcl:3,22-26: base_path must not contain "." or ".." segments; `,
		},
		{
			"invalid api base_path pattern double dot 2",
			`server {
			   api {
			     base_path = "/../s"
			   }
			 }`,
			`couper.hcl:3,22-27: base_path must not contain "." or ".." segments; `,
		},
		{
			"invalid api base_path pattern double dot 3",
			`server {
			   api {
			     base_path = "/s/../s"
			   }
			 }`,
			`couper.hcl:3,22-29: base_path must not contain "." or ".." segments; `,
		},
		{
			"invalid api base_path pattern double dot 4",
			`server {
			   api {
			     base_path = "/s/.."
			   }
			 }`,
			`couper.hcl:3,22-27: base_path must not contain "." or ".." segments; `,
		},
		{
			"invalid endpoint pattern single dot 1",
			`server {
			   api {
			     endpoint "./foo" {
			       response {
			         body = "1"
			       }
			     }
			   }
			 }`,
			`couper.hcl:3,18-25: endpoint path pattern must start with "/"; `,
		},
		{
			"invalid endpoint pattern single dot 2",
			`server {
			   api {
			     endpoint "/foo/./bar" {
			       response {
			         body = "1"
			       }
			     }
			   }
			 }`,
			`couper.hcl:3,18-30: endpoint path pattern must not contain "." or ".." segments; `,
		},
		{
			"invalid endpoint pattern single dot 3",
			`server {
			   api {
			     endpoint "/foo/." {
			       response {
			         body = "1"
			       }
			     }
			   }
			 }`,
			`couper.hcl:3,18-26: endpoint path pattern must not contain "." or ".." segments; `,
		},
		{
			"invalid endpoint pattern double dot 1",
			`server {
			   api {
			     endpoint "../foo" {
			       response {
			         body = "1"
			       }
			     }
			   }
			 }`,
			`couper.hcl:3,18-26: endpoint path pattern must start with "/"; `,
		},
		{
			"invalid endpoint pattern double dot 2",
			`server {
			   api {
			     endpoint "/foo/../bar" {
			       response {
			         body = "1"
			       }
			     }
			   }
			 }`,
			`couper.hcl:3,18-31: endpoint path pattern must not contain "." or ".." segments; `,
		},
		{
			"invalid endpoint pattern double dot 3",
			`server {
			   api {
			     endpoint "/foo/.." {
			       response {
			         body = "1"
			       }
			     }
			   }
			 }`,
			`couper.hcl:3,18-27: endpoint path pattern must not contain "." or ".." segments; `,
		},
		{
			"duplicate endpoint pattern 1",
			`server {
			   base_path = "/s"
			   api {
			     endpoint "/" {
			       response {
			         body = "1"
			       }
			     }
			     endpoint "/" {
			       response {
			         body = "2"
			       }
			     }
			   }
			 }`,
			"couper.hcl:9,18-21: duplicate endpoint; ",
		},
		{
			"duplicate endpoint pattern 2",
			`server {
			   base_path = "/s"
			   api {
			     endpoint "/" {
			       response {
			         body = "1"
			       }
			     }
			   }
			   api {
			     endpoint "/" {
			       response {
			         body = "2"
			       }
			     }
			   }
			 }`,
			"couper.hcl:11,18-21: duplicate endpoint; ",
		},
		{
			"duplicate endpoint pattern 3",
			`server {
			   base_path = "/s"
			   api {
			     endpoint "/" {
			       response {
			         body = "1"
			       }
			     }
			   }
			   endpoint "/" {
			     response {
			       body = "2"
			     }
			   }
			 }`,
			"couper.hcl:10,16-19: duplicate endpoint; ",
		},
		{
			"duplicate endpoint pattern 4",
			`server {
			   base_path = "/s"
			   endpoint "/" {
			     response {
			       body = "1"
			     }
			   }
			   api {
			     endpoint "/" {
			       response {
			         body = "2"
			       }
			     }
			   }
			 }`,
			"couper.hcl:9,18-21: duplicate endpoint; ",
		},
		{
			"duplicate endpoint pattern 5",
			`server {
			   base_path = "/s"
			   api {
			     base_path = "/a"
			     endpoint "/b/{c}" {
			       response {
			         body = "1"
			       }
			     }
			   }
			   api {
			     base_path = "/a/b"
			     endpoint "/{d}" {
			       response {
			         body = "2"
			       }
			     }
			   }
			 }`,
			"couper.hcl:13,18-24: duplicate endpoint; ",
		},
		{
			"duplicate endpoint pattern 6",
			`server {
			   endpoint "/a/b" {
			     response {
			       body = "1"
			     }
			   }
			   api {
			     base_path = "/a"
			     endpoint "/b" {
			       response {
			         body = "2"
			       }
			     }
			   }
			 }`,
			"couper.hcl:9,18-22: duplicate endpoint; ",
		},
		{
			"distinct endpoint patterns",
			`server {
			   base_path = "/s"
			   api {
			     base_path = "/a"
			     endpoint "/{b}/c" {
			       response {
			         body = "1"
			       }
			     }
			   }
			   api {
			     base_path = "/a/b"
			     endpoint "/c" {
			       response {
			         body = "2"
			       }
			     }
			   }
			 }`,
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			_, err := LoadBytes([]byte(tt.hcl), "couper.hcl")

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

func Test_validateBody_multiple(t *testing.T) {
	tests := []struct {
		name   string
		hcls   []string
		errors []string
	}{
		{
			"duplicate AC labels",
			[]string{
				`server {}
				 definitions {
				   saml "foo" {
				   }
				 }`,
				`server {}
				 definitions {
				   basic_auth "foo" {
				   }
				 }`,
			},
			[]string{"couper_0.hcl:3,13-18: AC labels must be unique; ", "couper_1.hcl:3,19-24: AC labels must be unique; "},
		},
		{
			"duplicate endpoint patterns",
			[]string{
				`server {
				   endpoint "/a/b" {
				     response {
				       body = "1"
				     }
				   }
				 }`,
				`server {
				   api {
				     base_path = "/a"
				     endpoint "/b" {
				       response {
				         body = "2"
				       }
				     }
				   }
				 }`,
			},
			[]string{"couper_1.hcl:4,19-23: duplicate endpoint; "},
		},
		{
			"duplicate signing profile labels",
			[]string{
				`server {}
				 definitions {
			       jwt "foo" {
			         signing_ttl = "1m"
			       }
				 }`,
				`server {}
				 definitions {
			       jwt_signing_profile "foo" {
			       }
				 }`,
			},
			[]string{"couper_0.hcl:3,15-20: JWT signing profile labels must be unique; ", "couper_1.hcl:3,31-36: JWT signing profile labels must be unique; "},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			var testContents []testContent
			for i, hcl := range tt.hcls {
				testContents = append(testContents, testContent{fmt.Sprintf("couper_%d.hcl", i), []byte(hcl)})
			}

			_, err := loadTestContents(testContents)

			var errMsg string
			if err != nil {
				errMsg = err.Error()
			}

			if !oneOfErrorMsgs(tt.errors, errMsg) {
				subT.Errorf("%q: Unexpected configuration error:\n\tWant one of: %v\n\tGot:         %q", tt.name, tt.errors, errMsg)
			}
		})
	}
}

func oneOfErrorMsgs(msgs []string, errorMsg string) bool {
	for _, msg := range msgs {
		if msg == errorMsg {
			return true
		}
	}
	return false
}

func TestAttributeObjectKeys(t *testing.T) {
	tests := []struct {
		name  string
		hcl   string
		error string
	}{
		{
			"add_request_headers",
			`server {
  api {
    endpoint "/a" {
      add_request_headers = {
        a = "a"
        A = "A"
      }
    }
  }
}`,
			"couper.hcl:4,29-7,8: key in an attribute must be unique: 'a'; Key must be unique for a.",
		},
		{
			"add_response_headers",
			`server {
  api {
    endpoint "/a" {
      add_response_headers = {
        a = "a"
        A = "A"
      }
    }
  }
}`,
			"couper.hcl:4,30-7,8: key in an attribute must be unique: 'a'; Key must be unique for a.",
		},
		{
			"required_permission",
			`server {
  api {
    endpoint "/a" {
      required_permission = {
        get = "a"
        GeT = "A"
      }
    }
  }
}`,
			"couper.hcl:4,29-7,8: key in an attribute must be unique: 'get'; Key must be unique for get.",
		},
		{
			"headers",
			`server {
  api {
    endpoint "/a" {
      request {
        headers = {
          a = "a"
          A = "A"
        }
      }
    }
  }
}`,
			"couper.hcl:5,19-8,10: key in an attribute must be unique: 'a'; Key must be unique for a.",
		},
		{
			"set_request_headers",
			`server {
  api {
    endpoint "/a" {
      set_request_headers = {
        a = "a"
        A = "A"
      }
    }
  }
}`,
			"couper.hcl:4,29-7,8: key in an attribute must be unique: 'a'; Key must be unique for a.",
		},
		{
			"set_response_headers",
			`server {
  api {
    endpoint "/a" {
      set_response_headers = {
        a = "a"
        A = "A"
      }
    }
  }
}`,
			"couper.hcl:4,30-7,8: key in an attribute must be unique: 'a'; Key must be unique for a.",
		},
		{
			"json_body",
			`server {
  api {
    endpoint "/a" {
      request {
        json_body = {
          a = "a"
          A = "A"
        }
      }
    }
  }
}`,
			"",
		},
		{
			"form_body",
			`server {
  api {
    endpoint "/a" {
      request {
        form_body = {
          a = "a"
          A = "A"
        }
      }
    }
  }
}`,
			"",
		},
		{
			"beta_roles_map",
			`server {}
definitions {
  jwt "a" {
    signature_algorithm = "HS256"
    key = "asdf"
    beta_roles_map = {
      a = []
      A = []
	}
  }
}`,
			"",
		},
		{
			"permissions_map",
			`server {}
definitions {
  jwt "a" {
    signature_algorithm = "HS256"
    key = "asdf"
    permissions_map = {
      a = []
      A = []
	}
  }
}`,
			"",
		},
		{
			"claims",
			`server {}
definitions {
  jwt "a" {
    signature_algorithm = "HS256"
    key = "asdf"
    claims = {
      a = "a"
      A = "A"
	}
  }
}`,
			"",
		},
		{
			"custom_log_fields",
			`server {}
definitions {
  jwt "a" {
    signature_algorithm = "HS256"
    key = "asdf"
    custom_log_fields = {
      a = "a"
      A = "A"
	}
  }
}`,
			"",
		},
		{
			"environment_variables",
			`server {}
defaults {
  environment_variables = {
    a = "a"
    A = "A"
  }
}`,
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			_, err := LoadBytes([]byte(tt.hcl), "couper.hcl")

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

func TestPermissionMixed(t *testing.T) {
	tests := []struct {
		name  string
		hcl   string
		error string
	}{
		{
			"mixed config: error",
			`server {
  api "foo" {
    endpoint "/a" {
      required_permission = "a"
      response {}
    }
    endpoint "/b" {
      response {}
    }
  }
}`,
			"configuration error: api with label \"foo\" has endpoint without required permission",
		},
		{
			"no mix: all endpoints with permission set",
			`server {
  api "foo" {
    endpoint "/a" {
      required_permission = "a"
      response {}
    }
    endpoint "/b" {
      required_permission = ""
      response {}
    }
  }
}`,
			"",
		},
		{
			"no mix: permission set by api",
			`server {
  api "foo" {
    required_permission = "api"
    endpoint "/a" {
      required_permission = "a"
      response {}
    }
    endpoint "/b" {
      response {}
    }
  }
}`,
			"",
		},
		{
			"no mix: disable_access_control",
			`server {
  api "foo" {
    endpoint "/a" {
      required_permission = "a"
      response {}
    }
    endpoint "/b" {
      disable_access_control = ["foo"]
      response {}
    }
  }
}
definitions {
  basic_auth "foo" {
    password = "asdf"
  }
}`,
			"",
		},
		{
			"no mix: separate apis",
			`server {
  api "foo" {
    endpoint "/a" {
      required_permission = "a"
      response {}
    }
  }
  api "bar" {
    endpoint "/b" {
      response {}
    }
  }
}`,
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			_, err := LoadBytes([]byte(tt.hcl), "couper.hcl")

			var errMsg string
			if err != nil {
				gErr, _ := err.(errors.GoError)
				errMsg = gErr.LogError()
			}

			if tt.error != errMsg {
				subT.Errorf("%q: Unexpected configuration error:\n\tWant: %q\n\tGot:  %q", tt.name, tt.error, errMsg)
			}
		})
	}
}

func TestPathAttr(t *testing.T) {
	tests := []struct {
		name  string
		hcl   string
		error string
	}{
		{
			"path in endpoint: error",
			`server {
  endpoint "/**" {
    path = "/a/**"
  }
}`,
			"couper.hcl:3,5-9: Unsupported argument; An argument named \"path\" is not expected here. Use the \"path\" attribute in a backend block instead.",
		},
		{
			"path in proxy: error",
			`server {
  endpoint "/**" {
    proxy {
      path = "/a/**"
    }
  }
}`,
			"couper.hcl:4,7-11: Unsupported argument; An argument named \"path\" is not expected here. Use the \"path\" attribute in a backend block instead.",
		},
		{
			"path in referenced backend: ok",
			`server {
  endpoint "/**" {
    proxy {
      backend = "a"
    }
  }
}
definitions {
  backend "a" {
    path = "/a/**"
  }
}`,
			"",
		},
		{
			"path in refined backend: ok",
			`server {
  endpoint "/**" {
    proxy {
      backend "a" {
        path = "/a/**"
      }
    }
  }
}
definitions {
  backend "a" {
  }
}`,
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			_, err := LoadBytes([]byte(tt.hcl), "couper.hcl")

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

func Test_checkReferencedAccessControls(t *testing.T) {
	tests := []struct {
		name  string
		hcl   string
		error string
	}{
		{
			"missing AC referenced by server access_control",
			`server {
			  access_control = ["undefined"]
			}`,
			`couper.hcl:2,23-36: referenced access control "undefined" is not defined; `,
		},
		{
			"missing AC referenced by server disable_access_control",
			`server {
			  disable_access_control = ["undefined"]
			}`,
			`couper.hcl:2,31-44: referenced access control "undefined" is not defined; `,
		},
		{
			"missing AC referenced by api access_control",
			`server {
			  api {
			    access_control = ["undefined"]
			  }
			}`,
			`couper.hcl:3,25-38: referenced access control "undefined" is not defined; `,
		},
		{
			"missing AC referenced by api disable_access_control",
			`server {
			  api {
			    disable_access_control = ["undefined"]
			  }
			}`,
			`couper.hcl:3,33-46: referenced access control "undefined" is not defined; `,
		},
		{
			"missing AC referenced by files access_control",
			`server {
			  files {
			    access_control = ["undefined"]
			    document_root = "htdocs"
			  }
			}`,
			`couper.hcl:3,25-38: referenced access control "undefined" is not defined; `,
		},
		{
			"missing AC referenced by files disable_access_control",
			`server {
			  files {
			    disable_access_control = ["undefined"]
			    document_root = "htdocs"
			  }
			}`,
			`couper.hcl:3,33-46: referenced access control "undefined" is not defined; `,
		},
		{
			"missing AC referenced by spa access_control",
			`server {
			  spa {
			    access_control = ["undefined"]
			    bootstrap_file = "foo"
			    paths = ["/**"]
			  }
			}`,
			`couper.hcl:3,25-38: referenced access control "undefined" is not defined; `,
		},
		{
			"missing AC referenced by spa disable_access_control",
			`server {
			  spa {
			    disable_access_control = ["undefined"]
			    bootstrap_file = "foo"
			    paths = ["/**"]
			  }
			}`,
			`couper.hcl:3,33-46: referenced access control "undefined" is not defined; `,
		},
		{
			"missing AC referenced by endpoint access_control",
			`server {
			  endpoint "/" {
			    access_control = ["undefined"]
			    response {
			      body = "OK"
			    }
			  }
			}`,
			`couper.hcl:3,25-38: referenced access control "undefined" is not defined; `,
		},
		{
			"missing AC referenced by endpoint disable_access_control",
			`server {
			  endpoint "/" {
			    disable_access_control = ["undefined"]
			    response {
			      body = "OK"
			    }
			  }
			}`,
			`couper.hcl:3,33-46: referenced access control "undefined" is not defined; `,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			_, err := LoadBytes([]byte(tt.hcl), "couper.hcl")

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
