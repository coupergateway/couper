package config_test

import (
	"reflect"
	"testing"

	"github.com/coupergateway/couper/config"
)

func TestAccessControl_List(t *testing.T) {
	type fields struct {
		AccessControl        []string
		DisableAccessControl []string
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{"none", fields{}, nil},
		{"none & disabled", fields{AccessControl: []string{}, DisableAccessControl: []string{"one"}}, nil},
		{"all", fields{AccessControl: []string{"one"}}, []string{"one"}},
		{"all disabled", fields{AccessControl: []string{"one"}, DisableAccessControl: []string{"one"}}, nil},
		{"2nd entry disabled", fields{AccessControl: []string{"one", "two"}, DisableAccessControl: []string{"two"}}, []string{"one"}},
		{"1st entry disabled", fields{AccessControl: []string{"one", "two"}, DisableAccessControl: []string{"one"}}, []string{"two"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			ac := config.AccessControl{
				AccessControl:        tt.fields.AccessControl,
				DisableAccessControl: tt.fields.DisableAccessControl,
			}
			if got := ac.List(); !reflect.DeepEqual(got, tt.want) {
				subT.Errorf("List() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAccessControl_Merge(t *testing.T) {
	type fields struct {
		AccessControl        []string
		DisableAccessControl []string
	}
	type args struct {
		oac config.AccessControl
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   config.AccessControl
	}{
		{"empty", fields{}, args{}, config.AccessControl{}},
		{"empty and not-empty", fields{}, args{config.AccessControl{AccessControl: []string{"one"}}}, config.AccessControl{AccessControl: []string{"one"}}},
		{"not-empty and empty", fields{AccessControl: []string{"one"}}, args{}, config.AccessControl{AccessControl: []string{"one"}}},
		{
			"contains in AccessControl",
			fields{AccessControl: []string{"one"}},
			args{config.AccessControl{AccessControl: []string{"one"}}},
			config.AccessControl{AccessControl: []string{"one"}},
		},
		{
			"contains in DisableAccessControl",
			fields{DisableAccessControl: []string{"one"}},
			args{config.AccessControl{DisableAccessControl: []string{"one"}}},
			config.AccessControl{DisableAccessControl: []string{"one"}},
		},
		{
			"not contains in DisableAccessControl",
			fields{},
			args{config.AccessControl{DisableAccessControl: []string{"one"}}},
			config.AccessControl{DisableAccessControl: []string{"one"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			ac := config.NewAccessControl(tt.fields.AccessControl, tt.fields.DisableAccessControl)

			if got := ac.Merge(tt.args.oac); !reflect.DeepEqual(got, tt.want) {
				subT.Errorf("Merge() = %v, want %v", got, tt.want)
			}
		})
	}
}
