package config

type ServerTLS struct {
	Ocsp               bool                 `hcl:"ocsp,optional"`
	OcspTTL            string               `hcl:"ocsp_ttl,optional" type:"duration" default:"12h"`
	ClientCertificate  []*ClientCertificate `hcl:"client_certificate,block"`
	ServerCertificates []*ServerCertificate `hcl:"server_certificate,block"`
}

type BackendTLS struct {
	ServerCertificate     string `hcl:"server_ca_certificate,optional"`
	ServerCertificateFile string `hcl:"server_ca_certificate_file,optional"`
	ClientCertificate     string `hcl:"client_certificate,optional"`
	ClientCertificateFile string `hcl:"client_certificate_file,optional"`
	ClientPrivateKey      string `hcl:"client_private_key,optional"`
	ClientPrivateKeyFile  string `hcl:"client_private_key_file,optional"`
}
