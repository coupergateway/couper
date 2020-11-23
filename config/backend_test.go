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
			Hostname: "a", Name: "a", Origin: "a", Path: "a", Timeout: "s", ConnectTimeout: "s", Options: hcl.EmptyBody(),
		}, args{&Backend{}}, &Backend{
			Hostname: "a", Name: "a", Origin: "a", Path: "a", Timeout: "s", ConnectTimeout: "s", Options: hcl.EmptyBody(),
		}},
		{"right", Backend{}, args{&Backend{
			Hostname: "a", Name: "a", Origin: "a", Path: "a", Timeout: "s", ConnectTimeout: "s", Options: hcl.EmptyBody(),
		}}, &Backend{
			Hostname: "a", Name: "a", Origin: "a", Path: "a", Timeout: "s", ConnectTimeout: "s", Options: hcl.EmptyBody(),
		}},
		{"override", Backend{
			Hostname: "a", Name: "a", Origin: "a", Path: "a", Timeout: "s", ConnectTimeout: "s", RequestBodyLimit: "2M", TTFBTimeout: "t", Options: hcl.EmptyBody(),
		}, args{&Backend{
			Hostname: "b", Name: "b", Origin: "b", Path: "b", Timeout: "m", ConnectTimeout: "h", RequestBodyLimit: "20M", TTFBTimeout: "o", Options: hcl.EmptyBody(),
		}}, &Backend{
			Hostname: "b", Name: "b", Origin: "b", Path: "b", Timeout: "m", ConnectTimeout: "h", RequestBodyLimit: "20M", TTFBTimeout: "o", Options: hcl.EmptyBody(),
		}},
		{"partial override", Backend{
			Hostname: "a", Name: "b", Origin: "c", Path: "d", Timeout: "e", ConnectTimeout: "f", TTFBTimeout: "t", Options: hcl.EmptyBody(),
		}, args{&Backend{
			Hostname: "c", ConnectTimeout: "d",
		}}, &Backend{
			Hostname: "c", Name: "b", Origin: "c", Path: "d", Timeout: "e", ConnectTimeout: "d", TTFBTimeout: "t", Options: hcl.EmptyBody(),
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Backend{
				ConnectTimeout:   tt.fields.ConnectTimeout,
				Hostname:         tt.fields.Hostname,
				Name:             tt.fields.Name,
				Options:          tt.fields.Options,
				Origin:           tt.fields.Origin,
				Path:             tt.fields.Path,
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
