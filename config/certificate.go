package config

type ClientCertificate struct {
	Name     string `hcl:",label,optional"`
	CA       string `hcl:"ca_certificate,optional"`
	CAFile   string `hcl:"ca_certificate_file,optional"`
	Leaf     string `hcl:"leaf_certificate,optional"`
	LeafFile string `hcl:"leaf_certificate_file,optional"`
}

type ServerCertificate struct {
	Name           string `hcl:",label,optional"`
	PublicKey      string `hcl:"public_key,optional"`
	PublicKeyFile  string `hcl:"public_key_file,optional"`
	PrivateKey     string `hcl:"private_key,optional"`
	PrivateKeyFile string `hcl:"private_key_file,optional"`
}
