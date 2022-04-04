package configload

import (
	"context"
	"testing"

	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config/runtime"
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
			"beta_scope",
			`server {
  api {
    endpoint "/a" {
      beta_scope = {
        get = "a"
        GeT = "A"
      }
    }
  }
}`,
			"couper.hcl:4,20-7,8: key in an attribute must be unique: 'get'; Key must be unique for get.",
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
			"beta_scope_map",
			`server {}
definitions {
  jwt "a" {
    signature_algorithm = "HS256"
    key = "asdf"
    beta_scope_map = {
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

	logger, _ := logrustest.NewNullLogger()
	log := logger.WithContext(context.TODO())

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			conf, err := LoadBytes([]byte(tt.hcl), "couper.hcl")
			if conf != nil {
				_, err = runtime.NewServerConfiguration(conf, log, nil)
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
