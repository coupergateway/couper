package configload

import (
	"strings"
	"testing"

	"github.com/avenga/couper/config"
)

func Test_refineEndpoints_noPattern(t *testing.T) {
	err := refineEndpoints(nil, config.Endpoints{{Pattern: ""}}, true)
	if err == nil || !strings.HasSuffix(err.Error(), "endpoint: missing path pattern; ") {
		t.Errorf("refineEndpoints() error = %v, wantErr: endpoint: missing path pattern ", err)
	}
}
