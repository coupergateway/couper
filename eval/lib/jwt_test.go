package lib_test

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function/stdlib"

	"github.com/avenga/couper/config/configload"
)

func TestJwtSignStatic(t *testing.T) {
	tests := []struct {
		name      string
		hcl       string
		jspLabel  string
		claims    string
		want      string
	}{
		{
			"HS256 / key",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "HS256"
					key = "$3cRe4"
					ttl = "0"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"MyToken",
			`{"sub":"12345"}`,
			"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJUSEVfQVVESUVOQ0UiLCJpc3MiOiJ0aGVfaXNzdWVyIiwic3ViIjoiMTIzNDUifQ.Hf-ZtIlsxR2bDOdAEMaDHaOBmfVWTQi9U68yV4YHW9w",
		},
		{
			"HS256 / key_file",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "HS256"
					key_file = "testdata/secret.txt"
					ttl = "0"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"MyToken",
			`{"sub":"12345"}`,
			"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJUSEVfQVVESUVOQ0UiLCJpc3MiOiJ0aGVfaXNzdWVyIiwic3ViIjoiMTIzNDUifQ.Hf-ZtIlsxR2bDOdAEMaDHaOBmfVWTQi9U68yV4YHW9w",
		},
		{
			"HS384 / key",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "HS384"
					key = "$3cRe4"
					ttl = "0"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"MyToken",
			`{"sub":"12345"}`,
			"eyJhbGciOiJIUzM4NCIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJUSEVfQVVESUVOQ0UiLCJpc3MiOiJ0aGVfaXNzdWVyIiwic3ViIjoiMTIzNDUifQ.fYh9VOfe9jv926lwyjNMHr98zqesELs-2v0feqqDByor7rlnHHUdWdZXkTcPRaDa",
		},
		{
			"HS384 / key_file",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "HS384"
					key_file = "testdata/secret.txt"
					ttl = "0"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"MyToken",
			`{"sub":"12345"}`,
			"eyJhbGciOiJIUzM4NCIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJUSEVfQVVESUVOQ0UiLCJpc3MiOiJ0aGVfaXNzdWVyIiwic3ViIjoiMTIzNDUifQ.fYh9VOfe9jv926lwyjNMHr98zqesELs-2v0feqqDByor7rlnHHUdWdZXkTcPRaDa",
		},
		{
			"HS512 / key",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "HS512"
					key = "$3cRe4"
					ttl = "0"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"MyToken",
			`{"sub":"12345"}`,
			"eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJUSEVfQVVESUVOQ0UiLCJpc3MiOiJ0aGVfaXNzdWVyIiwic3ViIjoiMTIzNDUifQ.LCzGZMYxwLAra2tHNDFBCSKVzMdZeZGxhgGuVr0e9mHDXMqpyOiCBWN2JB-9aDUNPHobwxEWofPY8M9icL3YXQ",
		},
		{
			"HS512 / key_file",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "HS512"
					key_file = "testdata/secret.txt"
					ttl = "0"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"MyToken",
			`{"sub":"12345"}`,
			"eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJUSEVfQVVESUVOQ0UiLCJpc3MiOiJ0aGVfaXNzdWVyIiwic3ViIjoiMTIzNDUifQ.LCzGZMYxwLAra2tHNDFBCSKVzMdZeZGxhgGuVr0e9mHDXMqpyOiCBWN2JB-9aDUNPHobwxEWofPY8M9icL3YXQ",
		},
		{
			"RS256 / key",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "RS256"
					key = <<EOF
-----BEGIN RSA PRIVATE KEY-----
MIICWwIBAAKBgQDGSd+sSTss2uOuVJKpumpFAamlt1CWLMTAZNAabF71Ur0P6u83
3RhAIjXDSA/QeVitzvqvCZpNtbOJVegaREqLMJqvFOUkFdLNRP3f9XjYFFvubo09
tcjX6oGEREKDqLG2MfZ2Z8LVzuJc6SwZMgVFk/63rdAOci3W9u3zOSGj4QIDAQAB
AoGAMzI1rw0FW1J0wLkTWQFJmOGSBLhs9Sk/75DX7kqWxe6D5A07kIfkUALFMNN1
SdVa4R10uibXkULdxRLKJ6YEPLGAN3UmdbnBGxZ+fHAKY3PxM5lL9d7ET08A0u/8
6vB+GZ8w0eqsp4EFzmXI5LS63cRo9GA5iliGpKWtd2IUA2UCQQDnZHJTHW21vrXv
GqXoPxOoQAflxvnHYDgNQcRJxlEokFmSK405n7G2//NrsSnXYmUsA/wdh9YsAYZ3
4xy6hKE3AkEA22Aw58FnypcRAKBTqEWHv957szAmz9R6mLJqG7283YWXL0VGDOuR
qdC4QjMrix3O8WbJxGNaVCrvYKVtKEfPpwJAGGWw4C6UKLuI90L6BzjPW8gUjRej
sm/kuREcHyM3320I5K6O32qFFGR8R/iQDtOjEzcAWCTAYjdu9CkQGGJvlQJAHpCR
X8jfmCdiFA9CeKBvYHk0DOw5jB1Tk3DQPds6tDaHsOta7jPoEJvnADo25+QYUCP9
GqKpFC8DORjzU3hl4wJACEzmqzAco2M4mVc+PxPX0b3LHaREyXURd+faFXUecxSF
BShcGHZl9nzWDtEZzgdX7cbG5nRUo1+whzBQdYoQmg==
-----END RSA PRIVATE KEY-----
					EOF
					ttl = "0"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"MyToken",
			`{"sub":"12345"}`,
			"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJUSEVfQVVESUVOQ0UiLCJpc3MiOiJ0aGVfaXNzdWVyIiwic3ViIjoiMTIzNDUifQ.oSS8rC1KonyZ-JZTZhkqZb5bN0_2Lrbl4J33nLgWroc5vDvmLW0KnX0RQfXy0OjX4uBBYTThActqqqM6vidaXmBfsQ77uB9narWeAptRnKqEPlY-onTHDmTMCz7vQ9wbLT7Aa6MYlhRqKX5adpPPbwBUuhm2I-yMF80nSmFpSk0",
		},
		{
			"RS256 / key_file",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "RS256"
					key_file = "testdata/rsa_priv.pem"
					ttl = "0"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"MyToken",
			`{"sub":"12345"}`,
			"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJUSEVfQVVESUVOQ0UiLCJpc3MiOiJ0aGVfaXNzdWVyIiwic3ViIjoiMTIzNDUifQ.oSS8rC1KonyZ-JZTZhkqZb5bN0_2Lrbl4J33nLgWroc5vDvmLW0KnX0RQfXy0OjX4uBBYTThActqqqM6vidaXmBfsQ77uB9narWeAptRnKqEPlY-onTHDmTMCz7vQ9wbLT7Aa6MYlhRqKX5adpPPbwBUuhm2I-yMF80nSmFpSk0",
		},
		{
			"RS384 / key",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "RS384"
					key = <<EOF
-----BEGIN RSA PRIVATE KEY-----
MIICWwIBAAKBgQDGSd+sSTss2uOuVJKpumpFAamlt1CWLMTAZNAabF71Ur0P6u83
3RhAIjXDSA/QeVitzvqvCZpNtbOJVegaREqLMJqvFOUkFdLNRP3f9XjYFFvubo09
tcjX6oGEREKDqLG2MfZ2Z8LVzuJc6SwZMgVFk/63rdAOci3W9u3zOSGj4QIDAQAB
AoGAMzI1rw0FW1J0wLkTWQFJmOGSBLhs9Sk/75DX7kqWxe6D5A07kIfkUALFMNN1
SdVa4R10uibXkULdxRLKJ6YEPLGAN3UmdbnBGxZ+fHAKY3PxM5lL9d7ET08A0u/8
6vB+GZ8w0eqsp4EFzmXI5LS63cRo9GA5iliGpKWtd2IUA2UCQQDnZHJTHW21vrXv
GqXoPxOoQAflxvnHYDgNQcRJxlEokFmSK405n7G2//NrsSnXYmUsA/wdh9YsAYZ3
4xy6hKE3AkEA22Aw58FnypcRAKBTqEWHv957szAmz9R6mLJqG7283YWXL0VGDOuR
qdC4QjMrix3O8WbJxGNaVCrvYKVtKEfPpwJAGGWw4C6UKLuI90L6BzjPW8gUjRej
sm/kuREcHyM3320I5K6O32qFFGR8R/iQDtOjEzcAWCTAYjdu9CkQGGJvlQJAHpCR
X8jfmCdiFA9CeKBvYHk0DOw5jB1Tk3DQPds6tDaHsOta7jPoEJvnADo25+QYUCP9
GqKpFC8DORjzU3hl4wJACEzmqzAco2M4mVc+PxPX0b3LHaREyXURd+faFXUecxSF
BShcGHZl9nzWDtEZzgdX7cbG5nRUo1+whzBQdYoQmg==
-----END RSA PRIVATE KEY-----
					EOF
					ttl = "0"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"MyToken",
			`{"sub":"12345"}`,
			"eyJhbGciOiJSUzM4NCIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJUSEVfQVVESUVOQ0UiLCJpc3MiOiJ0aGVfaXNzdWVyIiwic3ViIjoiMTIzNDUifQ.GaKMl5KxXXTlOUgfZy-Cs81jvVhp-2TjEZIg58qnjIbHH7P0YtIr8p4ikTHQhie7oXs5iwzQPdPqwJSUYHlia3t118mv1ie0IWjqmXhOcWsEODQYHfshIfKaqfIZaF7WFBZXfNdXX4g-8_aUrNPevZ_6vVhHq2844cjaKH-XGl4",
		},
		{
			"RS384 / key_file",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "RS384"
					key_file = "testdata/rsa_priv.pem"
					ttl = "0"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"MyToken",
			`{"sub":"12345"}`,
			"eyJhbGciOiJSUzM4NCIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJUSEVfQVVESUVOQ0UiLCJpc3MiOiJ0aGVfaXNzdWVyIiwic3ViIjoiMTIzNDUifQ.GaKMl5KxXXTlOUgfZy-Cs81jvVhp-2TjEZIg58qnjIbHH7P0YtIr8p4ikTHQhie7oXs5iwzQPdPqwJSUYHlia3t118mv1ie0IWjqmXhOcWsEODQYHfshIfKaqfIZaF7WFBZXfNdXX4g-8_aUrNPevZ_6vVhHq2844cjaKH-XGl4",
		},
		{
			"RS384 / key",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "RS512"
					key = <<EOF
-----BEGIN RSA PRIVATE KEY-----
MIICWwIBAAKBgQDGSd+sSTss2uOuVJKpumpFAamlt1CWLMTAZNAabF71Ur0P6u83
3RhAIjXDSA/QeVitzvqvCZpNtbOJVegaREqLMJqvFOUkFdLNRP3f9XjYFFvubo09
tcjX6oGEREKDqLG2MfZ2Z8LVzuJc6SwZMgVFk/63rdAOci3W9u3zOSGj4QIDAQAB
AoGAMzI1rw0FW1J0wLkTWQFJmOGSBLhs9Sk/75DX7kqWxe6D5A07kIfkUALFMNN1
SdVa4R10uibXkULdxRLKJ6YEPLGAN3UmdbnBGxZ+fHAKY3PxM5lL9d7ET08A0u/8
6vB+GZ8w0eqsp4EFzmXI5LS63cRo9GA5iliGpKWtd2IUA2UCQQDnZHJTHW21vrXv
GqXoPxOoQAflxvnHYDgNQcRJxlEokFmSK405n7G2//NrsSnXYmUsA/wdh9YsAYZ3
4xy6hKE3AkEA22Aw58FnypcRAKBTqEWHv957szAmz9R6mLJqG7283YWXL0VGDOuR
qdC4QjMrix3O8WbJxGNaVCrvYKVtKEfPpwJAGGWw4C6UKLuI90L6BzjPW8gUjRej
sm/kuREcHyM3320I5K6O32qFFGR8R/iQDtOjEzcAWCTAYjdu9CkQGGJvlQJAHpCR
X8jfmCdiFA9CeKBvYHk0DOw5jB1Tk3DQPds6tDaHsOta7jPoEJvnADo25+QYUCP9
GqKpFC8DORjzU3hl4wJACEzmqzAco2M4mVc+PxPX0b3LHaREyXURd+faFXUecxSF
BShcGHZl9nzWDtEZzgdX7cbG5nRUo1+whzBQdYoQmg==
-----END RSA PRIVATE KEY-----
					EOF
					ttl = "0"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"MyToken",
			`{"sub":"12345"}`,
			"eyJhbGciOiJSUzUxMiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJUSEVfQVVESUVOQ0UiLCJpc3MiOiJ0aGVfaXNzdWVyIiwic3ViIjoiMTIzNDUifQ.flU1adXUEaZuqkFwhcgJ8U3OXYOTC6RQCWw9rb7nkTNzt7XrU13EPtlxH5_7lpAvyBn4iyOCiJd19y1paupyeYbHEgUGsVXa4Iu1jQ8I7C41ejLNybdg7XpRzf3zt6tMC3W9Bp0TYRqrykTiQ0W4pg0sGJCV-e30dSDgkfuS_TM",
		},
		{
			"RS384 / key_file",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "RS512"
					key_file = "testdata/rsa_priv.pem"
					ttl = "0"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"MyToken",
			`{"sub":"12345"}`,
			"eyJhbGciOiJSUzUxMiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJUSEVfQVVESUVOQ0UiLCJpc3MiOiJ0aGVfaXNzdWVyIiwic3ViIjoiMTIzNDUifQ.flU1adXUEaZuqkFwhcgJ8U3OXYOTC6RQCWw9rb7nkTNzt7XrU13EPtlxH5_7lpAvyBn4iyOCiJd19y1paupyeYbHEgUGsVXa4Iu1jQ8I7C41ejLNybdg7XpRzf3zt6tMC3W9Bp0TYRqrykTiQ0W4pg0sGJCV-e30dSDgkfuS_TM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cf, err := configload.LoadBytes([]byte(tt.hcl), "couper.hcl")
			if err != nil {
				t.Fatal(err)
			}
			claims, err := stdlib.JSONDecode(cty.StringVal(tt.claims))
			if err != nil {
				t.Fatal(err)
			}
			token, err := cf.Context.Functions["jwt_sign"].Call([]cty.Value{cty.StringVal(tt.jspLabel), claims})
			if err != nil {
				t.Fatal(err)
			}
			if token.AsString() != tt.want {
				t.Errorf("Expected %q, got: %#v", tt.want, token.AsString())
			}
		})
	}
}

