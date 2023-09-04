package lib_test

import (
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/coupergateway/couper/config/configload"
	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/eval"
	"github.com/coupergateway/couper/internal/test"
)

func TestMerge(t *testing.T) {
	helper := test.New(t)

	cf, err := configload.LoadBytes([]byte(`server "test" {}`), "couper.hcl")
	helper.Must(err)

	hclContext := cf.Context.Value(request.ContextType).(*eval.Context).HCLContext()
	mergeFn := hclContext.Functions["merge"]

	tests := []struct {
		name string
		args []cty.Value
		want cty.Value
	}{
		{
			/*
				merge(
					{"k1": 1},
					{"k2": {"k2.2": 2}},
					{"k3": 3},
					null,
					{"k2": {"k4.2": 4}},
					{"k3": 5},
					{"k6": [6]},
					null,
					{"k6": ["7", true]},
					{"k6": [null]},
					{"k6": [8]},
					{"k9": [9]},
					{"k9": 10}
				)
			*/
			"merge objects ignoring null arguments",
			[]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"k1": cty.NumberIntVal(1),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"k2": cty.ObjectVal(map[string]cty.Value{
						"k2.2": cty.NumberIntVal(2),
					}),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"k3": cty.NumberIntVal(3),
				}),
				cty.NullVal(cty.Bool),
				cty.ObjectVal(map[string]cty.Value{
					"k2": cty.MapVal(map[string]cty.Value{ // ähnlicher Typ -> mergen
						"k4.2": cty.NumberIntVal(4),
					}),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"k3": cty.NumberIntVal(5), // gleicher Typ, primitive -> überschreiben
				}),
				cty.ObjectVal(map[string]cty.Value{
					"k6": cty.TupleVal([]cty.Value{
						cty.NumberIntVal(6),
					}),
				}),
				cty.NullVal(cty.Bool),
				cty.ObjectVal(map[string]cty.Value{
					"k6": cty.TupleVal([]cty.Value{ // gleicher Typ -> mergen
						cty.StringVal("7"),
						cty.BoolVal(true),
					}),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"k6": cty.ListVal([]cty.Value{ // ähnlicher Typ -> mergen
						cty.NullVal(cty.Bool),
					}),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"k6": cty.ListVal([]cty.Value{ // ähnlicher Typ -> mergen
						cty.NumberIntVal(8),
					}),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"k9": cty.ListVal([]cty.Value{
						cty.NumberIntVal(9),
					}),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"k9": cty.NumberIntVal(10), // unterschiedlicher Typ -> überschreiben
				}),
			},
			/*
				{
				  "k1": 1,
				  "k2": {"k2.2": 2, "k4.2": 4},
				  "k3": 5,
				  "k6": [6, "7", true, null, 8],
				  "k9": 10
				}
			*/
			cty.ObjectVal(map[string]cty.Value{
				"k1": cty.NumberIntVal(1),
				"k2": cty.ObjectVal(map[string]cty.Value{
					"k2.2": cty.NumberIntVal(2),
					"k4.2": cty.NumberIntVal(4),
				}),
				"k3": cty.NumberIntVal(5),
				"k6": cty.TupleVal([]cty.Value{
					cty.NumberIntVal(6),
					cty.StringVal("7"),
					cty.BoolVal(true),
					cty.NullVal(cty.Bool),
					cty.NumberIntVal(8),
				}),
				"k9": cty.NumberIntVal(10),
			}),
		},
		{
			/*
				merge(
					{"k1": 1}
				)
			*/
			"merge with only one object",
			[]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"k1": cty.NumberIntVal(1),
				}),
			},
			/*
				{
				  "k1": 1
				}
			*/
			cty.ObjectVal(map[string]cty.Value{
				"k1": cty.NumberIntVal(1),
			}),
		},
		{
			/*
				merge(
					{"k2": {"k2.2": 2}},
					{"k0": null},
					{"k2": {"k4.2": 4}},
					{"k2": {"k4.0": null}}
				)
			*/
			"merge objects with null",
			[]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"k2": cty.ObjectVal(map[string]cty.Value{
						"k2.2": cty.NumberIntVal(2),
					}),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"k0": cty.NullVal(cty.Bool),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"k2": cty.MapVal(map[string]cty.Value{
						"k4.2": cty.NumberIntVal(4),
					}),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"k2": cty.MapVal(map[string]cty.Value{
						"k4.0": cty.NullVal(cty.Bool),
					}),
				}),
			},
			/*
				{
				  "k0": null,
				  "k2": {"k4.2": 4}
				}
			*/
			cty.ObjectVal(map[string]cty.Value{
				"k0": cty.NullVal(cty.Bool),
				"k2": cty.ObjectVal(map[string]cty.Value{
					"k2.2": cty.NumberIntVal(2),
					"k4.2": cty.NumberIntVal(4),
					"k4.0": cty.NullVal(cty.Bool),
				}),
			}),
		},
		{
			/*
				merge(
					[1],
					null,
					["2"],
					[true],
					null,
					[{"k": 4}],
					[[5]]
				)
			*/
			"merge tuples ignoring null arguments",
			[]cty.Value{
				cty.TupleVal([]cty.Value{
					cty.NumberIntVal(1),
				}),
				cty.NullVal(cty.Bool),
				cty.TupleVal([]cty.Value{
					cty.StringVal("2"),
				}),
				cty.ListVal([]cty.Value{
					cty.BoolVal(true),
				}),
				cty.NullVal(cty.Bool),
				cty.TupleVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"k": cty.NumberIntVal(4),
					}),
				}),
				cty.TupleVal([]cty.Value{
					cty.TupleVal([]cty.Value{
						cty.NumberIntVal(5),
					}),
				}),
			},
			/*
				[
				  1,
				  "2",
				  true,
				  {"k": 4},
				  [5]
				]
			*/
			cty.TupleVal([]cty.Value{
				cty.NumberIntVal(1),
				cty.StringVal("2"),
				cty.BoolVal(true),
				cty.ObjectVal(map[string]cty.Value{
					"k": cty.NumberIntVal(4),
				}),
				cty.TupleVal([]cty.Value{
					cty.NumberIntVal(5),
				}),
			}),
		},
		{
			/*
				merge(
					[1]
				)
			*/
			"merge with only one tuple",
			[]cty.Value{
				cty.TupleVal([]cty.Value{
					cty.NumberIntVal(1),
				}),
			},
			/*
				[
				  1
				]
			*/
			cty.TupleVal([]cty.Value{
				cty.NumberIntVal(1),
			}),
		},
		{
			/*
				merge()
			*/
			"merge without parameters",
			[]cty.Value{},
			/* null */
			cty.NullVal(cty.Bool),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			h := test.New(subT)

			mergedV, merr := mergeFn.Call(tt.args)
			h.Must(merr)
			if !mergedV.RawEquals(tt.want) {
				subT.Errorf("Wrong return value:\nwant:\t%#v\ngot:\t%#v\n", tt.want, mergedV)
			}
		})
	}
}

