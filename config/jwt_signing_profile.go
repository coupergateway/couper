package config

import "github.com/hashicorp/hcl/v2"

type JWTSigningProfile struct {
	Claims             Claims         `hcl:"claims,optional"`
	Headers            hcl.Expression `hcl:"headers,optional"`
	Key                string         `hcl:"key,optional"`
	KeyFile            string         `hcl:"key_file,optional"`
	Name               string         `hcl:"name,label"`
	SignatureAlgorithm string         `hcl:"signature_algorithm"`
	TTL                string         `hcl:"ttl"`

	// internally used
	KeyBytes []byte
}