func TestJwtSignDynamic(t *testing.T) {
	tests := []struct {
		name      string
		hcl       string
		jspLabel  string
		claims    string
		wantTTL   int64
	}{
		{
			"ttl 1h",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "HS256"
					key = "$3cRe4"
					ttl = "1h"
					claims = {
						exp = 1234567890
					}
				}
			}
			`,
			"MyToken",
			`{"sub": "12345"}`,
			3600,
		},
		{
			"ttl 60.6s",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "HS256"
					key = "$3cRe4"
					ttl = "60.6s"
				}
			}
			`,
			"MyToken",
			`{"sub": "12345"}`,
			60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cf, err := configload.LoadBytes([]byte(tt.hcl), "couper.hcl")
			if err != nil {
				t.Fatal(err)
			}
			claims, err := stdlib.JSONDecode(cty.StringVal(tt.claims))
			if err != nil {
				t.Fatal(err)
			}
			now := time.Now().Unix()
			token, err := cf.Context.Functions["jwt_sign"].Call([]cty.Value{cty.StringVal(tt.jspLabel), claims})
			if err != nil {
				t.Fatal(err)
			}
			tokenParts := strings.Split(token.AsString(), ".")
			if len(tokenParts) != 3 {
				t.Errorf("Needs 3 parts, got: %d", len(tokenParts))
			}
			body, err := base64.RawURLEncoding.DecodeString(tokenParts[1])
			if err != nil {
				t.Fatal(err)
			}
			var resultClaims map[string]interface{}
			err = json.Unmarshal(body, &resultClaims)
			if err != nil {
				t.Fatal(err)
			}
			if resultClaims["exp"] == nil {
				t.Errorf("Expected exp claim, got: %#v", body)
			}
			exp := resultClaims["exp"].(float64)
			if int64(exp) - now != tt.wantTTL {
				t.Errorf(string(body))
				t.Errorf("Expected %d, got: %d", tt.wantTTL, int64(exp) - now)
			}
		})
	}
}

