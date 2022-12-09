package schema

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

type BlockSchema struct {
	Name   string
	Header *hcl.BlockHeaderSchema
	Body   *hcl.BodySchema
}

type schemaMap map[string]any
type blockMap map[string]*BlockSchema

type Registerer schemaMap

func (r Registerer) Add(parentType any, header *hcl.BlockHeaderSchema, bs BodySchema) bool {
	pt := whichParentType(parentType)

	if bs == nil {
		panic("missing reference in " + pt + " struct object: " + header.Type)
	}

	if pt == header.Type && len(bs.Schema().Blocks) == 0 {
		if _, exist := r[pt]; exist {
			return true
		}

		r[pt] = &BlockSchema{
			Name:   header.Type,
			Header: header,
			Body:   bs.Schema(),
		}
	} else {
		if _, exist := r[pt]; !exist {
			r[pt] = make(blockMap)
		} else if _, exist = r[pt].(blockMap)[header.Type]; exist {
			return true
		}
		r[pt].(blockMap)[header.Type] = &BlockSchema{
			Name:   header.Type,
			Header: header,
			Body:   bs.Schema(),
		}
	}

	// additionally self register
	return r.Add(header.Type, header, bs)
}

func (r Registerer) GetFor(obj any) *hcl.BodySchema {
	needle := whichParentType(obj)
	result, exist := r[needle]
	if !exist {
		return nil
	}

	schema := &hcl.BodySchema{}
	if isBlock(result) {
		for _, bh := range result.(blockMap) {
			schema.Blocks = append(schema.Blocks, *bh.Header)
		}
	} else {
		schema = result.(*BlockSchema).Body
	}
	if len(schema.Attributes) == 0 && len(schema.Blocks) == 0 {
		fmt.Printf("%#v\n", result)
		panic("missing schema for " + reflect.TypeOf(obj).String())
	}
	return schema
}

func (r Registerer) AddRecursive(obj any) {
	bs := obj.(BodySchema).Schema()
	for _, block := range bs.Blocks {
		instance := NewFromFieldType(block.Type, obj)
		if _, ok := instance.(BodySchema); !ok {
			instance = wrappedBody{instance}
		}
		if known := r.Add(obj, &block, instance.(BodySchema)); !known || reflect.TypeOf(obj).Name() == "Couper" {
			r.AddRecursive(instance)
		}
	}
}

func NewFromFieldType(name string, obj any) any {
	t := elemType(reflect.TypeOf(obj))

	for i := 0; i < t.NumField(); i++ {
		tagValue, exist := t.Field(i).Tag.Lookup("hcl")
		if !exist || tagValue != name+",block" {
			continue
		}
		field := t.Field(i)
		et := elemType(field.Type)
		return reflect.New(et).Interface()
	}
	return nil
}

func elemType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice {
		t = t.Elem()
	}
	return t
}

func whichParentType(pt any) string {
	switch v := pt.(type) {
	case string:
		return v
	case any:
		pType := reflect.TypeOf(v)
		if pType.Kind() == reflect.Ptr {
			pType = pType.Elem()
		}
		return strings.ToLower(strings.SplitAfter(pType.String(), ".")[1])
	}
	return ""
}

func isBlock(obj any) bool {
	_, is := obj.(blockMap)
	return is
}

type wrappedBody struct {
	obj any
}

func (wb wrappedBody) Schema() *hcl.BodySchema {
	s, _ := gohcl.ImpliedBodySchema(wb.obj)
	return s
}
