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

func TestReadFromAttrFileJSONObjectOptional(t *testing.T) {
	expMap := map[string][]string{
		"a": {"a1", "a2"},
		"b": {"b1"},
		"c": {},
	}

	type args struct {
		context   string
		attribute map[string][]string
		path      string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string][]string
		wantErr bool
	}{
		{"not configured", args{context: "testcase"}, nil, false},
		{"both configured", args{context: "testcase", attribute: expMap, path: "testdata/map.json"}, nil, true},
		{"attr configured", args{context: "testcase", attribute: expMap, path: ""}, expMap, false},
		{"path configured", args{context: "testcase", attribute: nil, path: "testdata/map.json"}, expMap, false},
		{"path configured, invalid file content", args{context: "testcase", attribute: nil, path: "testdata/map_error.json"}, nil, true},
		{"path configured, file not found", args{context: "testcase", attribute: nil, path: "testdata/file_not_found"}, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			got, err := reader.ReadFromAttrFileJSONObjectOptional(tt.args.context, tt.args.attribute, tt.args.path)
			if (err != nil) != tt.wantErr {
				subT.Errorf("ReadFromAttrFileJSONObject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				subT.Errorf("ReadFromAttrFileJSONObject() got = %v, want %v", got, tt.want)
			}
		})
	}
}
