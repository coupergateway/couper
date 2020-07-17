package config

import (
	"strings"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"net/http"

	"github.com/dgrijalva/jwt-go/v4"
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
	parser             *jwt.Parser
	key                *rsa.PublicKey
}

// TODO read values from Env

func (j *Jwt) Init(log *logrus.Entry) {
	j.log = log
	if j.KeyFile != "" {
		pem, err := ioutil.ReadFile(j.KeyFile)
		if err != nil {
			log.Fatal(err)
		}
		j.key, err = jwt.ParseRSAPublicKeyFromPEM([]byte(pem))
		if err != nil {
			log.Fatal(err)
		}
	} else if j.Key != "" {
		if strings.HasPrefix(j.SignatureAlgorithm, "RS") {
			var err error
			var pub interface{}
			// try pem
			block, _ := pem.Decode([]byte(j.Key))
			if block != nil {
				pub, err = x509.ParsePKIXPublicKey(block.Bytes)
			} else {
				// TODO
				log.Fatal("non-pem key not implemented")
			}
			if err != nil {
				log.Fatal(err)
			}
			j.key, _ = pub.(*rsa.PublicKey)
		} else {
			// TODO HMAC
		}
	} else {
		log.Fatal("Either key_file or key must be specified")
	}
	var options []jwt.ParserOption
	options = append(options, jwt.WithValidMethods([]string{j.SignatureAlgorithm}))
	if j.Claims.Issuer != "" {
		options = append(options, jwt.WithIssuer(j.Claims.Issuer))
	}
	if j.Claims.Audience != "" {
		options = append(options, jwt.WithAudience(j.Claims.Audience))
	} else {
		options = append(options, jwt.WithoutAudienceValidation())
	}
	j.parser = jwt.NewParser(options...)
}

func (j *Jwt) Check(req *http.Request) bool {
	tokenValue := ""
	if j.Header != "" {
		tokenValue = req.Header.Get(j.Header)
		if j.Header == "Authorization" {
			if strings.HasPrefix(strings.ToLower(tokenValue), "bearer ") {
				tokenValue = strings.Trim(tokenValue[7:len(tokenValue)], " ")
			} else {
				j.log.Error("Authorization header value must start with 'Bearer '")
				return false
			}
		}
	}
	// TODO j.Cookie, j.PostParam, j.QueryParam
	if tokenValue == "" {
		j.log.Error("token is empty")
		return false
	}
	token, err := j.parser.Parse(tokenValue, func(token *jwt.Token) (interface{}, error) {
		return j.key, nil
	})
	if err != nil {
		j.log.Error(err)
		return false
	}
	j.log.WithField("token", token).Debug()
	return true
}