func TestMergeErrors(t *testing.T) {
	helper := test.New(t)

	cf, err := configload.LoadBytes([]byte(`server "test" {}`), "couper.hcl")
	helper.Must(err)

	tests := []struct {
		name    string
		args    []cty.Value
		wantErr string
	}{
		{
			/*
				merge(
					{"k1": 1},
					true,
					{"k3": 3},
				)
			*/
			"mix objects with bool",
			[]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"k1": cty.NumberIntVal(1),
				}),
				cty.BoolVal(true),
				cty.ObjectVal(map[string]cty.Value{
					"k3": cty.NumberIntVal(3),
				}),
			},
			"cannot merge primitive value",
		},
		{
			/*
				merge(
					{"k1": 1},
					2,
					{"k3": 3},
				)
			*/
			"mix objects with integer",
			[]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"k1": cty.NumberIntVal(1),
				}),
				cty.NumberIntVal(2),
				cty.ObjectVal(map[string]cty.Value{
					"k3": cty.NumberIntVal(3),
				}),
			},
			"cannot merge primitive value",
		},
		{
			/*
				merge(
					{"k1": 1},
					"2",
					{"k3": 3},
				)
			*/
			"mix objects with string",
			[]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"k1": cty.NumberIntVal(1),
				}),
				cty.StringVal("2"),
				cty.ObjectVal(map[string]cty.Value{
					"k3": cty.NumberIntVal(3),
				}),
			},
			"cannot merge primitive value",
		},
		{
			/*
				merge(
					["1"],
					true,
					["3"],
				)
			*/
			"mix tuples with bool",
			[]cty.Value{
				cty.TupleVal([]cty.Value{
					cty.StringVal("1"),
				}),
				cty.BoolVal(true),
				cty.TupleVal([]cty.Value{
					cty.StringVal("3"),
				}),
			},
			"cannot merge primitive value",
		},
		{
			/*
				merge(
					["1"],
					2,
					["3"],
				)
			*/
			"mix tuples with integer",
			[]cty.Value{
				cty.TupleVal([]cty.Value{
					cty.StringVal("1"),
				}),
				cty.NumberIntVal(2),
				cty.TupleVal([]cty.Value{
					cty.StringVal("3"),
				}),
			},
			"cannot merge primitive value",
		},
		{
			/*
				merge(
					["1"],
					"2",
					["3"],
				)
			*/
			"mix tuples with string",
			[]cty.Value{
				cty.TupleVal([]cty.Value{
					cty.StringVal("1"),
				}),
				cty.StringVal("2"),
				cty.TupleVal([]cty.Value{
					cty.StringVal("3"),
				}),
			},
			"cannot merge primitive value",
		},
		{
			/*
				merge(
					{"k1": 1},
					["2"],
					{"k3": 3},
				)
			*/
			"mix objects with tuple",
			[]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"k1": cty.NumberIntVal(1),
				}),
				cty.TupleVal([]cty.Value{
					cty.StringVal("2"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"k3": cty.NumberIntVal(3),
				}),
			},
			"type mismatch",
		},
		{
			/*
				merge(
					["1"],
					{"k2": 2},
					["3"],
				)
			*/
			"mix tuples with object",
			[]cty.Value{
				cty.TupleVal([]cty.Value{
					cty.StringVal("1"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"k2": cty.NumberIntVal(2),
				}),
				cty.TupleVal([]cty.Value{
					cty.StringVal("3"),
				}),
			},
			"type mismatch",
		},
	}

	hclContext := cf.Context.Value(request.ContextType).(*eval.Context).HCLContext()

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			_, err := hclContext.Functions["merge"].Call(tt.args)
			if err == nil {
				subT.Error("Error expected")
			}
			if err != nil && err.Error() != tt.wantErr {
				subT.Errorf("Wrong error message; expected %#v, got: %#v", tt.wantErr, err.Error())
			}
		})
	}
}
