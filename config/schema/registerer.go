package schema

import (
	"reflect"
	"strings"

	"github.com/hashicorp/hcl/v2"
)

type BlockHeader struct {
	Name   string
	Header *hcl.BlockHeaderSchema
}

type Registerer map[string]map[BlockHeader]*hcl.BodySchema

func (r Registerer) Add(parentType any, header *hcl.BlockHeaderSchema, bs BodySchema) {
	pt := whichParentType(parentType)

	if _, exist := r[pt]; !exist {
		r[pt] = make(map[BlockHeader]*hcl.BodySchema)
	}

	blockHeader := BlockHeader{
		Name:   header.Type,
		Header: header,
	}
	// just once per parent
	if _, exist := r[pt][blockHeader]; exist {
		return // TODO: required check to prevent plugin overrides system ones
	}

	r[pt][blockHeader] = bs.Schema()

	// additionally self register
	r.Add(header.Type, header, bs)
}

func (r Registerer) GetFor(obj any) *hcl.BodySchema {
	needle := whichParentType(obj)
	result, exist := r[needle]
	if !exist {
		return nil
	}

	schema := &hcl.BodySchema{}
	for bh := range result {
		schema.Blocks = append(schema.Blocks, *bh.Header)
	}
	if len(schema.Blocks) == 0 {
		panic("missing schema for " + reflect.TypeOf(obj).String())
	}
	return schema
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
