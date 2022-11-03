package main

import (
	"os"
	"time"

	"github.com/avenga/couper/server"
)

func main() {
	selfSigned, err := server.NewCertificate(time.Hour*12, nil, nil)
	if err != nil {
		panic(err)
	}

	_ = os.WriteFile("couperRootCA.crt", selfSigned.CACertificate.Certificate, 0644)
	_ = os.WriteFile("couperRootCA.key", selfSigned.CACertificate.PrivateKey, 0644)
	_ = os.WriteFile("couperServer.crt", selfSigned.ServerCertificate.Certificate, 0644)
	_ = os.WriteFile("couperServer.key", selfSigned.ServerCertificate.PrivateKey, 0644)
	_ = os.WriteFile("couperClient.crt", selfSigned.ClientCertificate.Certificate, 0644)
	_ = os.WriteFile("couperClient.key", selfSigned.ClientCertificate.PrivateKey, 0644)
	_ = os.WriteFile("couperIntermediate.crt",
		append(selfSigned.ClientIntermediateCertificate.Certificate, selfSigned.CACertificate.Certificate...), 0644)

	println("certificates generated...")
}
