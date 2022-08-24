package configload

import (
	"context"
	"testing"

	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config/runtime"
	"github.com/avenga/couper/errors"
	logrustest "github.com/sirupsen/logrus/hooks/test"
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
		if err := verifyBodyAttributes(request, tc.content); !tc.expErr && err != nil {
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
			"couper.hcl:3,15-19: label contains invalid character(s), allowed are 'a-z', 'A-Z', '0-9' and '_'; ",
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
			"basic_auth reserved label beta_granted_permissions",
			`server {}
			 definitions {
			   basic_auth "beta_granted_permissions" {
			   }
			 }`,
			"couper.hcl:3,18-44: accessControl uses reserved name as label; ",
		},
		{
			"basic_auth reserved label beta_required_permission",
			`server {}
			 definitions {
			   basic_auth "beta_required_permission" {
			   }
			 }`,
			"couper.hcl:3,18-44: accessControl uses reserved name as label; ",
		},
		{
			"beta_oauth2 reserved label beta_granted_permissions",
			`server {}
			 definitions {
			   beta_oauth2 "beta_granted_permissions" {
			   }
			 }`,
			"couper.hcl:3,19-45: accessControl uses reserved name as label; ",
		},
		{
			"beta_oauth2 reserved label beta_required_permission",
			`server {}
			 definitions {
			   beta_oauth2 "beta_required_permission" {
			   }
			 }`,
			"couper.hcl:3,19-45: accessControl uses reserved name as label; ",
		},
		{
			"jwt reserved label beta_granted_permissions",
			`server {}
			 definitions {
			   jwt "beta_granted_permissions" {
			   }
			 }`,
			"couper.hcl:3,11-37: accessControl uses reserved name as label; ",
		},
		{
			"jwt reserved label beta_required_permission",
			`server {}
			 definitions {
			   jwt "beta_required_permission" {
			   }
			 }`,
			"couper.hcl:3,11-37: accessControl uses reserved name as label; ",
		},
		{
			"oidc reserved label beta_granted_permissions",
			`server {}
			 definitions {
			   oidc "beta_granted_permissions" {
			   }
			 }`,
			"couper.hcl:3,12-38: accessControl uses reserved name as label; ",
		},
		{
			"oidc reserved label beta_required_permission",
			`server {}
			 definitions {
			   oidc "beta_required_permission" {
			   }
			 }`,
			"couper.hcl:3,12-38: accessControl uses reserved name as label; ",
		},
		{
			"saml reserved label beta_granted_permissions",
			`server {}
			 definitions {
			   saml "beta_granted_permissions" {
			   }
			 }`,
			"couper.hcl:3,12-38: accessControl uses reserved name as label; ",
		},
		{
			"saml reserved label beta_required_permission",
			`server {}
			 definitions {
			   saml "beta_required_permission" {
			   }
			 }`,
			"couper.hcl:3,12-38: accessControl uses reserved name as label; ",
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

	logger, _ := logrustest.NewNullLogger()
	log := logger.WithContext(context.TODO())

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			conf, err := LoadBytes([]byte(tt.hcl), "couper.hcl")
			if conf != nil {
				tmpStoreCh := make(chan struct{})
				defer close(tmpStoreCh)

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
			"beta_required_permission",
			`server {
  api {
    endpoint "/a" {
      beta_required_permission = {
        get = "a"
        GeT = "A"
      }
    }
  }
}`,
			"couper.hcl:4,34-7,8: key in an attribute must be unique: 'get'; Key must be unique for get.",
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
			"beta_permissions_map",
			`server {}
definitions {
  jwt "a" {
    signature_algorithm = "HS256"
    key = "asdf"
    beta_permissions_map = {
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
      beta_required_permission = "a"
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
      beta_required_permission = "a"
      response {}
    }
    endpoint "/b" {
      beta_required_permission = ""
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
    beta_required_permission = "api"
    endpoint "/a" {
      beta_required_permission = "a"
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
      beta_required_permission = "a"
      response {}
    }
    endpoint "/b" {
      disable_access_control = ["foo"]
      response {}
    }
  }
}`,
			"",
		},
		{
			"no mix: separate apis",
			`server {
  api "foo" {
    endpoint "/a" {
      beta_required_permission = "a"
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
