package config

import (
	"errors"
	"strings"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
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
	pubkey             *rsa.PublicKey
	hmacSecret         []byte
}

// TODO read values from Env

func (j *Jwt) Init(log *logrus.Entry) {
	j.log = log
	if j.KeyFile != "" {
		pem, err := ioutil.ReadFile(j.KeyFile)
		if err != nil {
			log.Fatal(err)
		}
		j.pubkey, err = jwt.ParseRSAPublicKeyFromPEM([]byte(pem))
		if err != nil {
			log.Fatal(err)
		}
	} else if j.Key != "" {
		if strings.HasPrefix(j.SignatureAlgorithm, "RS") {
			// x5c from JWK
			derBytesCert, err := base64.StdEncoding.DecodeString(j.Key)
			if err != nil {
				log.Fatal(err)
			}
			cert, err := x509.ParseCertificate(derBytesCert)
			if err != nil {
				log.Fatal(err)
			}
			rsaPublickey, _ := cert.PublicKey.(*rsa.PublicKey)
			j.pubkey = &rsa.PublicKey{N: rsaPublickey.N, E: rsaPublickey.E}
		} else if strings.HasPrefix(j.SignatureAlgorithm, "HS") {
			j.hmacSecret = []byte(j.Key)
		} else {
			log.Fatal("only RSA and HMAC are supported")
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
	if j.Cookie != "" {
		cookie, err := req.Cookie(j.Cookie)
		if err != nil {
			j.log.Error(err)
		}
		tokenValue = cookie.Value
	} else if j.Header != "" {
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
	// TODO j.PostParam, j.QueryParam
	if tokenValue == "" {
		j.log.Error("token is empty")
		return false
	}
	token, err := j.parser.Parse(tokenValue, func(token *jwt.Token) (interface{}, error) {
		if strings.HasPrefix(j.SignatureAlgorithm, "RS") {
			return j.pubkey, nil
		} else if strings.HasPrefix(j.SignatureAlgorithm, "HS") {
			return j.hmacSecret, nil
		}
		return nil, errors.New("only RSA and HMAC are supported")
	})
	if err != nil {
		j.log.Error(err)
		return false
	}
	j.log.WithField("token", token).Debug()
	return true
}
