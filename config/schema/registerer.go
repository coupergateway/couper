package schema

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
)

type BlockHeader struct {
	Name   string
	Header *hcl.BlockHeaderSchema
}
type Registerer map[string]map[BlockHeader]*hcl.BodySchema

func (r Registerer) Add(parentType string, header *hcl.BlockHeaderSchema, bs BodySchema) error {
	if _, exist := r[parentType]; !exist {
		r[parentType] = make(map[BlockHeader]*hcl.BodySchema)
	}
	blockHeader := BlockHeader{
		Name:   header.Type,
		Header: header,
	}
	// just once per parent
	if _, exist := r[parentType][blockHeader]; exist {
		return fmt.Errorf("schema for %s already exists", header.Type)
	}

	r[parentType][blockHeader] = bs.Schema()

	return nil
}