func TestJwtSignError(t *testing.T) {
	tests := []struct {
		name      string
		hcl       string
		jspLabel  string
		claims    string
		wantErr   string
	}{
		{
			"No profile for label",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "HS256"
					key = "$3cRe4"
					ttl = "0"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"NoProfileForThisLabel",
			`{"sub":"12345"}`,
			"no signing profile for label",
		},
		{
			"Missing file for key_file",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "HS256"
					key_file = "not_there.txt"
					ttl = "0"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"MyToken",
			`{"sub":"12345"}`,
			"no such file or directory",
		},
		{
			"Missing key and key_file",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "HS256"
					ttl = "0"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"MyToken",
			`{"sub":"12345"}`,
			"either key_file or key must be specified",
		},
		{
			"Invalid ttl value",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "HS256"
					key = "$3cRe4"
					ttl = "invalid"
				}
			}
			`,
			"MyToken",
			`{"sub": "12345"}`,
			"time: invalid duration ",
		},
		{
			"argument claims no object",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "HS256"
					key = "$3cRe4"
					ttl = "0"
				}
			}
			`,
			"MyToken",
			`"no object"`,
			"json: cannot unmarshal string into Go value of type map[string]interface {}",
		},
		{
			"unsupported signature algorithm",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "invalid"
					key = "$3cRe4"
					ttl = "0"
				}
			}
			`,
			"MyToken",
			`{"sub": "12345"}`,
			"unsupported signing method",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cf, err := configload.LoadBytes([]byte(tt.hcl), "couper.hcl")
			if err != nil {
				t.Fatal(err)
			}
			claims, err := stdlib.JSONDecode(cty.StringVal(tt.claims))
			if err != nil {
				t.Fatal(err)
			}
			_, err = cf.Context.Functions["jwt_sign"].Call([]cty.Value{cty.StringVal(tt.jspLabel), claims})
			if err == nil {
				t.Fatal(err)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Expected %q, got: %#v", tt.wantErr, err.Error())
			}
		})
	}
}
