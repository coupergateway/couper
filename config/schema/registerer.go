package schema

import (
	"reflect"
	"strings"

	"github.com/hashicorp/hcl/v2"

	"github.com/avenga/couper/errors"
)

type BlockSchema struct {
	Name   string
	Header *hcl.BlockHeaderSchema
	Body   *hcl.BodySchema
}

type blockMap map[string]*BlockSchema

type Registerer blockMap

func (r Registerer) Add(header *hcl.BlockHeaderSchema, bs BodySchema, parentTypes ...string) {
	if bs == nil {
		panic("missing reference in struct object: " + header.Type)
	}

	bodySchema := bs.Schema()
	if header != nil {
		if _, exist := r[header.Type]; exist {
			return
		}
		r[header.Type] = &BlockSchema{
			Name:   header.Type,
			Header: header,
			Body:   bodySchema,
		}

		for _, pt := range parentTypes {
			r[pt].Body.Blocks = append(r[pt].Body.Blocks, *header)
		}
	}

	for _, block := range bodySchema.Blocks {
		instance := NewFromFieldType(block.Type, bs)
		if instance == nil {
			panic(header.Type + ": nil reference for " + block.Type)
		}
		b := block
		r.Add(&b, instance.(BodySchema))
	}
}

func (r Registerer) GetFor(obj any) *hcl.BodySchema {
	needle := whichParentType(obj)
	if needle == "couper" {
		result := &hcl.BodySchema{}
		for _, topLevel := range []string{"server", "definitions", "defaults", "settings"} {
			result.Blocks = append(result.Blocks, *r[topLevel].Header)
		}
		return result
	}
	result, exist := r[needle]
	if !exist {
		panic("missing schema for: " + needle)
	}

	return result.Body
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

// lookupMap maps exceptions for (bad) special struct names
var lookupMap = map[string]string{
	"oauth2ac":       "beta_oauth2",
	"oauth2req_auth": "oauth2",
	"open_api":       "openapi",
	"server_tls":     "tls",
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
		t := strings.SplitAfter(pType.String(), ".")[1] // rm pkg
		ttst := errors.TypeToSnakeString(t)
		if n, exist := lookupMap[ttst]; exist {
			return n
		}
		return ttst
	}
	return ""
}
