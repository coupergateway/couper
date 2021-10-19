package eval

import (
	"reflect"
	"testing"

	"github.com/avenga/couper/errors"

	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/avenga/couper/config"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

func TestValue(t *testing.T) {
	evalCtx := NewContext(nil, &config.Defaults{}).HCLContext()
	rootObj := cty.ObjectVal(map[string]cty.Value{
		"exist": cty.StringVal("here"),
	})
	evalCtx.Variables["rootvar"] = rootObj

	tests := []struct {
		name    string
		expStr  string
		want    cty.Value
		wantErr bool
	}{
		{"root non nil", "key = rootvar", rootObj, false},
		{"child non nil", "key = rootvar.exist", cty.StringVal("here"), false},
		{"child nil", "key = rootvar.child", cty.NilVal, false},
		{"child idx nil", "key = rootvar.child[2].sub", cty.NilVal, false},
		{"template attr value exp empty string", `key = "prefix${rootvar.child}"`, cty.StringVal("prefix"), false},
		//{"template attr value exp ternary string", `key = true ? rootvar.child : "${rootvar.child}"`, cty.StringVal(""), false},
		//{"template attr value exp ternary string", `keyA = "${rootvar.child}"` + "\n" + `keyB = rootvar.child`, cty.StringVal(""), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(st *testing.T) {
			f, diags := hclsyntax.ParseConfig([]byte(tt.expStr), "mockConfig", hcl.InitialPos)
			if diags.HasErrors() {
				st.Fatal(diags)
			}
			attrs, diags := f.Body.JustAttributes()
			if diags.HasErrors() {
				st.Fatal(diags)
			}

			for _, attr := range attrs {
				got, err := Value(evalCtx, attr.Expr)
				if (err != nil) != tt.wantErr {
					t.Errorf("Value() error = %v, wantErr %v", err, tt.wantErr)
					t.Error(err.(errors.GoError).LogError())
					return
				}
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("Value() got = %v, want %v", got.GoString(), tt.want.GoString())
				}
			}
		})
	}
}
