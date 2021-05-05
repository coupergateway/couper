package configload

import (
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/config"
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
