package config

import (
	"reflect"
	"testing"

	"github.com/hashicorp/hcl/v2"
)

func TestBackend_Merge(t *testing.T) {
	type args struct {
		other *Backend
	}
	tests := []struct {
		name   string
		fields Backend
		args   args
		want   *Backend
	}{
		{"nil", Backend{}, args{nil}, nil},
		{"empty", Backend{}, args{&Backend{}}, &Backend{}},
		{"left", Backend{
			Timeout: "s", ConnectTimeout: "s", Remain: hcl.EmptyBody(),
		}, args{&Backend{}}, &Backend{
			Timeout: "s", ConnectTimeout: "s", Remain: hcl.EmptyBody(),
		}},
		{"right", Backend{}, args{&Backend{
			Timeout: "s", ConnectTimeout: "s", Remain: hcl.EmptyBody(),
		}}, &Backend{
			Timeout: "s", ConnectTimeout: "s", Remain: hcl.EmptyBody(),
		}},
		{"override", Backend{
			Timeout: "s", ConnectTimeout: "s", RequestBodyLimit: "2M", TTFBTimeout: "t", Remain: hcl.EmptyBody(),
		}, args{&Backend{
			Timeout: "m", ConnectTimeout: "h", RequestBodyLimit: "20M", TTFBTimeout: "o", Remain: hcl.EmptyBody(),
		}}, &Backend{
			Timeout: "m", ConnectTimeout: "h", RequestBodyLimit: "20M", TTFBTimeout: "o", Remain: hcl.EmptyBody(),
		}},
		{"partial override", Backend{
			Timeout: "e", ConnectTimeout: "f", TTFBTimeout: "t", Remain: hcl.EmptyBody(),
		}, args{&Backend{
			ConnectTimeout: "d",
		}}, &Backend{
			Timeout: "e", ConnectTimeout: "d", TTFBTimeout: "t", Remain: hcl.EmptyBody(),
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Backend{
				ConnectTimeout:   tt.fields.ConnectTimeout,
				Name:             tt.fields.Name,
				Remain:           tt.fields.Remain,
				RequestBodyLimit: tt.fields.RequestBodyLimit,
				Timeout:          tt.fields.Timeout,
				TTFBTimeout:      tt.fields.TTFBTimeout,
			}
			if got, _ := b.Merge(tt.args.other); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Merge() = %v, want %v", got, tt.want)
			}
		})
	}
}
