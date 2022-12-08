package config

type ServerTLS struct {
	// TBA
	//Ocsp               bool                 `hcl:"ocsp,optional"`
	//OcspTTL            string               `hcl:"ocsp_ttl,optional" type:"duration" default:"12h"`
	ClientCertificate  []*ClientCertificate `hcl:"client_certificate,block" docs:"Configures a [client certificate](/configuration/block/client_certificate) (zero or more)."`
	ServerCertificates []*ServerCertificate `hcl:"server_certificate,block" docs:"Configures a [server certificate](/configuration/block/server_certificate) (zero or more)."`
}

type BackendTLS struct {
	ServerCertificate     string `hcl:"server_ca_certificate,optional" docs:"Public part of the certificate authority in DER or PEM format. Mutually exclusive with {server_ca_certificate_file}."`
	ServerCertificateFile string `hcl:"server_ca_certificate_file,optional" docs:"Reference to a file containing the public part of the certificate authority file in DER or PEM format. Mutually exclusive with {server_ca_certificate}."`
	ClientCertificate     string `hcl:"client_certificate,optional" docs:"Public part of the client certificate in DER or PEM format. Mutually exclusive with {client_certificate_file}."`
	ClientCertificateFile string `hcl:"client_certificate_file,optional" docs:"Reference to a file containing the public part of the client certificate file in DER or PEM format. Mutually exclusive with {client_certificate}."`
	ClientPrivateKey      string `hcl:"client_private_key,optional" docs:"Private part of the client certificate in DER or PEM format. Required to complete an mTLS handshake. Mutually exclusive with {client_private_key_file}."`
	ClientPrivateKeyFile  string `hcl:"client_private_key_file,optional" docs:"Reference to a file containing the private part of the client certificate file in DER or PEM format. Required to complete an mTLS handshake. Mutually exclusive with {client_private_key}."`
}
