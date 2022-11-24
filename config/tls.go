package config

type ServerTLS struct {
	// TBA
	//Ocsp               bool                 `hcl:"ocsp,optional"`
	//OcspTTL            string               `hcl:"ocsp_ttl,optional" type:"duration" default:"12h"`
	ClientCertificate  []*ClientCertificate `hcl:"client_certificate,block"`
	ServerCertificates []*ServerCertificate `hcl:"server_certificate,block"`
}

type BackendTLS struct {
	ServerCertificate     string `hcl:"server_ca_certificate,optional" docs:"Public part of the certificate authority in DER or PEM format."`
	ServerCertificateFile string `hcl:"server_ca_certificate_file,optional" docs:"Public part of the certificate authority file in DER or PEM format."`
	ClientCertificate     string `hcl:"client_certificate,optional" docs:"Public part of the client certificate in DER or PEM format."`
	ClientCertificateFile string `hcl:"client_certificate_file,optional" docs:"Public part of the client certificate file in DER or PEM format."`
	ClientPrivateKey      string `hcl:"client_private_key,optional" docs:"Private part of the client certificate in DER or PEM format. Required to complete an mTLS handshake."`
	ClientPrivateKeyFile  string `hcl:"client_private_key_file,optional" docs:"Private part of the client certificate file in DER or PEM format. Required to complete an mTLS handshake."`
}
