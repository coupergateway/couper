package config

import (
	"net/http"

	"github.com/sirupsen/logrus"
)

type Claims struct {
	Issuer   string  `hcl:"iss,optional"`
	Audience string  `hcl:"aud,optional"`
}

type Jwt struct {
	Name               string  `hcl:"name,label"`
	Cookie             string  `hcl:"cookie,optional"`
	Header             string  `hcl:"header,optional"`
	PostParam          string  `hcl:"post_param,optional"`
	QueryParam         string  `hcl:"query_parm,optional"`
	Key                string  `hcl:"key,optional"`
	KeyFile            string  `hcl:"key_file,optional"`
	SignatureAlgorithm string  `hcl:"signature_algorithm"`
	Claims             *Claims `hcl:"claims,block"`
	log                *logrus.Entry
}

func (j *Jwt) Init(log *logrus.Entry) {
	j.log = log
}

func (j *Jwt) Check(req *http.Request) bool {
	return true
}
