package reader_test

import (
	"os"
	"reflect"
	"runtime"
	"testing"

	"github.com/avenga/couper/config/reader"
)

func TestReadFromAttrFile(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	expBytes, ferr := os.ReadFile(file)
	if ferr != nil {
		t.Fatal(ferr)
	}

	type args struct {
		context   string
		attribute string
		path      string
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{"not configured", args{context: "testcase"}, nil, true},
		{"both configured", args{context: "testcase", attribute: "", path: ""}, nil, true},
		{"attr configured", args{context: "testcase", attribute: "key", path: ""}, []byte("key"), false},
		{"path configured", args{context: "testcase", attribute: "", path: file}, expBytes, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			got, err := reader.ReadFromAttrFile(tt.args.context, tt.args.attribute, tt.args.path)
			if (err != nil) != tt.wantErr {
				subT.Errorf("ReadFromAttrFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				subT.Errorf("ReadFromAttrFile() got = %v, want %v", got, tt.want)
			}
		})
	}
}
