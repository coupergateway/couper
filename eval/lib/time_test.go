package lib_test

import (
	"testing"
	"time"

	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/config/configload"
	"github.com/avenga/couper/internal/test"
)

func TestUnixtime(t *testing.T) {
	tests := []struct {
		name string
		hcl  string
		want int64
	}{
		{
			"unixtime",
			`
			server "test" {
			}
			`,
			time.Now().Unix(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helper := test.New(t)
			cf, err := configload.LoadBytes([]byte(tt.hcl), "couper.hcl")
			helper.Must(err)
			now, err := cf.Context.HCLContext().Functions["unixtime"].Call([]cty.Value{})
			helper.Must(err)

			if !cty.Number.Equals(now.Type()) {
				t.Errorf("Wrong return type; expected %s, got: %s", cty.Number.FriendlyName(), now.Type().FriendlyName())
			}

			bfnow := now.AsBigFloat()
			inow, _ := bfnow.Int64()
			if inow != tt.want {
				t.Errorf("Wrong return value; expected %d, got: %d", tt.want, inow)
			}
		})
	}
}
