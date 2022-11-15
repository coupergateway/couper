package config

type ClientCertificate struct {
	Name     string `hcl:",label,optional"`
	CA       string `hcl:"ca_certificate,optional" docs:"Public part of the certificate authority in DER or PEM format."`
	CAFile   string `hcl:"ca_certificate_file,optional" docs:"Public part of the certificate authority file in DER or PEM format."`
	Leaf     string `hcl:"leaf_certificate,optional" docs:"Public part of the client certificate in DER or PEM format."`
	LeafFile string `hcl:"leaf_certificate_file,optional" docs:"Public part of the client certificate file in DER or PEM format."`
}

type ServerCertificate struct {
	Name           string `hcl:",label,optional"`
	PublicKey      string `hcl:"public_key,optional" docs:"Public part of the certificate in DER or PEM format."`
	PublicKeyFile  string `hcl:"public_key_file,optional" docs:"Public part of the certificate file in DER or PEM format."`
	PrivateKey     string `hcl:"private_key,optional" docs:"Private part of the certificate in DER or PEM format."`
	PrivateKeyFile string `hcl:"private_key_file,optional" docs:"Private part of the certificate file in DER or PEM format."`
}
