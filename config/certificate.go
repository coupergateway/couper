package config

type ClientCertificate struct {
	Name     string `hcl:",label_optional"`
	CA       string `hcl:"ca_certificate,optional" docs:"Public part of the certificate authority in DER or PEM format. Mutually exclusive with {ca_certificate_file}."`
	CAFile   string `hcl:"ca_certificate_file,optional" docs:"Reference to a file containing the public part of the certificate authority file in DER or PEM format. Mutually exclusive with {ca_certificate}."`
	Leaf     string `hcl:"leaf_certificate,optional" docs:"Public part of the client certificate in DER or PEM format. Mutually exclusive with {leaf_certificate_file}."`
	LeafFile string `hcl:"leaf_certificate_file,optional" docs:"Reference to a file containing the public part of the client certificate file in DER or PEM format. Mutually exclusive with {leaf_certificate}."`
}

type ServerCertificate struct {
	Name           string `hcl:",label_optional"`
	PublicKey      string `hcl:"public_key,optional" docs:"Public part of the certificate in DER or PEM format. Mutually exclusive with {public_key_file}."`
	PublicKeyFile  string `hcl:"public_key_file,optional" docs:"Reference to a file containing the public part of the certificate file in DER or PEM format. Mutually exclusive with {public_key}."`
	PrivateKey     string `hcl:"private_key,optional" docs:"Private part of the certificate in DER or PEM format. Mutually exclusive with {private_key_file}."`
	PrivateKeyFile string `hcl:"private_key_file,optional" docs:"Reference to a file containing the private part of the certificate file in DER or PEM format. Mutually exclusive with {private_key}."`
}
