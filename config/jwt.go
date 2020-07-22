package config

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go/v4"
	"github.com/sirupsen/logrus"
)

type Claims struct {
	Issuer   string `hcl:"iss,optional"`
	Audience string `hcl:"aud,optional"`
}

type JWT struct {
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

var (
	ErrorBearerRequired = errors.New("authorization header value must start with 'Bearer '")
	ErrorEmptyToken     = errors.New("empty token")
	ErrorMissingKey     = errors.New("either key_file or key must be specified")
	ErrorNotSupported   = errors.New("only RSA and HMAC are supported")
)

// TODO read values from Env

func (j *JWT) Init(log *logrus.Entry) {
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
			panic(ErrorNotSupported)
		}
	} else {
		panic(ErrorMissingKey)
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

func (j *JWT) Check(req *http.Request) error {
	var tokenValue string
	var err error

	if j.Cookie != "" {
		cookie, err := req.Cookie(j.Cookie)
		if err != nil {
			return err
		}
		tokenValue = cookie.Value
	} else if j.Header != "" && j.Header == "Autorization" {
		if tokenValue, err = getBearer(req.Header.Get(j.Header)); err != nil {
			return err
		}
	}

	// TODO j.PostParam, j.QueryParam
	if tokenValue == "" {
		return ErrorEmptyToken
	}

	token, err := j.parser.Parse(tokenValue, func(token *jwt.Token) (interface{}, error) {
		if strings.HasPrefix(j.SignatureAlgorithm, "RS") {
			return j.pubkey, nil
		} else if strings.HasPrefix(j.SignatureAlgorithm, "HS") {
			return j.hmacSecret, nil
		}
		return nil, ErrorNotSupported
	})
	if err != nil {
		return err
	}
	j.log.WithField("token", token).Debug()
	return nil
}

func getBearer(val string) (string, error) {
	const bearer = "bearer "
	if strings.HasPrefix(strings.ToLower(val), bearer) {
		return strings.Trim(val[len(bearer):], " "), nil
	}
	return "", ErrorBearerRequired
}
