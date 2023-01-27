package config

import "github.com/hashicorp/hcl/v2"

type JWTSigningProfile struct {
	Claims             Claims         `hcl:"claims,optional" docs:"Claims for the JWT payload, claim values are evaluated per request."`
	Headers            hcl.Expression `hcl:"headers,optional" docs:"Additional HTTP header fields for the JWT, {typ} has the default value {JWT}, {alg} cannot be set."`
	Key                string         `hcl:"key,optional" docs:"Private key (in PEM format) for {RS*} and {ES*} variants or the secret for {HS*} algorithms. Mutually exclusive with {key_file}."`
	KeyFile            string         `hcl:"key_file,optional" docs:"Reference to file containing signing key. Mutually exclusive with {key}. See {key} for more information."`
	Name               string         `hcl:"name,label,optional"`
	SignatureAlgorithm string         `hcl:"signature_algorithm" docs:"Algorithm used for signing: {\"RS256\"}, {\"RS384\"}, {\"RS512\"}, {\"HS256\"}, {\"HS384\"}, {\"HS512\"}, {\"ES256\"}, {\"ES384\"}, {\"ES512\"}."`
	TTL                string         `hcl:"ttl" docs:"The token's time-to-live, creates the {exp} claim."`
}
