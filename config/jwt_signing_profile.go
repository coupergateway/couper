package config

type JWTSigningProfile struct {
	Claims             Claims   `hcl:"claims,optional"`
	Key                string   `hcl:"key,optional"`
	KeyFile            string   `hcl:"key_file,optional"`
	Name               string   `hcl:"name,label"`
	SignatureAlgorithm string   `hcl:"signature_algorithm"`
	TTL                string   `hcl:"ttl,optional"`
}
