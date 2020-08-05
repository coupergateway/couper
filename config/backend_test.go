package config

import (
	"reflect"
	"testing"

	"github.com/hashicorp/hcl/v2"
)

func TestBackend_Merge(t *testing.T) {
	type fields struct {
		Hostname       string
		Name           string
		Origin         string
		Path           string
		Timeout        string
		ConnectTimeout string
		TTFBTimeout    string
		Options        hcl.Body
	}
	type args struct {
		other *Backend
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *Backend
	}{
		{"nil", fields{}, args{nil}, nil},
		{"empty", fields{}, args{&Backend{}}, &Backend{}},
		{"left", fields{
			Hostname: "a", Name: "a", Origin: "a", Path: "a", Timeout: "s", ConnectTimeout: "s", Options: hcl.EmptyBody(),
		}, args{&Backend{}}, &Backend{
			Hostname: "a", Name: "a", Origin: "a", Path: "a", Timeout: "s", ConnectTimeout: "s", Options: hcl.EmptyBody(),
		}},
		{"right", fields{}, args{&Backend{
			Hostname: "a", Name: "a", Origin: "a", Path: "a", Timeout: "s", ConnectTimeout: "s", Options: hcl.EmptyBody(),
		}}, &Backend{
			Hostname: "a", Name: "a", Origin: "a", Path: "a", Timeout: "s", ConnectTimeout: "s", Options: hcl.EmptyBody(),
		}},
		{"override", fields{
			Hostname: "a", Name: "a", Origin: "a", Path: "a", Timeout: "s", ConnectTimeout: "s", TTFBTimeout: "t", Options: hcl.EmptyBody(),
		}, args{&Backend{
			Hostname: "b", Name: "b", Origin: "b", Path: "b", Timeout: "m", ConnectTimeout: "h", TTFBTimeout: "o", Options: hcl.EmptyBody(),
		}}, &Backend{
			Hostname: "b", Name: "b", Origin: "b", Path: "b", Timeout: "m", ConnectTimeout: "h", TTFBTimeout: "o", Options: hcl.EmptyBody(),
		}},
		{"partial override", fields{
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
				ConnectTimeout: tt.fields.ConnectTimeout,
				Hostname:       tt.fields.Hostname,
				Name:           tt.fields.Name,
				Options:        tt.fields.Options,
				Origin:         tt.fields.Origin,
				Path:           tt.fields.Path,
				Timeout:        tt.fields.Timeout,
				TTFBTimeout:    tt.fields.TTFBTimeout,
			}
			if got, _ := b.Merge(tt.args.other); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Merge() = %v, want %v", got, tt.want)
			}
		})
	}
}
